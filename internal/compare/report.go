package compare

import "strings"

// Report is the JSON-serialized comparison report written by the
// regression-compare tool (--report-file). The schema is a CI contract:
// .github/workflows/regression.yaml's summary job reads has_breaking,
// breaking_count, acceptable_count, and files[].breaking_diffs[] with
// kind/resource/attribute/expected/actual. Field names and tags must not
// change without updating that workflow.
type Report struct {
	BaseDir         string       `json:"base_dir"`
	PRDir           string       `json:"pr_dir"`
	HasBreaking     bool         `json:"has_breaking"`
	AcceptableCount int          `json:"acceptable_count"`
	BreakingCount   int          `json:"breaking_count"`
	Files           []FileReport `json:"files"`

	// Error is set by the shell runners when an export or the compare tool
	// itself crashed (see write_error_report in tests/regression/run-local.sh):
	// the report then carries no files. Renderers must tolerate this shape.
	Error string `json:"error,omitempty"`
}

// FileReport holds the comparison result for a single file.
type FileReport struct {
	Path            string       `json:"path"`
	Kind            string       `json:"kind"` // missing_file, extra_file, compared
	AcceptableDiffs []DiffReport `json:"acceptable_diffs,omitempty"`
	BreakingDiffs   []DiffReport `json:"breaking_diffs,omitempty"`
}

// DiffReport is a single diff entry in the report.
type DiffReport struct {
	Resource  string `json:"resource"`
	Kind      string `json:"kind"`
	Attribute string `json:"attribute,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
}

// mdTable renders a GitHub-flavored markdown table. Values are emitted
// verbatim (callers escape pipes/backticks as needed); the header row is
// bold-agnostic plain text.
func mdTable(sb *strings.Builder, headers []string, rows [][]string) {
	sb.WriteString("| ")
	sb.WriteString(strings.Join(headers, " | "))
	sb.WriteString(" |\n")
	sb.WriteString("|")
	for range headers {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")
	for _, row := range rows {
		sb.WriteString("| ")
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteString(" |\n")
	}
}
