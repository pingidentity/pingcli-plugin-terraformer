#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Regression test runner — compares export output between base and current branch
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
    PINGCLI_PINGONE_ENVIRONMENT_ID \
    PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID \
    PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET; do
    if [ -z "${!var:-}" ]; then
      fail "Required environment variable not set: ${var}"
      missing=1
    fi
  done

  if ! command -v jq &>/dev/null; then
    fail "Required tool not found: jq (install via 'brew install jq' or your package manager)"
    missing=1
  fi

  [ "$missing" -eq 0 ] || exit 1
}

# ---------------------------------------------------------------------------
# Optional env vars with defaults
# ---------------------------------------------------------------------------
apply_defaults() {
  : "${PINGCLI_PINGONE_REGION_CODE:=NA}"
  : "${PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID:=${PINGCLI_PINGONE_ENVIRONMENT_ID}}"
  : "${REGRESSION_BASE:=main}"

  export PINGCLI_PINGONE_REGION_CODE
  export PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID
  export REGRESSION_BASE
}

# ---------------------------------------------------------------------------
# Globals (set after apply_defaults)
# ---------------------------------------------------------------------------
REPO_ROOT=""
TMPDIR_LOCAL=""
WORKTREE_DIR=""

setup_dirs() {
  # Locate repo root (directory containing the Makefile)
  REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  if [ ! -f "${REPO_ROOT}/Makefile" ]; then
    die "Could not locate repo root (Makefile not found at ${REPO_ROOT})"
  fi

  TMPDIR_LOCAL="$(mktemp -d "${TMPDIR:-/tmp}/pingcli-regression.XXXXXX")"
  WORKTREE_DIR="${TMPDIR_LOCAL}/worktree-base"

  info "Repo root  : ${REPO_ROOT}"
  info "Temp dir   : ${TMPDIR_LOCAL}"
  info "Base branch: ${REGRESSION_BASE}"
}

# ---------------------------------------------------------------------------
# Cleanup trap
# ---------------------------------------------------------------------------
cleanup() {
  if [ -n "${WORKTREE_DIR}" ] && [ -d "${WORKTREE_DIR}" ]; then
    git -C "${REPO_ROOT}" worktree remove --force "${WORKTREE_DIR}" 2>/dev/null || true
  fi
  if [ -n "${TMPDIR_LOCAL}" ] && [ -d "${TMPDIR_LOCAL}" ]; then
    rm -rf "${TMPDIR_LOCAL}"
  fi
}

# ---------------------------------------------------------------------------
# Build binaries
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
  local name="$1"
  local format="$2"
  local skip_deps="$3"
  local include_imports="$4"
  local include_values="$5"
  local outdir="$6"

  local args="export --output-format ${format} --out ${outdir} --module-name regression-test --module-dir regression-module"

  [ "${skip_deps}" = "true" ]      && args="${args} --skip-dependencies"
  [ "${include_imports}" = "true" ] && args="${args} --include-imports"
  [ "${include_values}" = "true" ]  && args="${args} --include-values"

  printf '%s' "${args}"
}

# ---------------------------------------------------------------------------
# Run a single matrix entry
# Returns 0 if no breaking changes, 1 if breaking changes found.
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

  # Build args as a string and use eval for word splitting
  local base_args pr_args
  base_args=$(build_args "${name}" "${format}" "${skip_deps}" "${include_imports}" "${include_values}" "${outdir_base}")
  "${TMPDIR_LOCAL}/binary-base" ${base_args}

  pr_args=$(build_args "${name}" "${format}" "${skip_deps}" "${include_imports}" "${include_values}" "${outdir_pr}")
  "${TMPDIR_LOCAL}/binary-pr" ${pr_args}

  "${TMPDIR_LOCAL}/regression-compare" \
    --base-dir "${outdir_base}" \
    --pr-dir   "${outdir_pr}"   \
    --report-file "${report}" || true   # compare exits non-zero on breaking; we handle it ourselves

  # Return whether breaking changes were found
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
  printf '%s Regression Test Summary%s\n' "${BOLD}" "${RESET}"
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
# Copy reports to repo root
# ---------------------------------------------------------------------------
copy_reports() {
  local reports_dir="${REPO_ROOT}/regression-reports"
  mkdir -p "${reports_dir}"
  for f in "${TMPDIR_LOCAL}"/report-*.json; do
    [ -f "${f}" ] && cp "${f}" "${reports_dir}/"
  done
  info "Reports copied to: ${reports_dir}"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  check_prerequisites
  apply_defaults
  setup_dirs

  trap cleanup EXIT

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

  copy_reports
  print_summary names[@] statuses[@] breaking_counts[@] acceptable_counts[@]

  if [ "${overall_exit}" -ne 0 ]; then
    fail "Regression test FAILED — breaking changes detected."
  else
    success "Regression test PASSED — no breaking changes."
  fi

  exit "${overall_exit}"
}

main "$@"
