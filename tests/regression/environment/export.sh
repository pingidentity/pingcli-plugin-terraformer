#!/usr/bin/env bash
# Export the regression environment's variable values (and, optionally, the
# one-time state-adoption import file) using the read-only credential.
#
# Why this script exists:
#   - The committed config in this directory holds only variable *declarations*.
#     The values (ping-export-terraform.auto.tfvars) are git-ignored and ride
#     in the TERRAFORM_TFVARS_BASE64 GitHub secret (and 1Password). This script
#     generates that file from the live environment, ready to upload as the
#     secret.
#   - It also wraps the state-adoption export (--imports) that produces the
#     ephemeral ping-export-imports.tf file for the one-time `terraform apply`
#     import of the live environment (see README.md "Adopting state").
#
# Usage:
#   ./export.sh                # export tfvars (full --include-values export to a scratch dir,
#                              #  copy ping-export-terraform.auto.tfvars here)
#   ./export.sh --imports      # additionally copy ping-export-imports.tf for state adoption
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
SCRATCH="$(mktemp -d /tmp/regression-env-export.XXXXXX)"
trap 'rm -rf "${SCRATCH}"' EXIT

IMPORTS=0
UPLOAD_ONLY=0
for arg in "$@"; do
  case "${arg}" in
    --imports) IMPORTS=1 ;;
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

if [ "${IMPORTS}" -eq 1 ]; then
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
