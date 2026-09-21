#!/usr/bin/env bash
# Export PingOne regression environment content.
#
# The committed config in this directory is the source of truth for the
# managed environment. This script runs a full export — .tf config, the
# tfvars values file, and the state-adoption import file (all from one
# export run) — with safeguards so user-maintained state survives:
#
#   Config (.tf)   — the tracked files are backed up to .config-backup/
#                    before being replaced; --revert restores them. Review
#                    the git diff before committing: the raw export carries
#                    embedded secrets from plain variables and lacks
#                    hand-curated lifecycle blocks.
#
#   tfvars         — variable values are git-ignored and ride in the
#                    TERRAFORM_TFVARS_BASE64 GitHub secret (and 1Password).
#                    Variable names are compared against the existing file:
#                      • identical variable set  → nothing copied; your
#                        reviewed values file stays as-is (no churn, no
#                        re-review needed)
#                      • new variables present   → the existing file is
#                        backed up (ping-export-terraform.auto.tfvars.bak,
#                        next to the new file) and the fresh export replaces
#                        it, so you can diff the two and port values over.
#
#   imports.tf     — ping-export-imports.tf is committed. Its import blocks
#                    auto-import resources absent from state on the next
#                    apply and are inert no-ops for resources already
#                    managed — so a rebuilt environment self-imports.
#
# Usage:
#   ./export.sh                # back up config + tfvars, then replace both
#                              #  with a fresh full export (imports included)
#   ./export.sh --revert       # restore the config backup made by the last export
#   ./export.sh --upload-only  # upload the reviewed tfvars as TERRAFORM_TFVARS_BASE64
#                              #  (export and upload are always separate steps: always
#                              #  review the generated values before uploading)
#   ./export.sh --help
#
# Credentials: reads the standard PINGCLI_PINGONE_* environment variables.
# With 1Password CLI, prefix the whole script:
#   op run --env-file=<your op env file> -- ./export.sh
# The exported values file contains the live environment's variable values —
# treat it as a secret. It is git-ignored here (see .gitignore).
# Upload requires a gh account with repository-secret write access.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
ENV_DIR="tests/regression/environment"
TFVARS_FILE="${ENV_DIR}/ping-export-terraform.auto.tfvars"
BACKUP="${ENV_DIR}/.config-backup"

UPLOAD_ONLY=0
REVERT=0
for arg in "$@"; do
  case "${arg}" in
    --revert) REVERT=1 ;;
    --upload-only) UPLOAD_ONLY=1 ;;
    -h|--help)
      awk 'NR>1 && /^#/{print} NR>1 && !/^#/{exit}' "$0"; exit 0 ;;
    *)
      printf 'unknown argument: %s\n' "${arg}" >&2; exit 2 ;;
  esac
done

upload_tfvars() {
  if [ ! -f "${TFVARS_FILE}" ]; then
    printf 'no tfvars file at %s — run an export first\n' "${TFVARS_FILE}" >&2
    exit 1
  fi
  command -v gh >/dev/null 2>&1 || { echo "gh CLI is required to upload" >&2; exit 1; }
  _repo="$(gh repo view --json nameWithOwner -q .nameWithOwner)"
  echo "==> uploading $(basename "${TFVARS_FILE}") as TERRAFORM_TFVARS_BASE64 to ${_repo}"
  base64 < "${TFVARS_FILE}" | gh secret set TERRAFORM_TFVARS_BASE64 --repo "${_repo}"
  echo "uploaded TERRAFORM_TFVARS_BASE64 to ${_repo}"
}

# --upload-only skips the export entirely (no PingOne credentials needed).
if [ "${UPLOAD_ONLY}" -eq 1 ]; then
  upload_tfvars
  exit 0
fi

# --revert restores the config backup from the last export.
if [ "${REVERT:-0}" -eq 1 ]; then
  if [ ! -f "${BACKUP}/manifest" ]; then
    printf 'no config backup found at %s — nothing to revert\n' "${BACKUP}" >&2
    exit 1
  fi
  echo "==> reverting config to backup from: $(cat "${BACKUP}/manifest")"
  (cd "${BACKUP}/config" && tar cf - .) | (cd "${ENV_DIR}" && tar xf -)
  rm -rf "${BACKUP}"
  echo "config restored; backup removed."
  echo "note: tfvars was NOT restored (it is not part of the committed config)."
  exit 0
fi

