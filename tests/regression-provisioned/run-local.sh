#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Provisioned-environment E2E test runner.
#
# Creates a throwaway PingOne environment via tools/tf-regression-provision.
# The environment itself and every fixture inside it are defined as
# Terraform in terraform-test-data/root, all authenticated with a single
# org-admin credential. Once applied, runs the same base-vs-PR export
# comparison as tests/regression/run-local.sh against that fresh environment.
# The environment is always torn down on exit, success or failure.
# ──────────────────────────────────────────────────────────────────────────────

# ---------------------------------------------------------------------------
# Color helpers (gracefully degrade when not a TTY)
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
  RED=$(tput setaf 1 2>/dev/null || printf '')
  GREEN=$(tput setaf 2 2>/dev/null || printf '')
  YELLOW=$(tput setaf 3 2>/dev/null || printf '')
  BOLD=$(tput bold 2>/dev/null || printf '')
  RESET=$(tput sgr0 2>/dev/null || printf '')
else
  RED='' GREEN='' YELLOW='' BOLD='' RESET=''
fi

info()    { printf '%s[INFO]%s  %s\n' "${BOLD}" "${RESET}" "$*"; }
success() { printf '%s[PASS]%s  %s\n' "${GREEN}" "${RESET}" "$*"; }
warn()    { printf '%s[WARN]%s  %s\n' "${YELLOW}" "${RESET}" "$*"; }
fail()    { printf '%s[FAIL]%s  %s\n' "${RED}" "${RESET}" "$*" >&2; }
die()     { fail "$*"; exit 1; }

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
check_prerequisites() {
  local missing=0

  for var in \
    PINGCLI_PINGONE_ORGADMIN_CLIENT_ID \
    PINGCLI_PINGONE_ORGADMIN_CLIENT_SECRET \
    PINGCLI_PINGONE_ORGADMIN_ENVIRONMENT_ID \
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

# ---------------------------------------------------------------------------
# Optional env vars with defaults
# ---------------------------------------------------------------------------
apply_defaults() {
  : "${PINGCLI_PINGONE_ORGADMIN_REGION_CODE:=NA}"
  : "${REGRESSION_BASE:=main}"
  : "${E2E_KEEP_ENVIRONMENT:=0}"

  export PINGCLI_PINGONE_ORGADMIN_REGION_CODE
  export REGRESSION_BASE
  export E2E_KEEP_ENVIRONMENT
}

# ---------------------------------------------------------------------------
# Globals (set after apply_defaults)
# ---------------------------------------------------------------------------
REPO_ROOT=""
TMPDIR_LOCAL=""
WORKTREE_DIR=""
TF_DIR=""
PROVISIONED_ENV_ID=""

setup_dirs() {
  REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  if [ ! -f "${REPO_ROOT}/Makefile" ]; then
    die "Could not locate repo root (Makefile not found at ${REPO_ROOT})"
  fi

  TMPDIR_LOCAL="$(mktemp -d "${TMPDIR:-/tmp}/pingcli-e2e.XXXXXX")"
  WORKTREE_DIR="${TMPDIR_LOCAL}/worktree-base"
  TF_DIR="${REPO_ROOT}/terraform-test-data/root"

  info "Repo root  : ${REPO_ROOT}"
  info "Temp dir   : ${TMPDIR_LOCAL}"
  info "Base branch: ${REGRESSION_BASE}"
}

# ---------------------------------------------------------------------------
# Cleanup trap - always tears down the provisioned environment, even on
# failure, unless the developer explicitly asked to keep it for debugging.
#
# Trigger condition is the presence of terraform.tfvars.json in TF_DIR, NOT
# PROVISIONED_ENV_ID: `terraform apply` can create the environment and then
# fail on a later resource, in which case create() never reaches the point
# of setting PROVISIONED_ENV_ID, but a real (now-orphaned) environment and
# its tfvars/state already exist in TF_DIR and must still be torn down.
# ---------------------------------------------------------------------------
cleanup() {
  if [ -f "${TF_DIR}/terraform.tfvars.json" ]; then
    if [ "${E2E_KEEP_ENVIRONMENT}" = "1" ]; then
      warn "E2E_KEEP_ENVIRONMENT=1 set - leaving provisioned environment in place."
      warn "Remember to destroy it manually when done (terraform destroy in ${TF_DIR})."
    else
      info "Tearing down provisioned environment${PROVISIONED_ENV_ID:+ ${PROVISIONED_ENV_ID}}..."
      "${TMPDIR_LOCAL}/tf-regression-provision" \
        --action destroy \
        --terraform-dir "${TF_DIR}" \
        || warn "Teardown reported an error - verify manually in the PingOne admin console."
    fi
  fi

  if [ -n "${WORKTREE_DIR}" ] && [ -d "${WORKTREE_DIR}" ]; then
    git -C "${REPO_ROOT}" worktree remove --force "${WORKTREE_DIR}" 2>/dev/null || true
  fi
  if [ -n "${TMPDIR_LOCAL}" ] && [ -d "${TMPDIR_LOCAL}" ]; then
    rm -rf "${TMPDIR_LOCAL}"
  fi
}

# ---------------------------------------------------------------------------
# Provision a throwaway environment and apply terraform-test-data/root.
# Populates PINGCLI_PINGONE_* env vars for the export steps below from the
# tool's JSON stdout - never echoed, only parsed with jq.
# ---------------------------------------------------------------------------
provision() {
  info "Building tf-regression-provision..."
  (cd "${REPO_ROOT}" && go build -o "${TMPDIR_LOCAL}/tf-regression-provision" ./tools/tf-regression-provision/)

  info "Provisioning throwaway PingOne environment (this can take a minute)..."
  local result="${TMPDIR_LOCAL}/provision-result.json"
  "${TMPDIR_LOCAL}/tf-regression-provision" \
    --action create \
    --terraform-dir "${TF_DIR}" \
    --name-prefix "pingcli-terraformer-e2e" \
    >"${result}"

  PROVISIONED_ENV_ID=$(jq -r '.target_environment_id' "${result}")
  # Auth environment (where the org-admin credential's OAuth token is
  # acquired) is distinct from the export target (the throwaway environment
  # just created) - mirrors internal/platform/pingone.NewFromCredentials's
  # workerEnvID vs exportEnvID split.
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

# ---------------------------------------------------------------------------
# Build export/compare binaries (base branch vs current branch)
# ---------------------------------------------------------------------------
build_binaries() {
  info "Creating git worktree for base branch '${REGRESSION_BASE}'..."
  git -C "${REPO_ROOT}" worktree add --detach "${WORKTREE_DIR}" "origin/${REGRESSION_BASE}" \
    2>/dev/null || \
  git -C "${REPO_ROOT}" worktree add --detach "${WORKTREE_DIR}" "${REGRESSION_BASE}"

  info "Building base binary from '${REGRESSION_BASE}'..."
  (cd "${WORKTREE_DIR}" && go build -o "${TMPDIR_LOCAL}/binary-base" .)

  info "Building PR binary from current branch..."
  (cd "${REPO_ROOT}" && go build -o "${TMPDIR_LOCAL}/binary-pr" .)

  info "Building regression-compare tool..."
  (cd "${REPO_ROOT}" && go build -o "${TMPDIR_LOCAL}/regression-compare" ./tools/regression-compare/)

  success "All binaries built."
}

# ---------------------------------------------------------------------------
# Build export CLI args from a matrix entry
# ---------------------------------------------------------------------------
build_args() {
  local format="$1"
  local skip_deps="$2"
  local include_imports="$3"
  local include_values="$4"
  local outdir="$5"

  local args="export --output-format ${format} --out ${outdir} --module-name e2e-test --module-dir e2e-module"

  [ "${skip_deps}" = "true" ]       && args="${args} --skip-dependencies"
  [ "${include_imports}" = "true" ] && args="${args} --include-imports"
  [ "${include_values}" = "true" ]  && args="${args} --include-values"

  printf '%s' "${args}"
}

# ---------------------------------------------------------------------------
# Run a single matrix entry (base binary vs PR binary against the
# provisioned environment). Returns 0 if no breaking changes, 1 otherwise.
# ---------------------------------------------------------------------------
run_entry() {
  local name="$1"
  local format="$2"
  local skip_deps="$3"
  local include_imports="$4"
  local include_values="$5"

  local outdir_base="${TMPDIR_LOCAL}/output-base-${name}"
  local outdir_pr="${TMPDIR_LOCAL}/output-pr-${name}"
  local report="${TMPDIR_LOCAL}/report-${name}.json"

  mkdir -p "${outdir_base}" "${outdir_pr}"

  info "Running matrix entry: ${name}"

  local base_args pr_args
  base_args=$(build_args "${format}" "${skip_deps}" "${include_imports}" "${include_values}" "${outdir_base}")
  "${TMPDIR_LOCAL}/binary-base" ${base_args}

  pr_args=$(build_args "${format}" "${skip_deps}" "${include_imports}" "${include_values}" "${outdir_pr}")
  "${TMPDIR_LOCAL}/binary-pr" ${pr_args}

  "${TMPDIR_LOCAL}/regression-compare" \
    --base-dir "${outdir_base}" \
    --pr-dir   "${outdir_pr}"   \
    --report-file "${report}" || true   # compare exits non-zero on breaking; handled below

  if [ -f "${report}" ] && jq -e '.has_breaking == true' "${report}" &>/dev/null; then
    return 1
  fi
  return 0
}

# ---------------------------------------------------------------------------
# Print summary table
# ---------------------------------------------------------------------------
print_summary() {
  local -a names=("${!1}")
  local -a statuses=("${!2}")
  local -a breaking=("${!3}")
  local -a acceptable=("${!4}")

  local col_name=20 col_status=10 col_break=10 col_acc=10

  printf '\n%s══════════════════════════════════════════════%s\n' "${BOLD}" "${RESET}"
  printf '%s E2E Test Summary%s\n' "${BOLD}" "${RESET}"
  printf '%s══════════════════════════════════════════════%s\n' "${BOLD}" "${RESET}"
  printf ' %-*s  %-*s  %-*s  %-*s\n' \
    "${col_name}" "Matrix Entry" \
    "${col_status}" "Status" \
    "${col_break}" "Breaking" \
    "${col_acc}" "Acceptable"
  printf ' %s  %s  %s  %s\n' \
    "$(printf '─%.0s' $(seq 1 ${col_name}))" \
    "$(printf '─%.0s' $(seq 1 ${col_status}))" \
    "$(printf '─%.0s' $(seq 1 ${col_break}))" \
    "$(printf '─%.0s' $(seq 1 ${col_acc}))"

  for i in "${!names[@]}"; do
    local name="${names[$i]}"
    local status="${statuses[$i]}"
    local brk="${breaking[$i]}"
    local acc="${acceptable[$i]}"

    if [ "${status}" = "PASS" ]; then
      printf ' %-*s  %s%-*s%s  %-*s  %-*s\n' \
        "${col_name}" "${name}" \
        "${GREEN}" "${col_status}" "✅ PASS" "${RESET}" \
        "${col_break}" "${brk}" \
        "${col_acc}" "${acc}"
    else
      printf ' %-*s  %s%-*s%s  %-*s  %-*s\n' \
        "${col_name}" "${name}" \
        "${RED}" "${col_status}" "❌ FAIL" "${RESET}" \
        "${col_break}" "${brk}" \
        "${col_acc}" "${acc}"
    fi
  done

  printf '%s══════════════════════════════════════════════%s\n\n' "${BOLD}" "${RESET}"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  check_prerequisites
  apply_defaults
  setup_dirs

  trap cleanup EXIT

  provision
  build_binaries

  local matrix_file="${REPO_ROOT}/tests/regression/matrix.json"
  [ -f "${matrix_file}" ] || die "Matrix file not found: ${matrix_file}"

  local entry_count
  entry_count=$(jq 'length' "${matrix_file}")

  local -a names=()
  local -a statuses=()
  local -a breaking_counts=()
  local -a acceptable_counts=()
  local overall_exit=0

  for (( i=0; i<entry_count; i++ )); do
    local name format skip_deps include_imports include_values
    name=$(jq -r ".[$i].name" "${matrix_file}")
    format=$(jq -r ".[$i][\"output-format\"]" "${matrix_file}")
    skip_deps=$(jq -r ".[$i][\"skip-dependencies\"]" "${matrix_file}")
    include_imports=$(jq -r ".[$i][\"include-imports\"]" "${matrix_file}")
    include_values=$(jq -r ".[$i][\"include-values\"]" "${matrix_file}")

    local entry_exit=0
    run_entry "${name}" "${format}" "${skip_deps}" "${include_imports}" "${include_values}" || entry_exit=$?

    local report="${TMPDIR_LOCAL}/report-${name}.json"
    local brk=0 acc=0
    if [ -f "${report}" ]; then
      brk=$(jq -r '.breaking_count // 0' "${report}")
      acc=$(jq -r '.acceptable_count // 0' "${report}")
    fi

    names+=("${name}")
    breaking_counts+=("${brk}")
    acceptable_counts+=("${acc}")

    if [ "${entry_exit}" -eq 0 ]; then
      statuses+=("PASS")
      success "Matrix entry '${name}' passed (breaking=${brk}, acceptable=${acc})"
    else
      statuses+=("FAIL")
      fail    "Matrix entry '${name}' has breaking changes (breaking=${brk}, acceptable=${acc})"
      overall_exit=1
    fi
  done

  print_summary names[@] statuses[@] breaking_counts[@] acceptable_counts[@]

  if [ "${overall_exit}" -ne 0 ]; then
    fail "E2E test FAILED - breaking changes detected."
  else
    success "E2E test PASSED - no breaking changes."
  fi

  exit "${overall_exit}"
}

main "$@"
