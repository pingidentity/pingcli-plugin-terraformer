package review

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// ChecklistStatus classifies a checklist row.
type ChecklistStatus string

const (
	// StatusPass means the tool verified the item mechanically.
	StatusPass ChecklistStatus = "PASS"
	// StatusCheck means a human must verify the item.
	StatusCheck ChecklistStatus = "CHECK"
	// StatusSkipped means the item could not run (e.g. no live credentials).
	StatusSkipped ChecklistStatus = "SKIPPED"
	// StatusWarn means the mechanical check found something to look at.
	StatusWarn ChecklistStatus = "WARN"
)

// ChecklistRow is one line of the review checklist.
type ChecklistRow struct {
	Status ChecklistStatus
	Item   string
	Detail string
}

// ChecklistInput carries what the checklist generator checks.
type ChecklistInput struct {
	// Types under review (sorted by the caller).
	Types []string

	// Defs maps resource_type to definition.
	Defs map[string]*schema.ResourceDefinition

	// RepoDir is the repository root for disk checks.
	RepoDir string

	// ExportDir is the bundle's export directory ("" in dry-run).
	ExportDir string

	// ExportModuleDir is the generated child-module directory name.
	ExportModuleDir string

	// FallbackVariables lists fallback-variable names emitted by the export
	// (from ExportResult.FallbackVariables), for the explanation prompt.
	FallbackVariables []string

	// UnresolvedUUIDs maps resource_type → unresolved UUID count found in the
	// generated HCL.
	UnresolvedUUIDs map[string]int

	// EnvID is the live environment the export came from ("" in dry-run).
	EnvID string

	// KeepEnvironment notes the caller ran with E2E_KEEP_ENVIRONMENT=1.
	KeepEnvironment bool

	// LiveMode is true when an export/snapshot was produced.
	LiveMode bool
}

