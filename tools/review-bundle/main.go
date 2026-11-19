// Command review-bundle generates the resource review bundle: a
// human-reviewable report accompanying every resource-adding or
// resource-changing PR.
//
// Contents (into --out, default review-bundle/):
//
//	meta.json      — provenance (base ref, mode, environment ID; never credentials)
//	summary.md     — definition-driven summary tables + changed-fields report
//	checklist.md   — review checklist, auto-answered where mechanical
//	review.html    — single self-contained review page (no external assets)
//	export/        — restricted live export (when credentials are present)
//	snapshot/      — raw API JSON per reviewed type (when credentials are present)
//
// Resource types default to auto-detection from the git diff against
// --base-ref; pass --resource-types to review specific types without a diff.
// Without the PINGCLI_PINGONE_* environment variables the tool runs in
// dry-run mode: the definition-driven files are produced, the live export and
// snapshot are omitted, and checklist rows depending on them are marked
// SKIPPED. The caller provisions and destroys the environment (see
// tests/regression-provisioned/README.md); this tool never does.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pingoneplatform "github.com/pingidentity/pingcli-plugin-terraformer/internal/platform/pingone"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/export"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/review"
)

func main() {
	var (
		resourceTypes = flag.String("resource-types", "", "Comma-separated resource types to review (default: auto-detect from git diff against --base-ref)")
		baseRef       = flag.String("base-ref", "main", "Git ref to diff against for change detection")
		out           = flag.String("out", "review-bundle", "Output directory for the bundle")
		repoDir       = flag.String("repo-dir", ".", "Repository root (default: current directory)")
		detectOnly    = flag.Bool("detect-only", false, "Print detected types and closure as JSON and exit (no bundle)")
		moduleName    = flag.String("module-name", "review-bundle", "Generated module name")
		moduleDir     = flag.String("module-dir", "module", "Generated child-module directory name (under export/)")
	)
	flag.Parse()

	if err := run(*resourceTypes, *baseRef, *out, *repoDir, *detectOnly, *moduleName, *moduleDir); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(resourceTypes, baseRef, out, repoDir string, detectOnly bool, moduleName, moduleDir string) error {
	// 1. Determine reviewed types.
	var changes []review.TypeChange
	platformCoreChanged := false

	if resourceTypes != "" {
		// Explicit list: no diff, no field changes.
		for _, rt := range strings.Split(resourceTypes, ",") {
			rt = strings.TrimSpace(rt)
			if rt == "" {
				continue
			}
			changes = append(changes, review.TypeChange{ResourceType: rt})
		}
	} else {
		var err error
		changes, platformCoreChanged, err = review.DetectChangedTypes(review.DetectOptions{
			BaseRef: baseRef,
			RepoDir: repoDir,
		})
		if err != nil {
			return fmt.Errorf("detect changed types: %w", err)
		}
	}

	seeds := make([]string, 0, len(changes))
	for _, tc := range changes {
		if !tc.Removed {
			seeds = append(seeds, tc.ResourceType)
		}
	}

	embeddedRules := pingoneplatform.NewEmbeddedReferenceRegistry().Rules()

	// 2. Compute the dependency-closed review set.
	closure, err := review.ComputeClosure(seeds, embeddedRules)
	if err != nil {
		return fmt.Errorf("compute closure: %w", err)
	}

	if detectOnly {
		data, err := review.DetectOnlyJSON(seeds, closure)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	if len(closure.Types) == 0 {
		return fmt.Errorf("no reviewable types found (seeds: %v); pass --resource-types explicitly", seeds)
	}

	// 3. Load definitions for the summary — straight from the closure
	// registry, which holds exactly the reviewed set.
	defMap := map[string]*schema.ResourceDefinition{}
	for _, rt := range closure.Types {
		def, err := closure.Registry.Get(rt)
		if err != nil {
			return fmt.Errorf("get definition %s: %w", rt, err)
		}
		defMap[rt] = def
	}

	// 4. Live phase (skipped in dry-run).
	creds, live := review.CredentialsFromEnv(os.Getenv)
	exportDir := filepath.Join(out, "export")
	snapshotDir := filepath.Join(out, "snapshot")

	var fallbackVars []string
	unresolved := map[string]int{}
	mode := "dry-run"

	if live {
		mode = "live"
		ctx := context.Background()
		client, err := pingoneplatform.NewFromCredentials(ctx, creds.WorkerEnvironmentID, creds.ExportEnvironmentID, creds.RegionCode, creds.ClientID, creds.ClientSecret)
		if err != nil {
			return fmt.Errorf("create API client: %w", err)
		}

		// Restricted export through the shared pipeline: the filtered
		// registry avoids listing types whose service is absent (#163).
		if err := os.MkdirAll(exportDir, 0o755); err != nil {
			return fmt.Errorf("create export dir: %w", err)
		}
		if err := export.Run(ctx, export.RunOptions{
			Registry:      closure.Registry,
			Client:        client,
			Logger:        stderrLogger{},
			EnvironmentID: creds.EnvID(),
			OutputDir:     exportDir,
			ModuleDir:     moduleDir,
			ModuleName:    moduleName,
			OutputFormat:  "hcl",
			IncludeValues: true,
		}); err != nil {
			return fmt.Errorf("restricted export: %w", err)
		}

		// Snapshot raw API responses for the live review types.
		liveTypes := liveSubset(closure.Types, closure.SkippedLive)
		snap := &review.Snapshotter{Client: client, EnvironmentID: creds.EnvID()}
		snapshotted, err := snap.WriteSnapshots(ctx, snapshotDir, liveTypes)
		if err != nil {
			return fmt.Errorf("snapshot: %w", err)
		}
		_ = snapshotted

		// Post-export mechanical checks.
		fallbackVars = collectFallbackVars(exportDir)
		unresolved = countUnresolvedUUIDs(exportDir, moduleDir, closure.Types)
	}

	// 5. Render the bundle.
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("create bundle dir: %w", err)
	}

	summaryIn := review.SummaryInput{
		Defs:          defMap,
		AllTypes:      typeSet(closure.Types),
		Changes:       changes,
		EmbeddedRules: embeddedRules,
	}
	summary := review.RenderSummary(summaryIn)
	if err := os.WriteFile(filepath.Join(out, "summary.md"), []byte(summary), 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}

	checklist := review.RenderChecklist(review.ChecklistInput{
		Types:             closure.Types,
		Defs:              defMap,
		RepoDir:           repoDir,
		ExportDir:         exportDir,
		ExportModuleDir:   moduleDir,
		FallbackVariables: fallbackVars,
		UnresolvedUUIDs:   unresolved,
		EnvID:             creds.EnvID(),
		LiveMode:          live,
	})
	if err := os.WriteFile(filepath.Join(out, "checklist.md"), []byte(checklist), 0o644); err != nil {
		return fmt.Errorf("write checklist: %w", err)
	}

	// Export HCL per reviewed type for the HTML page (empty in dry-run).
	exports := map[string]string{}
	if live {
		for _, rt := range liveSubset(closure.Types, closure.SkippedLive) {
			data, err := os.ReadFile(filepath.Join(exportDir, moduleDir, rt+".tf"))
			if err == nil {
				exports[rt] = string(data)
			}
		}
	}

	// Snapshot JSON per reviewed type for the HTML page.
	snapshots := map[string]string{}
	if live {
		for _, rt := range liveSubset(closure.Types, closure.SkippedLive) {
			data, err := os.ReadFile(filepath.Join(snapshotDir, rt+".json"))
			if err == nil {
				snapshots[rt] = string(data)
			}
		}
	}

	meta := review.Meta{
		BaseRef:             baseRef,
		Mode:                mode,
		ResourceTypes:       closure.Types,
		Changes:             changes,
		PlatformCoreChanged: platformCoreChanged,
		SkippedLive:         closure.SkippedLive,
	}
	if live {
		meta.TargetEnvironmentID = creds.EnvID()
		meta.Region = creds.RegionCode
	}
	metaData, err := review.WriteMeta(meta)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "meta.json"), metaData, 0o644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}

	html, err := review.RenderHTML(review.HTMLPageData{
		Title:     "Resource Review Bundle",
		Meta:      review.DescribeClosure(closure) + " · mode: " + mode,
		Types:     closure.Types,
		Summary:   summary,
		Checklist: checklist,
		Exports:   exports,
		Snapshots: snapshots,
	})
	if err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	if err := os.WriteFile(filepath.Join(out, "review.html"), []byte(html), 0o644); err != nil {
		return fmt.Errorf("write html: %w", err)
	}

	// 6. Secret-scan guard: the bundle must never contain credential values.
	if live {
		if err := review.SecretScanGuard(out, creds.ClientSecret); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Review bundle written to %s (mode: %s, types: %s)\n",
		out, mode, strings.Join(closure.Types, ", "))
	return nil
}