for var in \
  PINGCLI_PINGONE_ENVIRONMENT_ID \
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID \
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET; do
  [ -n "${!var:-}" ] || { printf 'required env var not set: %s\n' "${var}" >&2; exit 1; }
done
: "${PINGCLI_PINGONE_REGION_CODE:=NA}"
: "${PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID:=${PINGCLI_PINGONE_ENVIRONMENT_ID}}"
export PINGCLI_PINGONE_REGION_CODE PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID

command -v terraform >/dev/null 2>&1 || { echo "terraform is required" >&2; exit 1; }
BIN="./pingcli-terraformer"
[ -x "${BIN}" ] || { echo "building ${BIN}..."; make build >/dev/null; }

# Back up the committed config (tracked files only) before replacing it.
# Tar of git-tracked files = exactly what --revert restores.
mkdir -p "${BACKUP}/config"
(cd "${ENV_DIR}" && git ls-files . | tar cf - -T -) | tar xf - -C "${BACKUP}/config"
git -C . rev-parse HEAD > "${BACKUP}/manifest"
echo "==> backed up committed config (revert with: ./export.sh --revert)"

# Back up the tfvars: the export overwrites it, and its values are
# user-maintained. If the fresh export has the same variable set, the backup
# is simply restored below; otherwise diff the two and port values.
TFVARS_UNCHANGED=0
if [ -f "${TFVARS_FILE}" ]; then
  cp "${TFVARS_FILE}" "${TFVARS_FILE}.bak"
  echo "==> backed up existing tfvars to ping-export-terraform.auto.tfvars.bak"
fi

# One export run produces the full config, tfvars, and the state-adoption
# import file (--include-imports emits ping-export-imports.tf alongside).
echo "==> exporting full config to ${ENV_DIR}"
mkdir -p "${ENV_DIR}"
"${BIN}" export \
  --output-format hcl \
  --include-values \
  --include-imports \
  --out "${ENV_DIR}"

# If the variable set did not change, restore the reviewed tfvars — a
# same-set re-export must not clobber user-maintained values.
_generated="${TFVARS_FILE}"
var_names() { sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)[[:space:]]*=.*/\1/p' "$1" | sort -u; }
if [ -f "${TFVARS_FILE}.bak" ]; then
  _new_vars="$(comm -13 <(var_names "${TFVARS_FILE}.bak") <(var_names "${_generated}"))"
  _gone_vars="$(comm -23 <(var_names "${TFVARS_FILE}.bak") <(var_names "${_generated}"))"
  if [ -z "${_new_vars}" ] && [ -z "${_gone_vars}" ]; then
    mv "${TFVARS_FILE}.bak" "${TFVARS_FILE}"
    echo "==> tfvars variable set unchanged — reviewed values restored ($(var_names "${TFVARS_FILE}" | wc -l | tr -d ' ') variables)"
  elif [ -z "${_new_vars}" ]; then
    mv "${TFVARS_FILE}.bak" "${TFVARS_FILE}"
    echo "==> $(echo "${_gone_vars}" | wc -l | tr -d ' ') variable(s) no longer generated (kept in your file as-is):"
    echo "${_gone_vars}" | sed 's/^/  - /'
    echo "    tfvars values preserved"
  else
    echo "==> generated tfvars contains $(echo "${_new_vars}" | wc -l | tr -d ' ') new variable(s):"
    echo "${_new_vars}" | sed 's/^/  + /'
    [ -n "${_gone_vars}" ] && {
      echo "    and $(echo "${_gone_vars}" | wc -l | tr -d ' ') variable(s) no longer generated:"
      echo "${_gone_vars}" | sed 's/^/  - /'
    }
    echo "    old values kept in ping-export-terraform.auto.tfvars.bak — diff the"
    echo "    two, port your reviewed values, then upload with: ./export.sh --upload-only"
  fi
fi

echo
echo "wrote fresh full config into ${ENV_DIR} (git will show the diff)"
echo "wrote ${ENV_DIR}/ping-export-imports.tf (committed — its import blocks"
echo "  auto-import resources absent from state and are no-ops otherwise)"
echo "  -> review before committing: plain DaVinci variables holding passwords"
echo "     (e.g. testPW) export verbatim in BOTH the .tf graph_data and tfvars;"
echo "     secret values export as empty strings marked"
echo "     \"Secret value - provide manually\". There is no --upload that skips"
echo "     this step on purpose."
echo "  -> revert config with: ./export.sh --revert"
