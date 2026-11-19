package compare

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
)

// diffPattern groups breaking diffs that share the same kind, attribute, and
// expected→actual transition. A real regression run can emit hundreds of
// diffs that are one repeated change (e.g. environment_id moving from
// var.pingone_environment_id to a resource reference on every resource);
// the pattern view collapses those to a single row.
type diffPattern struct {
	Kind      string
	Attribute string
	Expected  string
	Actual    string

	// Count of diffs in the group.
	Count int

	// Resources holds "file: resource" locators, capped at maxResourcesListed
	// (plus MoreCount for the overflow).
	Resources []string
	MoreCount int
}

// maxResourcesListed caps the per-pattern resource list in markdown.
const maxResourcesListed = 20

// patternKey is the grouping signature.
func patternKey(d DiffReport) string {
	return d.Kind + "\x00" + d.Attribute + "\x00" + d.Expected + "\x00" + d.Actual
}

// groupBreakingDiffs partitions breaking diffs into repeated patterns
// (count > 1) and one-off diffs, both sorted deterministically.
func groupBreakingDiffs(report *Report) (patterns []diffPattern, oneOffs []DiffReport) {
	byKey := map[string]*diffPattern{}
	var order []string

	for _, f := range report.Files {
		for _, d := range f.BreakingDiffs {
			key := patternKey(d)
			if _, seen := byKey[key]; !seen {
				byKey[key] = &diffPattern{
					Kind:      d.Kind,
					Attribute: d.Attribute,
					Expected:  d.Expected,
					Actual:    d.Actual,
				}
				order = append(order, key)
			}
			p := byKey[key]
			p.Count++
			if p.Count <= maxResourcesListed {
				p.Resources = append(p.Resources, f.Path+": "+d.Resource)
			} else {
				p.MoreCount++
			}
		}
	}

	// Deterministic order: most-frequent pattern first, then by first sight.
	sorted := make([]*diffPattern, 0, len(order))
	for _, k := range order {
		sorted = append(sorted, byKey[k])
	}
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Count > sorted[j].Count })

	for _, p := range sorted {
		if p.Count > 1 {
			patterns = append(patterns, *p)
		} else {
			oneOffs = append(oneOffs, DiffReport{
				Kind: p.Kind, Attribute: p.Attribute, Expected: p.Expected, Actual: p.Actual,
			})
		}
	}

	// One-offs are keyed only by pattern, so file/resource context was lost;
	// recover the locators for rendering.
	for i := range oneOffs {
		for _, f := range report.Files {
			for _, d := range f.BreakingDiffs {
				if patternKey(d) == patternKey(oneOffs[i]) {
					oneOffs[i].Resource = d.Resource
					if oneOffs[i].Resource != "" {
						break
					}
				}
			}
		}
	}

	return patterns, oneOffs
}

// diffLocator renders a diff for a table cell: what changed, where.
func diffLocator(d DiffReport) string {
	switch d.Kind {
	case "missing_file", "extra_file":
		// The tool sets Resource to the file path for file-level entries.
		return d.Resource
	case "missing_resource", "extra_resource":
		return d.Resource
	default:
		return d.Resource + " · " + d.Attribute
	}
}

// diffChange renders the expected→actual transition for a table cell.
func diffChange(d DiffReport) string {
	switch d.Kind {
	case "missing_file":
		return "file present in base export, absent from PR export"
	case "missing_resource":
		return "resource absent from PR export"
	case "missing_attribute", "missing_block":
		return d.Attribute + " absent from PR export"
	case "extra_resource", "extra_attribute":
		return "added in PR export"
	default:
		return d.Expected + " → " + d.Actual
	}
}

