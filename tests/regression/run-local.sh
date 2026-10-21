#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Regression test runner — compares export output between base and current branch
#
# Reports, export outputs, and per-entry logs land in
# regression-reports/run-<timestamp>/ (also linked from regression-reports/latest)
# and persist after the run — inspect them even when a comparison fails.
# The summary table, the verdict, and the report locations print at the end
# regardless of exit status.
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
#
# names/statuses/breaking_counts/acceptable_counts are file-scoped so the
# EXIT trap (finalize) can print the summary even when main aborts.
# ---------------------------------------------------------------------------
REPO_ROOT=""
TMPDIR_LOCAL=""
WORKTREE_DIR=""
REPORTS_DIR=""

names=()
statuses=()
breaking_counts=()
acceptable_counts=()

setup_dirs() {
  # Locate repo root (directory containing the Makefile)
  REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  if [ ! -f "${REPO_ROOT}/Makefile" ]; then
    die "Could not locate repo root (Makefile not found at ${REPO_ROOT})"
  fi

  # Working area (worktree + binaries) is temporary; reports and export
  # outputs persist under regression-reports/ so failures can be inspected.
  TMPDIR_LOCAL="$(mktemp -d "${TMPDIR:-/tmp}/pingcli-regression.XXXXXX")"
  WORKTREE_DIR="${TMPDIR_LOCAL}/worktree-base"
  REPORTS_DIR="${REPO_ROOT}/regression-reports/run-$(date +%Y%m%d-%H%M%S)"
  mkdir -p "${REPORTS_DIR}"

  info "Repo root  : ${REPO_ROOT}"
  info "Base branch: ${REGRESSION_BASE}"
  info "Reports    : ${REPORTS_DIR}"
}

# ---------------------------------------------------------------------------
# EXIT trap: print the summary table and verdict (always, pass or fail),
# clean up the temporary worktree/binaries, then point at the persisted
# reports. The script's exit status is preserved.
# ---------------------------------------------------------------------------
finalize() {
  local rc=$?

  if [ "${#names[@]}" -gt 0 ]; then
    print_summary names[@] statuses[@] breaking_counts[@] acceptable_counts[@]

    if [ "${rc}" -eq 0 ]; then
      success "Regression test PASSED — no breaking changes."
    else
      fail "Regression test FAILED — breaking changes or errors detected."
    fi
  fi

  if [ -n "${WORKTREE_DIR}" ] && [ -d "${WORKTREE_DIR}" ]; then
    git -C "${REPO_ROOT}" worktree remove --force "${WORKTREE_DIR}" 2>/dev/null || true
  fi
  if [ -n "${TMPDIR_LOCAL}" ] && [ -d "${TMPDIR_LOCAL}" ]; then
    rm -rf "${TMPDIR_LOCAL}"
  fi

  if [ -d "${REPORTS_DIR}" ]; then
    ln -sfn "$(basename "${REPORTS_DIR}")" "${REPO_ROOT}/regression-reports/latest" || true
    info "Reports and export outputs: ${REPORTS_DIR}"
    info "Quick access: regression-reports/latest/"
    for i in "${!names[@]}"; do
      if [ "${statuses[$i]}" != "PASS" ]; then
        info "  ${names[$i]}: ${REPORTS_DIR}/report-${names[$i]}.md (report-${names[$i]}.html · diff: log-compare-${names[$i]}.txt)"
      fi
    done
  fi

  exit "${rc}"
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
  local format="$1"
  local skip_deps="$2"
  local include_imports="$3"
  local include_values="$4"
  local outdir="$5"

  local args="export --output-format ${format} --out ${outdir} --module-name regression-test --module-dir regression-module"

  [ "${skip_deps}" = "true" ]      && args="${args} --skip-dependencies"
  [ "${include_imports}" = "true" ] && args="${args} --include-imports"
  [ "${include_values}" = "true" ]  && args="${args} --include-values"

  printf '%s' "${args}"
}

# write_error_report records an infrastructure failure (export crash, compare
# crash) as a report file, so the entry still shows up in the persisted
# evidence instead of reading as a pass. Writes both the JSON error shape the
# classification logic reads and a friendly markdown twin.
write_error_report() {
  local report="$1"
  local error="$2"
  jq -n --arg err "${error}" \
    '{has_breaking: false, error: $err, breaking_count: 0, acceptable_count: 0}' > "${report}"
  {
    echo "# Regression Comparison Report"
    echo
    echo "> **⚠️ The comparison errored:** ${error}"
    echo "> This entry's exports did not complete. See \`log-base-*.txt\` / \`log-pr-*.txt\` for the failing step's output."
  } > "${report%.json}.md"
}

