package compare

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// breakingDiff is a test shorthand.
func breakingDiff(kind, resource, attr, expected, actual string) DiffReport {
	return DiffReport{Kind: kind, Resource: resource, Attribute: attr, Expected: expected, Actual: actual}
}

func TestRenderMarkdown_GroupsRepeatedDiffs(t *testing.T) {
	report := &Report{
		HasBreaking:   true,
		BreakingCount: 3,
		Files: []FileReport{
			{
				Path: "mod/app.tf",
				Kind: "compared",
				BreakingDiffs: []DiffReport{
					breakingDiff("value_mismatch", "pingone_application.a", "environment_id", "var.pingone_environment_id", "pingone_environment.x.id"),
					breakingDiff("value_mismatch", "pingone_application.b", "environment_id", "var.pingone_environment_id", "pingone_environment.x.id"),
				},
			},
			{
				Path: "mod/other.tf",
				Kind: "compared",
				BreakingDiffs: []DiffReport{
					breakingDiff("value_mismatch", "pingone_population.p", "environment_id", "var.pingone_environment_id", "pingone_environment.x.id"),
				},
			},
		},
	}

	md := RenderMarkdown(report)

	// The three diffs collapse into one pattern row.
	assert.Contains(t, md, "repeated patterns (1)")
	assert.Contains(t, md, "3 resources")
	// The transition renders in the patterns table; the details section
	// restates it via diffChange. Both are the single grouped pattern.
	assert.GreaterOrEqual(t, strings.Count(md, "var.pingone_environment_id → pingone_environment.x.id"), 1)

	// Affected resources listed in the collapsed section.
	assert.Contains(t, md, "mod/app.tf: pingone_application.a")
	assert.Contains(t, md, "mod/other.tf: pingone_population.p")
}

func TestRenderMarkdown_CountOneDiffStaysIndividual(t *testing.T) {
	report := &Report{
		HasBreaking:   true,
		BreakingCount: 1,
		Files: []FileReport{
			{
				Path:          "mod/x.tf",
				Kind:          "compared",
				BreakingDiffs: []DiffReport{breakingDiff("missing_attribute", "pingone_group.g", "description", "", "")},
			},
		},
	}

	md := RenderMarkdown(report)

	assert.NotContains(t, md, "repeated patterns")
	assert.Contains(t, md, "Other breaking changes (1)")
	assert.Contains(t, md, "`missing_attribute`")
	assert.Contains(t, md, "description absent from PR export")
}

func TestRenderMarkdown_ResourceLocatorNotDuplicated(t *testing.T) {
	// A count-1 diff keeps its resource/attribute context, and the
	// individual-table change cell doesn't repeat "absent" twice.
	report := &Report{
		HasBreaking:   true,
		BreakingCount: 1,
		Files: []FileReport{
			{
				Path:          "mod/x.tf",
				Kind:          "compared",
				BreakingDiffs: []DiffReport{breakingDiff("value_mismatch", "pingone_group.g", "name", "before", "after")},
			},
		},
	}

	md := RenderMarkdown(report)

	assert.Contains(t, md, "pingone_group.g · name")
	assert.Contains(t, md, "before → after")
}

func TestRenderMarkdown_FileLevelKinds(t *testing.T) {
	report := &Report{
		HasBreaking:     true,
		BreakingCount:   1,
		AcceptableCount: 1,
		Files: []FileReport{
			{
				Path:          "mod/removed.tf",
				Kind:          "missing_file",
				BreakingDiffs: []DiffReport{{Kind: "missing_file", Resource: "mod/removed.tf"}},
			},
			{
				Path:            "mod/added.tf",
				Kind:            "extra_file",
				AcceptableDiffs: []DiffReport{{Kind: "extra_file", Resource: "mod/added.tf"}},
			},
		},
	}

	md := RenderMarkdown(report)

	assert.Contains(t, md, "mod/removed.tf")
	assert.Contains(t, md, "mod/added.tf")
	assert.Contains(t, md, "Acceptable additions (1)")
}