// RenderChecklist renders the review checklist as markdown.
func RenderChecklist(in ChecklistInput) string {
	var sb strings.Builder
	rows := buildChecklist(in)

	sb.WriteString("# Resource Review Checklist\n\n")
	if in.LiveMode {
		fmt.Fprintf(&sb, "Exported from live environment `%s`", in.EnvID)
		if in.KeepEnvironment {
			sb.WriteString(" — **left running** (`E2E_KEEP_ENVIRONMENT=1`); destroy manually when review completes.")
		} else {
			sb.WriteString(" (destroyed after export).")
		}
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Dry-run mode (no live credentials): export and snapshot items are skipped. ")
		sb.WriteString("Re-run with the `PINGCLI_PINGONE_*` environment variables set for the full review.\n\n")
	}

	for _, r := range rows {
		fmt.Fprintf(&sb, "- **[%s]** %s", r.Status, r.Item)
		if r.Detail != "" {
			fmt.Fprintf(&sb, " — %s", r.Detail)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("Attach or summarize this bundle in the PR description. Every **CHECK** row needs an explicit note in the PR; every **WARN** row needs a fix or a justification.\n")
	return sb.String()
}

func buildChecklist(in ChecklistInput) []ChecklistRow {
	var rows []ChecklistRow

	// Fixture checks (mechanical, work in both modes).
	for _, rt := range in.Types {
		rows = append(rows, checkE2EFixture(in, rt))
		rows = append(rows, checkRegressionEnvFixture(in, rt))
	}

	// Definition-level prompts.
	for _, rt := range in.Types {
		def, ok := in.Defs[rt]
		if !ok {
			continue
		}
		rows = append(rows, definitionPrompts(def)...)
	}

	// Export evidence rows.
	rows = append(rows, exportEvidenceRows(in)...)

	// Live-environment row.
	if in.LiveMode {
		rows = append(rows, ChecklistRow{
			Status: StatusPass,
			Item:   "Live environment contained the change",
			Detail: fmt.Sprintf("environment `%s`; see `meta.json` for provenance", in.EnvID),
		})
	} else {
		rows = append(rows, ChecklistRow{
			Status: StatusSkipped,
			Item:   "Live environment contained the change",
			Detail: "no credentials — run with PINGCLI_PINGONE_* env vars",
		})
	}

	return rows
}

// checkE2EFixture verifies the type has a fixture module wired into the
// composition root.
func checkE2EFixture(in ChecklistInput, rt string) ChecklistRow {
	short := strings.TrimPrefix(rt, "pingone_")

	// pingone_environment IS the composition root — the throwaway
	// environment the fixtures are created in. No fixture module applies.
	if rt == "pingone_environment" {
		return ChecklistRow{Status: StatusPass, Item: "`pingone_environment` E2E fixture",
			Detail: "is the composition root itself (terraform-test-data/root)"}
	}

	fixtureDir := in.RepoDir + "/terraform-test-data"
	category, fixtureMain := findFixtureDir(fixtureDir, short)

	if fixtureMain == "" {
		if isAutoCoveredType(rt) {
			return ChecklistRow{Status: StatusPass, Item: fmt.Sprintf("`%s` E2E fixture", rt),
				Detail: "auto-covered by its parent fixture (no module needed)"}
		}
		return ChecklistRow{Status: StatusWarn, Item: fmt.Sprintf("`%s` E2E fixture", rt),
			Detail: "no fixture module found in terraform-test-data/ — add one per the new-resource skill Phase 5f, or document it as a known gap"}
	}

	// Wired into root?
	rootMain, err := os.ReadFile(fixtureDir + "/root/main.tf")
	wired := err == nil && strings.Contains(string(rootMain), "/"+short+"\"")

	if wired {
		return ChecklistRow{Status: StatusPass, Item: fmt.Sprintf("`%s` E2E fixture", rt),
			Detail: fixtureMain + " (" + category + " fixture)"}
	}
	return ChecklistRow{Status: StatusWarn, Item: fmt.Sprintf("`%s` E2E fixture", rt),
		Detail: "fixture module exists but is not wired into terraform-test-data/root/main.tf"}
}

// findFixtureDir locates terraform-test-data/<category>/<short>/main.tf.
func findFixtureDir(root, short string) (category, mainPath string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "root" {
			continue
		}
		candidate := root + "/" + e.Name() + "/" + short
		if fi, err := os.Stat(candidate + "/main.tf"); err == nil && !fi.IsDir() {
			return e.Name(), candidate + "/main.tf"
		}
	}
	return "", ""
}

// isAutoCoveredType lists resource types that need no fixture of their own
// because PingOne generates them on parent creation.
func isAutoCoveredType(rt string) bool {
	switch rt {
	case "pingone_resource_secret", "pingone_application_secret":
		return true
	}
	return false
}

// checkRegressionEnvFixture greps the static regression environment config.
func checkRegressionEnvFixture(in ChecklistInput, rt string) ChecklistRow {
	dir := in.RepoDir + "/tests/regression/environment"
	found := grepDir(dir, rt)
	if found {
		return ChecklistRow{Status: StatusPass, Item: fmt.Sprintf("`%s` in static regression environment", rt)}
	}
	return ChecklistRow{Status: StatusCheck, Item: fmt.Sprintf("`%s` in static regression environment", rt),
		Detail: "not found in tests/regression/environment/ — add via the regression-env-apply workflow, or note why it is absent"}
}

// definitionPrompts emits the definition-level CHECK rows.
func definitionPrompts(def *schema.ResourceDefinition) []ChecklistRow {
	var rows []ChecklistRow
	rt := def.Metadata.ResourceType

	var computed, sensitive []string
	collectAttrFlags(def.Attributes, &computed, &sensitive)
	sort.Strings(computed)
	sort.Strings(sensitive)

	if len(computed) > 0 {
		rows = append(rows, ChecklistRow{Status: StatusCheck,
			Item:   fmt.Sprintf("`%s` computed attributes", rt),
			Detail: "declared: " + joinStrings(computed) + " — every one present in the API response must carry source_path (rule from #125)"})
	}
	if len(sensitive) > 0 {
		rows = append(rows, ChecklistRow{Status: StatusCheck,
			Item:   fmt.Sprintf("`%s` sensitive attributes", rt),
			Detail: "declared: " + joinStrings(sensitive) + " — confirm the API masks them and masked ones declare masked_secret"})
	}

	rows = append(rows, ChecklistRow{Status: StatusCheck,
		Item:   fmt.Sprintf("`%s` provider syntax", rt),
		Detail: "verify nested structures are Blocks (`name { }`) vs object attributes (`name = { }`) and the import_id_format against the registry docs"})

	return rows
}

func collectAttrFlags(attrs []schema.AttributeDefinition, computed, sensitive *[]string) {
	for i := range attrs {
		a := &attrs[i]
		if a.Computed {
			*computed = append(*computed, a.TerraformName)
		}
		if a.Sensitive {
			*sensitive = append(*sensitive, a.TerraformName)
		}
		collectAttrFlags(a.NestedAttributes, computed, sensitive)
	}
}

// exportEvidenceRows renders the export-dependent checklist rows.
func exportEvidenceRows(in ChecklistInput) []ChecklistRow {
	if !in.LiveMode {
		return []ChecklistRow{{Status: StatusSkipped, Item: "Export evidence",
			Detail: "no live credentials — re-run with PINGCLI_PINGONE_* set"}}
	}

	var rows []ChecklistRow
	for _, rt := range in.Types {
		tfPath := in.ExportDir + "/" + in.ExportModuleDir + "/" + rt + ".tf"
		data, err := os.ReadFile(tfPath)
		switch {
		case err != nil:
			rows = append(rows, ChecklistRow{Status: StatusWarn, Item: fmt.Sprintf("`%s` exported", rt),
				Detail: tfPath + " missing from export output"})
		case len(strings.TrimSpace(string(data))) == 0:
			rows = append(rows, ChecklistRow{Status: StatusWarn, Item: fmt.Sprintf("`%s` exported", rt),
				Detail: "export file is empty — no instances in the environment or handler filtered everything"})
		default:
			rows = append(rows, ChecklistRow{Status: StatusPass, Item: fmt.Sprintf("`%s` exported", rt)})
		}
		if n := in.UnresolvedUUIDs[rt]; n > 0 {
			rows = append(rows, ChecklistRow{Status: StatusWarn, Item: fmt.Sprintf("`%s` unresolved UUIDs", rt),
				Detail: fmt.Sprintf("%d raw UUIDs in exported HCL — fix references_type config or add EmbeddedReferenceRule entries", n)})
		}
	}

	if len(in.FallbackVariables) > 0 {
		rows = append(rows, ChecklistRow{Status: StatusCheck, Item: "Fallback variables",
			Detail: "export emitted " + joinStrings(in.FallbackVariables) + " — explain in the PR why each target type is not yet exported"})
	}
	return rows
}

// grepDir reports whether any file under dir contains the needle.
func grepDir(dir, needle string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		path := dir + "/" + e.Name()
		if e.IsDir() {
			if grepDir(path, needle) {
				return true
			}
			continue
		}
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), needle) {
			return true
		}
	}
	return false
}
