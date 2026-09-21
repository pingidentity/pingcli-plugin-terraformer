#!/usr/bin/env bash
# Export PingOne regression environment content.
#
# The committed config in this directory is hand-curated: the .tf files are
# the source of truth for the managed environment, and the variable values
# (ping-export-terraform.auto.tfvars) are git-ignored and ride in the
# TERRAFORM_TFVARS_BASE64 GitHub secret (and 1Password). This script has two
# distinct jobs:
#
#   Default  — compare the freshly generated tfvars variable names against
#              the existing ping-export-terraform.auto.tfvars:
#                • identical variable sets  → nothing copied; your reviewed
#                  values file stays as-is (no churn, no re-review needed).
#                • new variables present    → the existing file is backed up
#                  (ping-export-terraform.auto.tfvars.bak, next to the new
#                  file) and the fresh export replaces it, so you can diff
#                  the two and port your values over.
#              The committed .tf files are never touched.
#              Always review before uploading.
#
#   --full   — replace the committed .tf config with a fresh export (after
#              backing the current config up), for wholesale baseline
#              refreshes. The export also generates the state-adoption
#              import file (ping-export-imports.tf, git-ignored) for the
#              one-time `terraform apply` import (README.md "Adopting
#              state"). Review before committing: the raw export carries
#              embedded secrets from plain variables and lacks hand-curated
#              lifecycle blocks. Reversible with --revert.
#
# Usage:
#   ./export.sh                # compare generated tfvars vs existing; copy only
#                              #  if new variables appeared (backs up the old file)
#   ./export.sh --full         # back up current config, then replace the .tf
#                              #  files with a fresh full export (import
#                              #  file included)
#   ./export.sh --revert       # restore the config backup made by the last --full
#   ./export.sh --upload-only  # upload the reviewed tfvars as TERRAFORM_TFVARS_BASE64
#                              #  (export and upload are always separate steps: always
#                              #  review the generated values before uploading)
#   ./export.sh --help
#
# Credentials: reads the standard PINGCLI_PINGONE_* environment variables.
# With 1Password CLI, prefix the whole script:
#   op run --env-file=<your op env file> -- ./export.sh
# The exported values file contains the live environment's variable values —
# treat it as a secret. Both are git-ignored here (see .gitignore).
# Upload requires a gh account with repository-secret write access.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
ENV_DIR="tests/regression/environment"
TFVARS_FILE="${ENV_DIR}/ping-export-terraform.auto.tfvars"
BACKUP="${ENV_DIR}/.config-backup"
SCRATCH="$(mktemp -d /tmp/regression-env-export.XXXXXX)"
trap 'rm -rf "${SCRATCH}"' EXIT

FULL=0
UPLOAD_ONLY=0
REVERT=0
for arg in "$@"; do
  case "${arg}" in
    --full) FULL=1 ;;
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

# --revert restores the config backup from the last --full.
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

if [ "${FULL}" -eq 1 ]; then
  # Back up the committed config (tracked files only) before replacing it.
  # Tar of git-tracked files = exactly what --revert restores.
  mkdir -p "${BACKUP}/config"
  (cd "${ENV_DIR}" && git ls-files . | tar cf - -T -) | tar xf - -C "${BACKUP}/config"
  git -C . rev-parse HEAD > "${BACKUP}/manifest"
  echo "==> backed up committed config (revert with: ./export.sh --revert)"

  # One export run produces the full config AND the state-adoption import
  # file (--include-imports emits ping-export-imports.tf alongside).
  echo "==> exporting full config to ${ENV_DIR}"
  mkdir -p "${ENV_DIR}"
  "${BIN}" export \
    --output-format hcl \
    --include-values \
    --include-imports \
    --out "${ENV_DIR}"

  echo
  echo "wrote fresh full config into ${ENV_DIR} (git will show the diff)"
  echo "wrote ${ENV_DIR}/ping-export-imports.tf (git-ignored; for the one-time"
  echo "  state adoption — delete after 'terraform apply' consumes it)"
  echo "  -> review before committing: plain DaVinci variables holding passwords"
  echo "     (e.g. testPW) export verbatim in BOTH the .tf graph_data and tfvars;"
  echo "     secret values export as empty strings marked"
  echo "     \"Secret value - provide manually\"."
  echo "  -> revert with: ./export.sh --revert"
else
  echo "==> exporting regression environment to ${SCRATCH}"
  "${BIN}" export \
    --output-format hcl \
    --include-values \
    --out "${SCRATCH}"

  _generated="${SCRATCH}/ping-export-terraform.auto.tfvars"
  var_names() { sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)[[:space:]]*=.*/\1/p' "$1" | sort -u; }

  # Copy only when the generated file introduces variables the existing file
  # lacks — a pure re-export with the same variable set would otherwise
  # clobber user-maintained values and force an unnecessary re-review.
  if [ ! -f "${TFVARS_FILE}" ]; then
    mkdir -p "${ENV_DIR}"
    cp "${_generated}" "${TFVARS_FILE}"
    echo "no existing tfvars — wrote ${TFVARS_FILE}"
  else
    _new_vars="$(comm -13 <(var_names "${TFVARS_FILE}") <(var_names "${_generated}"))"
    _gone_vars="$(comm -23 <(var_names "${TFVARS_FILE}") <(var_names "${_generated}"))"
    if [ -z "${_new_vars}" ]; then
      echo "tfvars variable set unchanged ($(var_names "${_generated}" | wc -l | tr -d ' ') variables; nothing new)"
      [ -n "${_gone_vars}" ] && {
        echo "note: $(echo "${_gone_vars}" | wc -l | tr -d ' ') variable(s) no longer generated (left in your file as-is):"
        echo "${_gone_vars}" | sed 's/^/  - /'
      }
      echo "existing values preserved — nothing copied."
    else
      echo "generated tfvars contains $(echo "${_new_vars}" | wc -l | tr -d ' ') new variable(s):"
      echo "${_new_vars}" | sed 's/^/  + /'
      [ -n "${_gone_vars}" ] && {
        echo "and $(echo "${_gone_vars}" | wc -l | tr -d ' ') variable(s) no longer generated:"
        echo "${_gone_vars}" | sed 's/^/  - /'
      }
      mkdir -p "${ENV_DIR}"
      cp "${TFVARS_FILE}" "${TFVARS_FILE}.bak"
      cp "${_generated}" "${TFVARS_FILE}"
      echo
      echo "old file backed up to ${TFVARS_FILE}.bak (right next to the new one)"
      echo "  -> diff the two, port your reviewed values into the new file, then"
      echo "     upload with: ./export.sh --upload-only"
    fi
    echo
    echo "Review is mandatory: secret values export as empty strings marked"
    echo "\"Secret value - provide manually\", but plain DaVinci variables holding"
    echo "passwords (e.g. testPW) export verbatim — scrub anything you do not want"
    echo "in the CI secret. There is no --upload that skips this step on purpose."
  fi
fi