func TestRenderMarkdown_ErrorShapeReport(t *testing.T) {
	// The shape the shell runners write when an export/compare crashes.
	report := &Report{
		Error:       "PR export failed (exit 3)",
		HasBreaking: false,
	}

	md := RenderMarkdown(report)
	assert.Contains(t, md, "comparison errored")
	assert.Contains(t, md, "PR export failed (exit 3)")
	assert.NotContains(t, md, "Verdict")

	html, err := RenderHTML(report)
	require.NoError(t, err)
	assert.Contains(t, html, "The comparison errored")
	assert.Contains(t, html, "PR export failed (exit 3)")
	assert.NotContains(t, html, "Breaking changes")
}

func TestRenderMarkdown_PatternCapAndOverflow(t *testing.T) {
	var files []FileReport
	for i := 0; i < 25; i++ {
		files = append(files, FileReport{
			Path: fmt.Sprintf("mod/f%02d.tf", i),
			Kind: "compared",
			BreakingDiffs: []DiffReport{
				breakingDiff("value_mismatch", fmt.Sprintf("res.r%d", i), "environment_id", "var.a", "ref.b"),
			},
		})
	}
	report := &Report{HasBreaking: true, BreakingCount: 25, Files: files}

	md := RenderMarkdown(report)

	assert.Contains(t, md, "25 resources")
	assert.Contains(t, md, "…and 5 more")
	// 20 locators in the collapsed list + 1 inline example in the table row.
	assert.Equal(t, 21, strings.Count(md, "mod/f"))
}

func TestRenderHTML_PatternsAndEscaping(t *testing.T) {
	report := &Report{
		HasBreaking:   true,
		BreakingCount: 2,
		BaseDir:       "/tmp/base",
		PRDir:         "/tmp/pr",
		Files: []FileReport{
			{
				Path: "mod/a<b>.tf",
				Kind: "compared",
				BreakingDiffs: []DiffReport{
					breakingDiff("value_mismatch", "res.<script>alert(1)</script>", "attr", `"quoted & <value>"`, "after"),
					breakingDiff("value_mismatch", "res.two", "attr", `"quoted & <value>"`, "after"),
				},
			},
		},
	}

	html, err := RenderHTML(report)
	require.NoError(t, err)

	// Script injection is escaped.
	assert.NotContains(t, html, "<script>alert(1)</script>")
	assert.Contains(t, html, "&lt;script&gt;")

	// Pattern grouping present.
	assert.Contains(t, html, "repeated patterns (1)")
	assert.Contains(t, html, "2 resources")

	// Self-contained: no external assets.
	assert.NotContains(t, html, "http://")
	assert.NotContains(t, html, "https://")
	assert.Contains(t, html, "color-scheme: light dark")
}

func TestRenderHTML_PassAndNoDiffs(t *testing.T) {
	pass := &Report{AcceptableCount: 2, Files: []FileReport{
		{Path: "mod/new.tf", Kind: "extra_file", AcceptableDiffs: []DiffReport{{Kind: "extra_file", Resource: "mod/new.tf"}}},
	}}
	html, err := RenderHTML(pass)
	require.NoError(t, err)
	assert.Contains(t, html, "verdict-pass")
	assert.Contains(t, html, "Acceptable additions")

	clean := &Report{}
	html, err = RenderHTML(clean)
	require.NoError(t, err)
	assert.Contains(t, html, "verdict-pass")
}

func TestRenderMarkdown_MDTableEscapesPipes(t *testing.T) {
	// A value containing a pipe must not break the table: GFM treats | as a
	// column separator even inside code spans, so it is backslash-escaped.
	report := &Report{
		HasBreaking:   true,
		BreakingCount: 1,
		Files: []FileReport{
			{
				Path:          "mod/x.tf",
				Kind:          "compared",
				BreakingDiffs: []DiffReport{breakingDiff("value_mismatch", "res.r", "attr", "a|b", "c|d")},
			},
		},
	}

	md := RenderMarkdown(report)
	assert.Contains(t, md, `a\|b → c\|d`)
	// The change-table stays a table: header, separator, one row.
	assert.Contains(t, md, "| `value_mismatch` | `res.r · attr` | `a\\|b → c\\|d` |")
}