// loadDefinitions loads each review type's definition from the embedded FS.
// (Superseded: definitions come from the closure registry.)

// liveSubset filters out types skipped from the live phase.
func liveSubset(types []string, skipped map[string]string) []string {
	var out []string
	for _, rt := range types {
		if _, is := skipped[rt]; !is {
			out = append(out, rt)
		}
	}
	return out
}

// typeSet converts a slice to a set.
func typeSet(types []string) map[string]bool {
	m := make(map[string]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}

// fallbackVarName matches generated variable declarations in variables.tf.
var fallbackVarName = regexp.MustCompile(`(?m)^variable "([^"]+)"`)

// collectFallbackVars lists variable names declared in the export's
// variables.tf (root-module fallback variables included).
func collectFallbackVars(exportDir string) []string {
	data, err := os.ReadFile(filepath.Join(exportDir, "review-bundle-variables.tf"))
	if err != nil {
		// Module-name prefixes vary; also try the generic root variables file.
		data, err = os.ReadFile(filepath.Join(exportDir, "variables.tf"))
		if err != nil {
			return nil
		}
	}
	var out []string
	for _, m := range fallbackVarName.FindAllStringSubmatch(string(data), -1) {
		out = append(out, m[1])
	}
	return out
}

// unresolvedUUIDPattern is the Phase 5b check from the new-resource skill.
var unresolvedUUIDPattern = regexp.MustCompile(`"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"`)

// countUnresolvedUUIDs counts raw quoted UUIDs per reviewed type's generated
// HCL file.
func countUnresolvedUUIDs(exportDir, moduleDir string, types []string) map[string]int {
	out := map[string]int{}
	for _, rt := range types {
		data, err := os.ReadFile(filepath.Join(exportDir, moduleDir, rt+".tf"))
		if err != nil {
			continue
		}
		out[rt] = len(unresolvedUUIDPattern.FindAll(data, -1))
	}
	return out
}

// stderrLogger adapts the pipeline's Logger to stderr output.
type stderrLogger struct{}

func (stderrLogger) Message(msg string, fields map[string]string) error {
	fmt.Fprintln(os.Stderr, msg)
	return nil
}

func (stderrLogger) Warn(msg string, fields map[string]string) error {
	fmt.Fprintln(os.Stderr, "Warning:", msg)
	return nil
}
