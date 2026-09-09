#!/usr/bin/env bash
# Export PingOne regression environment content.
#
# The committed config in this directory is hand-curated: the .tf files are
# the source of truth for the managed environment, and the variable values
# (ping-export-terraform.auto.tfvars) are git-ignored and ride in the
# TERRAFORM_TFVARS_BASE64 GitHub secret (and 1Password). This script has two
# distinct jobs:
#
#   Default  — refresh ONLY the tfvars values file from the live environment
#              (full --include-values export to a temp dir; only the tfvars
#              file is copied out). The committed .tf files are untouched.
#              Always review the generated values before uploading.
#
#   --full   — replace the committed .tf config with a fresh export (after
#              backing the current config up), for wholesale baseline
#              refreshes. Review before committing: the raw export carries
#              embedded secrets from plain variables and lacks hand-curated
#              lifecycle blocks. Reversible with --revert.
#
#   --imports — additionally generate the ephemeral state-adoption import
#              file (README.md "Adopting state"). Only valid with --full.
#
# Usage:
#   ./export.sh                # refresh ping-export-terraform.auto.tfvars only
#   ./export.sh --full         # back up current config, then replace the .tf
#                              #  files with a fresh full export
#   ./export.sh --full --imports
#                              # same, plus the one-time state-adoption import file
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
IMPORTS=0
UPLOAD_ONLY=0
REVERT=0
for arg in "$@"; do
  case "${arg}" in
    --full) FULL=1 ;;
    --imports) IMPORTS=1 ;;
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

: "${FULL:=0}"
if [ "${IMPORTS}" -eq 1 ] && [ "${FULL}" -ne 1 ]; then
  printf -- '--imports is only valid with --full\n' >&2
  exit 2
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

  echo "==> exporting full config to ${ENV_DIR}"
  mkdir -p "${ENV_DIR}"
  "${BIN}" export \
    --output-format hcl \
    --include-values \
    --out "${ENV_DIR}"

  echo
  echo "wrote fresh full config into ${ENV_DIR} (git will show the diff)"
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

  mkdir -p "${ENV_DIR}"
  cp "${SCRATCH}/ping-export-terraform.auto.tfvars" "${ENV_DIR}/"

  echo
  echo "wrote ${TFVARS_FILE}"
  echo "  -> review it, then upload with: ./export.sh --upload-only"
  echo
  echo "Review is mandatory: secret values export as empty strings marked"
  echo "\"Secret value - provide manually\", but plain DaVinci variables holding"
  echo "passwords (e.g. testPW) export verbatim — scrub anything you do not want"
  echo "in the CI secret. There is no --upload that skips this step on purpose."
fi

if [ "${IMPORTS}" -eq 1 ] && [ "${FULL}" -eq 1 ]; then
  echo
  echo "==> generating state-adoption import blocks"
  "${BIN}" export \
    --output-format hcl \
    --include-imports \
    --out "${SCRATCH}/imports" \
    --module-name ping-export --module-dir ping-export-module
  cp "${SCRATCH}/imports/ping-export-imports.tf" "${ENV_DIR}/"
  echo "wrote ${ENV_DIR}/ping-export-imports.tf"
  echo "  (git-ignored; delete after 'terraform apply' consumes it — see README.md)"
fi