# ---------------------------------------------------------------------------
# Run a single matrix entry
# Returns 0 if no breaking changes, 1 if breaking changes found, 2 on an
# infrastructure error (export or compare failure). Export logs are captured
# to files to keep the console readable; compare output streams and is
# captured too.
# ---------------------------------------------------------------------------
run_entry() {
  local name="$1"
  local format="$2"
  local skip_deps="$3"
  local include_imports="$4"
  local include_values="$5"

  local outdir_base="${REPORTS_DIR}/output-base-${name}"
  local outdir_pr="${REPORTS_DIR}/output-pr-${name}"
  local report="${REPORTS_DIR}/report-${name}.json"

  mkdir -p "${outdir_base}" "${outdir_pr}"

  info "Running matrix entry: ${name}"

  local base_args pr_args
  base_args=$(build_args "${format}" "${skip_deps}" "${include_imports}" "${include_values}" "${outdir_base}")

  set +e
  "${TMPDIR_LOCAL}/binary-base" ${base_args} >"${REPORTS_DIR}/log-base-${name}.txt" 2>&1
  local base_rc=$?
  set -e
  if [ "${base_rc}" -ne 0 ]; then
    fail "Base export failed for '${name}' (exit ${base_rc}) — see ${REPORTS_DIR}/log-base-${name}.txt"
    write_error_report "${report}" "base export failed (exit ${base_rc})"
    return 2
  fi

  pr_args=$(build_args "${format}" "${skip_deps}" "${include_imports}" "${include_values}" "${outdir_pr}")

  set +e
  "${TMPDIR_LOCAL}/binary-pr" ${pr_args} >"${REPORTS_DIR}/log-pr-${name}.txt" 2>&1
  local pr_rc=$?
  set -e
  if [ "${pr_rc}" -ne 0 ]; then
    fail "PR export failed for '${name}' (exit ${pr_rc}) — see ${REPORTS_DIR}/log-pr-${name}.txt"
    write_error_report "${report}" "PR export failed (exit ${pr_rc})"
    return 2
  fi

  set +e
  "${TMPDIR_LOCAL}/regression-compare" \
    --base-dir "${outdir_base}" \
    --pr-dir   "${outdir_pr}"   \
    --report-file "${report}" \
    --markdown "${REPORTS_DIR}/report-${name}.md" \
    --html "${REPORTS_DIR}/report-${name}.html" 2>&1 | tee "${REPORTS_DIR}/log-compare-${name}.txt"
  local compare_rc=${PIPESTATUS[0]}
  set -e

  if [ ! -f "${report}" ]; then
    fail "regression-compare produced no report for '${name}' (exit ${compare_rc}) — see ${REPORTS_DIR}/log-compare-${name}.txt"
    write_error_report "${report}" "regression-compare produced no report (exit ${compare_rc})"
    return 2
  fi

  if jq -e '.has_breaking == true' "${report}" >/dev/null 2>&1; then
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

  local col_name=20 col_status=12 col_break=10 col_acc=10

  printf '\n%s══════════════════════════════════════════════════%s\n' "${BOLD}" "${RESET}"
  printf '%s Regression Test Summary%s\n' "${BOLD}" "${RESET}"
  printf '%s══════════════════════════════════════════════════%s\n' "${BOLD}" "${RESET}"
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

    case "${status}" in
      PASS)
        printf ' %-*s  %s%-*s%s  %-*s  %-*s\n' \
          "${col_name}" "${name}" \
          "${GREEN}" "${col_status}" "✅ PASS" "${RESET}" \
          "${col_break}" "${brk}" \
          "${col_acc}" "${acc}"
        ;;
      ERROR)
        printf ' %-*s  %s%-*s%s  %-*s  %-*s\n' \
          "${col_name}" "${name}" \
          "${YELLOW}" "${col_status}" "⚠️ ERROR" "${RESET}" \
          "${col_break}" "${brk}" \
          "${col_acc}" "${acc}"
        ;;
      *)
        printf ' %-*s  %s%-*s%s  %-*s  %-*s\n' \
          "${col_name}" "${name}" \
          "${RED}" "${col_status}" "❌ BREAKING" "${RESET}" \
          "${col_break}" "${brk}" \
          "${col_acc}" "${acc}"
        ;;
    esac
  done

  printf '%s══════════════════════════════════════════════════%s\n\n' "${BOLD}" "${RESET}"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  check_prerequisites
  apply_defaults
  setup_dirs

  trap finalize EXIT

  build_binaries

  local matrix_file="${REPO_ROOT}/tests/regression/matrix.json"
  [ -f "${matrix_file}" ] || die "Matrix file not found: ${matrix_file}"

  local entry_count
  entry_count=$(jq 'length' "${matrix_file}")

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

    local report="${REPORTS_DIR}/report-${name}.json"
    local brk=0 acc=0
    if [ -f "${report}" ]; then
      brk=$(jq -r '.breaking_count // 0' "${report}")
      acc=$(jq -r '.acceptable_count // 0' "${report}")
    fi

    names+=("${name}")
    breaking_counts+=("${brk}")
    acceptable_counts+=("${acc}")

    case "${entry_exit}" in
      0)
        statuses+=("PASS")
        success "Matrix entry '${name}' passed (breaking=${brk}, acceptable=${acc})"
        ;;
      2)
        statuses+=("ERROR")
        fail "Matrix entry '${name}' errored — see ${REPORTS_DIR}"
        overall_exit=1
        ;;
      *)
        statuses+=("BREAKING")
        fail "Matrix entry '${name}' has breaking changes (breaking=${brk}, acceptable=${acc})"
        overall_exit=1
        ;;
    esac
  done

  exit "${overall_exit}"
}

main "$@"
