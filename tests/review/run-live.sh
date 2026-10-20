#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Live review-bundle runner (make review-bundle-live).
#
# Provisions a throwaway PingOne environment via tools/tf-regression-provision
# (the review tool itself never provisions), applies the
# terraform-test-data/root fixtures into it, then runs tools/review-bundle
# against that fresh environment with the five PINGCLI_PINGONE_* worker
# credentials parsed from the provision tool's stdout.
#
# The environment is KEPT after the run by default: the point of the review
# bundle is "a live environment still exists that contains the change", so a
# reviewer can inspect it. REVIEW_DESTROY=1 opts into tearing it down at exit.
# ──────────────────────────────────────────────────────────────────────────────

info() { printf '[INFO]  %s\n' "$*"; }
success() { printf '[PASS]  %s\n' "$*"; }
fail() { printf '[FAIL]  %s\n' "$*" >&2; }
die() { fail "$*"; exit 1; }

check_prerequisites() {
  local missing=0

  for var in \
    PINGCLI_PINGONE_ORGADMIN_CLIENT_ID \
    PINGCLI_PINGONE_ORGADMIN_CLIENT_SECRET \
    PINGCLI_PINGONE_ORGADMIN_ENVIRONMENT_ID \
    PINGCLI_PINGONE_ORGADMIN_REGION_CODE \
    PINGCLI_PINGONE_ORGADMIN_LICENSE_ID; do
    if [ -z "${!var:-}" ]; then
      fail "Required environment variable not set: ${var}"
      missing=1
    fi
  done

  for tool in jq terraform; do
    if ! command -v "${tool}" &>/dev/null; then
      fail "Required tool not found: ${tool}"
      missing=1
    fi
  done

  [ "$missing" -eq 0 ] || exit 1
}

apply_defaults() {
  : "${REVIEW_BASE:=main}"
  : "${REVIEW_DESTROY:=0}"
  : "${E2E_KEEP_ENVIRONMENT:=0}"

  export REVIEW_BASE REVIEW_DESTROY E2E_KEEP_ENVIRONMENT
}

REPO_ROOT=""
TMPDIR_LOCAL=""
TF_DIR=""
PROVISIONED_ENV_ID=""

setup_dirs() {
  REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  if [ ! -f "${REPO_ROOT}/Makefile" ]; then
    die "Could not locate repo root (Makefile not found at ${REPO_ROOT})"
  fi

  TMPDIR_LOCAL="$(mktemp -d "${TMPDIR:-/tmp}/pingcli-review.XXXXXX")"
  TF_DIR="${REPO_ROOT}/terraform-test-data/root"

  info "Repo root  : ${REPO_ROOT}"
  info "Temp dir   : ${TMPDIR_LOCAL}"
  info "Base ref   : ${REVIEW_BASE}"
}

# ---------------------------------------------------------------------------
# Cleanup trap. The provisioned environment is kept by default (REVIEW_DESTROY
# unset/0); with REVIEW_DESTROY=1 it is torn down. Trigger condition for the
# destroy path is the presence of terraform.tfvars.json in TF_DIR, NOT
# PROVISIONED_ENV_ID: terraform apply can create the environment and then fail
# on a later fixture, in which case PROVISIONED_ENV_ID is never set but a real
# (now-orphaned) environment must still be torn down.
# ---------------------------------------------------------------------------
cleanup() {
  if [ "${REVIEW_DESTROY}" = "1" ] && [ -f "${TF_DIR}/terraform.tfvars.json" ]; then
    info "Tearing down provisioned environment${PROVISIONED_ENV_ID:+ ${PROVISIONED_ENV_ID}}..."
    "${TMPDIR_LOCAL}/tf-regression-provision" \
      --action destroy \
      --terraform-dir "${TF_DIR}" \
      || fail "Teardown reported an error - verify manually in the PingOne admin console."
  elif [ -f "${TF_DIR}/terraform.tfvars.json" ]; then
    info ""
    success "Provisioned environment ${PROVISIONED_ENV_ID} left running for review."
    info "Destroy it when review completes:"
    info "  cd ${TF_DIR} && terraform destroy"
    info "(or re-run with REVIEW_DESTROY=1; E2E_KEEP_ENVIRONMENT=1 only affects"
    info " the checklist wording, this script never auto-destroys)"
  fi

  if [ -n "${TMPDIR_LOCAL}" ] && [ -d "${TMPDIR_LOCAL}" ]; then
    # The provision result JSON (contains the worker client secret) is deleted
    # by provision() immediately after parsing; TMPDIR_LOCAL holds only
    # binaries at this point.
    rm -rf "${TMPDIR_LOCAL}"
  fi
}

# ---------------------------------------------------------------------------
# Provision a throwaway environment and apply terraform-test-data/root.
# Populates the five PINGCLI_PINGONE_* env vars from the tool's stdout JSON -
# never echoed, parsed with jq, and the result file is deleted immediately.
# ---------------------------------------------------------------------------
provision() {
  info "Building tf-regression-provision..."
  (cd "${REPO_ROOT}" && go build -o "${TMPDIR_LOCAL}/tf-regression-provision" ./tools/tf-regression-provision/)

  info "Provisioning throwaway PingOne environment (this can take a minute)..."
  local result="${TMPDIR_LOCAL}/provision-result.json"
  "${TMPDIR_LOCAL}/tf-regression-provision" \
    --action create \
    --terraform-dir "${TF_DIR}" \
    --name-prefix "pingcli-review-bundle" \
    >"${result}"

  PROVISIONED_ENV_ID=$(jq -r '.target_environment_id' "${result}")
  # Auth environment (where the OAuth token is acquired) is distinct from the
  # export target (the throwaway environment just created) - mirrors
  # internal/platform/pingone.NewFromCredentials's workerEnvID vs exportEnvID
  # split.
  PINGCLI_PINGONE_ENVIRONMENT_ID=$(jq -r '.auth_environment_id' "${result}")
  PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID="${PROVISIONED_ENV_ID}"
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID=$(jq -r '.client_id' "${result}")
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET=$(jq -r '.client_secret' "${result}")
  PINGCLI_PINGONE_REGION_CODE=$(jq -r '.region_code' "${result}")
  export PINGCLI_PINGONE_ENVIRONMENT_ID PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID
  export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET
  export PINGCLI_PINGONE_REGION_CODE
  rm -f "${result}"

  success "Provisioned environment ${PROVISIONED_ENV_ID}."
}

main() {
  check_prerequisites
  apply_defaults
  setup_dirs

  trap cleanup EXIT

  provision

  info "Generating review bundle (base ref: ${REVIEW_BASE})..."
  (cd "${REPO_ROOT}" && go run ./tools/review-bundle/ \
    --base-ref "${REVIEW_BASE}" \
    --out review-bundle)

  success "Review bundle written to ${REPO_ROOT}/review-bundle/"
}

main "$@"