// RenderMarkdown renders the report as GitHub-flavored markdown.
func RenderMarkdown(report *Report) string {
	var sb strings.Builder

	title := "Regression Comparison Report"
	fmt.Fprintf(&sb, "# %s\n\n", title)

	if report.Error != "" {
		fmt.Fprintf(&sb, "> **⚠️ The comparison errored:** %s\n", report.Error)
		fmt.Fprintf(&sb, "> This entry's exports did not complete. See `log-base-*.txt` / `log-pr-*.txt` for the failing step's output.\n\n")
		return sb.String()
	}

	if report.HasBreaking {
		fmt.Fprintf(&sb, "**Verdict: ❌ FAIL** — %d breaking change(s), %d acceptable addition(s).\n\n",
			report.BreakingCount, report.AcceptableCount)
	} else if report.AcceptableCount > 0 {
		fmt.Fprintf(&sb, "**Verdict: ✅ PASS** — %d acceptable addition(s), no breaking changes.\n\n", report.AcceptableCount)
	} else {
		sb.WriteString("**Verdict: ✅ PASS** — no differences.\n\n")
	}

	patterns, oneOffs := groupBreakingDiffs(report)

	if len(patterns) > 0 {
		fmt.Fprintf(&sb, "## Breaking changes — repeated patterns (%d)\n\n", len(patterns))
		fmt.Fprintf(&sb, "Each row is one change applied to N resources.\n\n")
		var rows [][]string
		for _, p := range patterns {
			change := p.Expected + " → " + p.Actual
			if change == " → " {
				change = "presence changed"
			}
			loc := fmt.Sprintf("%d resources", p.Count)
			if len(p.Resources) > 0 {
				loc = fmt.Sprintf("%d resources (e.g. `%s`)", p.Count, p.Resources[0])
			}
			rows = append(rows, []string{
				fmt.Sprintf("`%s`", p.Kind),
				fmt.Sprintf("`%s`", p.Attribute),
				wrapCode(change),
				loc,
			})
		}
		mdTable(&sb, []string{"Kind", "Attribute", "Expected → Actual", "Scope"}, rows)

		sb.WriteString("\n<details><summary>Affected resources</summary>\n\n")
		for _, p := range patterns {
			fmt.Fprintf(&sb, "**`%s`** `%s` — %s\n", p.Kind, p.Attribute, diffChange(DiffReport{
				Kind: p.Kind, Attribute: p.Attribute, Expected: p.Expected, Actual: p.Actual,
			}))
			for _, r := range p.Resources {
				fmt.Fprintf(&sb, "- `%s`\n", r)
			}
			if p.MoreCount > 0 {
				fmt.Fprintf(&sb, "- …and %d more\n", p.MoreCount)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("</details>\n\n")
	}

	if len(oneOffs) > 0 {
		fmt.Fprintf(&sb, "## Other breaking changes (%d)\n\n", len(oneOffs))
		var rows [][]string
		for _, d := range oneOffs {
			rows = append(rows, []string{
				fmt.Sprintf("`%s`", d.Kind),
				wrapCode(diffLocator(d)),
				wrapCode(diffChange(d)),
			})
		}
		mdTable(&sb, []string{"Kind", "Where", "Change"}, rows)
		sb.WriteString("\n")
	}

	if report.AcceptableCount > 0 {
		fmt.Fprintf(&sb, "## Acceptable additions (%d)\n\n", report.AcceptableCount)
		var rows [][]string
		shown := 0
		const maxAcceptableRows = 50
		for _, f := range report.Files {
			for _, d := range f.AcceptableDiffs {
				if shown >= maxAcceptableRows {
					break
				}
				rows = append(rows, []string{
					fmt.Sprintf("`%s`", d.Kind),
					wrapCode(diffLocator(d)),
				})
				shown++
			}
		}
		if report.AcceptableCount > shown {
			rows = append(rows, []string{"…", fmt.Sprintf("…and %d more (see the JSON report)", report.AcceptableCount-shown)})
		}
		mdTable(&sb, []string{"Kind", "Where"}, rows)
		sb.WriteString("\n")
	}

	return sb.String()
}

// wrapCode backtick-quotes a value for a markdown table cell, escaping any
// backticks inside by switching the fence to a double backtick, and escaping
// pipes (GFM treats them as column separators even inside code spans).
func wrapCode(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "|", "\\|")
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return "`` " + s + " ``"
}

// htmlReportData is the template data for the HTML report page.
type htmlReportData struct {
	Title     string
	Verdict   string
	VerdictOK bool
	Counts    string
	BaseDir   string
	PRDir     string
	Error     string

	Patterns []diffPattern
	OneOffs  []DiffReport

	HasAcceptable bool
	Acceptable    []fileDiffRow
	AcceptableCap int
}

// fileDiffRow is one row of the HTML diff table.
type fileDiffRow struct {
	Kind   string
	Where  string
	Change string
}

var htmlReportTemplate = template.Must(template.New("regression-report").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
:root { color-scheme: light dark; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; padding: 2rem; line-height: 1.5; }
h1 { font-size: 1.4rem; margin-bottom: 0.25rem; }
h2 { font-size: 1.1rem; margin-top: 1.5rem; }
.verdict-fail { color: #b91c1c; font-weight: 600; }
.verdict-pass { color: #15803d; font-weight: 600; }
.error-box { border-left: 4px solid #d97706; background: #fffbeb; padding: 0.75rem 1rem; margin: 1rem 0; }
.meta { color: #666; font-size: 0.85rem; }
table { border-collapse: collapse; margin: 0.5rem 0 1rem; width: 100%; max-width: 60rem; font-size: 0.9rem; }
th, td { border: 1px solid #8884; padding: 0.35rem 0.6rem; text-align: left; vertical-align: top; }
th { background: #8881; }
code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.85em; word-break: break-all; }
details { margin: 0.75rem 0; }
summary { cursor: pointer; font-weight: 600; }
ul { margin: 0.25rem 0 0.75rem; }
@media (prefers-color-scheme: dark) {
  body { background: #1a1a1a; color: #e5e5e5; }
  .verdict-fail { color: #f87171; }
  .verdict-pass { color: #4ade80; }
  .error-box { background: #271a05; }
  .meta { color: #999; }
}
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<p class="verdict-{{if .VerdictOK}}pass{{else}}fail{{end}}">{{.Verdict}}</p>
<p class="meta">{{.Counts}} · base: <code>{{.BaseDir}}</code> · pr: <code>{{.PRDir}}</code></p>

{{if .Error}}
<div class="error-box"><strong>⚠️ The comparison errored:</strong> {{.Error}}<br>
This entry's exports did not complete. See <code>log-base-*.txt</code> / <code>log-pr-*.txt</code> for the failing step's output.</div>
{{end}}

{{if .Patterns}}
<h2>Breaking changes — repeated patterns ({{len .Patterns}})</h2>
<p class="meta">Each row is one change applied to N resources.</p>
<table>
<tr><th>Kind</th><th>Attribute</th><th>Expected → Actual</th><th>Scope</th><th>Examples</th></tr>
{{range .Patterns}}
<tr>
  <td><code>{{.Kind}}</code></td>
  <td><code>{{.Attribute}}</code></td>
  <td>{{if or .Expected .Actual}}<code>{{.Expected}}</code> → <code>{{.Actual}}</code>{{else}}presence changed{{end}}</td>
  <td>{{.Count}} resources</td>
  <td>
    <details><summary>{{len .Resources}} shown{{if .MoreCount}} (+{{.MoreCount}} more){{end}}</summary>
      <ul>{{range .Resources}}<li><code>{{.}}</code></li>{{end}}</ul>
    </details>
  </td>
</tr>
{{end}}
</table>
{{end}}

{{if .OneOffs}}
<h2>Other breaking changes ({{len .OneOffs}})</h2>
<table>
<tr><th>Kind</th><th>Where</th><th>Change</th></tr>
{{range .OneOffs}}
<tr><td><code>{{.Kind}}</code></td><td><code>{{.Resource}}{{if .Attribute}} · {{.Attribute}}{{end}}</code></td><td>{{if or .Expected .Actual}}<code>{{.Expected}}</code> → <code>{{.Actual}}</code>{{else}}{{.Kind}}{{end}}</td></tr>
{{end}}
</table>
{{end}}

{{if .HasAcceptable}}
<h2>Acceptable additions ({{.AcceptableCap}})</h2>
<table>
<tr><th>Kind</th><th>Where</th></tr>
{{range .Acceptable}}
<tr><td><code>{{.Kind}}</code></td><td><code>{{.Where}}</code></td></tr>
{{end}}
</table>
{{end}}
</body>
</html>
`))

// RenderHTML renders the report as a self-contained HTML page. All content
// is auto-escaped by html/template; no template.HTML conversions.
func RenderHTML(report *Report) (string, error) {
	data := htmlReportData{
		Title:   "Regression Comparison Report",
		BaseDir: report.BaseDir,
		PRDir:   report.PRDir,
		Error:   report.Error,
	}

	if report.Error != "" {
		data.Verdict = "ERROR"
		data.VerdictOK = false
		data.Counts = "comparison did not complete"
	} else if report.HasBreaking {
		data.Verdict = "❌ FAIL"
		data.VerdictOK = false
		data.Counts = fmt.Sprintf("%d breaking, %d acceptable", report.BreakingCount, report.AcceptableCount)
	} else {
		data.Verdict = "✅ PASS"
		data.VerdictOK = true
		data.Counts = fmt.Sprintf("%d acceptable", report.AcceptableCount)
	}

	if report.Error == "" {
		patterns, oneOffs := groupBreakingDiffs(report)
		data.Patterns = patterns
		data.OneOffs = oneOffs

		const maxAcceptableRows = 50
		shown := 0
		for _, f := range report.Files {
			for _, d := range f.AcceptableDiffs {
				if shown >= maxAcceptableRows {
					break
				}
				data.Acceptable = append(data.Acceptable, fileDiffRow{
					Kind:  d.Kind,
					Where: diffLocator(d),
				})
				shown++
			}
		}
		data.HasAcceptable = len(data.Acceptable) > 0
		data.AcceptableCap = report.AcceptableCount
	}

	var sb strings.Builder
	if err := htmlReportTemplate.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}
