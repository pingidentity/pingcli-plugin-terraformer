package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/compare"
)

// Report, FileReport, and DiffReport live in internal/compare/report.go —
// the JSON schema is a CI contract consumed by
// .github/workflows/regression.yaml's summary job.

func main() {
	baseDir := flag.String("base-dir", "", "Path to base branch export output directory")
	prDir := flag.String("pr-dir", "", "Path to PR branch export output directory")
	reportFile := flag.String("report-file", "", "Optional path to write JSON report")
	markdownFile := flag.String("markdown", "", "Optional path to write a grouped markdown report")
	htmlFile := flag.String("html", "", "Optional path to write a self-contained HTML report")
	flag.Parse()

	if *baseDir == "" || *prDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: regression-compare --base-dir <path> --pr-dir <path> [--report-file <path>] [--markdown <path>] [--html <path>]\n")
		os.Exit(2)
	}

	// Validate directories exist
	for _, d := range []struct{ name, path string }{{"base-dir", *baseDir}, {"pr-dir", *prDir}} {
		info, err := os.Stat(d.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s %q: %v\n", d.name, d.path, err)
			os.Exit(2)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: %s %q is not a directory\n", d.name, d.path)
			os.Exit(2)
		}
	}

	result, err := compare.CompareDirectories(*baseDir, *prDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Comparison error: %v\n", err)
		os.Exit(2)
	}

	// Build report
	report := buildReport(*baseDir, *prDir, result)

	// Print human-readable summary
	printSummary(report)

	// Write JSON report if requested
	if *reportFile != "" {
		if writeErr := writeJSONReport(*reportFile, report); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing report: %v\n", writeErr)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "\nReport written to: %s\n", *reportFile)
	}

	// Write friendly reports if requested.
	if *markdownFile != "" {
		if writeErr := os.WriteFile(*markdownFile, []byte(compare.RenderMarkdown(report)), 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing markdown report: %v\n", writeErr)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "Markdown report written to: %s\n", *markdownFile)
	}
	if *htmlFile != "" {
		html, htmlErr := compare.RenderHTML(report)
		if htmlErr != nil {
			fmt.Fprintf(os.Stderr, "Error rendering HTML report: %v\n", htmlErr)
			os.Exit(2)
		}
		if writeErr := os.WriteFile(*htmlFile, []byte(html), 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing HTML report: %v\n", writeErr)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "HTML report written to: %s\n", *htmlFile)
	}

	// Exit code: 0 = no breaking changes, 1 = breaking changes found
	if report.HasBreaking {
		os.Exit(1)
	}
}

func buildReport(baseDir, prDir string, result *compare.DirectoryResult) *compare.Report {
	report := &compare.Report{
		BaseDir:     baseDir,
		PRDir:       prDir,
		HasBreaking: result.HasBreakingDiffs(),
	}

	for _, f := range result.Files {
		fr := compare.FileReport{
			Path: f.Path,
			Kind: string(f.Kind),
		}

		switch f.Kind {
		case compare.FileMissing:
			// Missing file = breaking
			fr.BreakingDiffs = append(fr.BreakingDiffs, compare.DiffReport{
				Kind:     "missing_file",
				Resource: f.Path,
			})
			report.BreakingCount++
		case compare.FileExtra:
			// Extra file = acceptable
			fr.AcceptableDiffs = append(fr.AcceptableDiffs, compare.DiffReport{
				Kind:     "extra_file",
				Resource: f.Path,
			})
			report.AcceptableCount++
		case compare.FileCompared:
			if f.Result != nil {
				for _, d := range f.Result.Diffs {
					dr := compare.DiffReport{
						Resource:  d.Resource,
						Kind:      string(d.Kind),
						Attribute: d.Attribute,
						Expected:  d.Expected,
						Actual:    d.Actual,
					}
					severity := compare.ClassifyDiff(d)
					if severity == compare.SeverityBreaking {
						fr.BreakingDiffs = append(fr.BreakingDiffs, dr)
						report.BreakingCount++
					} else {
						fr.AcceptableDiffs = append(fr.AcceptableDiffs, dr)
						report.AcceptableCount++
					}
				}
			}
		}

		report.Files = append(report.Files, fr)
	}

	return report
}

func printSummary(report *compare.Report) {
	fmt.Println("=== Regression Comparison Report ===")
	fmt.Printf("Base: %s\n", report.BaseDir)
	fmt.Printf("PR:   %s\n", report.PRDir)
	fmt.Println()

	if !report.HasBreaking && report.AcceptableCount == 0 {
		fmt.Println("No differences detected.")
		return
	}

	// Print breaking changes
	if report.BreakingCount > 0 {
		fmt.Printf("BREAKING CHANGES (%d):\n", report.BreakingCount)
		for _, f := range report.Files {
			for _, d := range f.BreakingDiffs {
				printDiff(f.Path, d)
			}
		}
		fmt.Println()
	}

	// Print acceptable changes
	if report.AcceptableCount > 0 {
		fmt.Printf("Acceptable additions (%d):\n", report.AcceptableCount)
		for _, f := range report.Files {
			for _, d := range f.AcceptableDiffs {
				printDiff(f.Path, d)
			}
		}
		fmt.Println()
	}

	// Final verdict
	if report.HasBreaking {
		fmt.Println("RESULT: FAIL - Breaking changes detected. Review required.")
	} else {
		fmt.Println("RESULT: PASS - Only acceptable additions found.")
	}
}

func printDiff(filePath string, d compare.DiffReport) {
	switch d.Kind {
	case "missing_file":
		fmt.Printf("  - [%s] FILE REMOVED\n", filePath)
	case "extra_file":
		fmt.Printf("  + [%s] FILE ADDED\n", filePath)
	case string(compare.DiffMissingResource):
		fmt.Printf("  - [%s] Resource removed: %s\n", filePath, d.Resource)
	case string(compare.DiffExtraResource):
		fmt.Printf("  + [%s] Resource added: %s\n", filePath, d.Resource)
	case string(compare.DiffMissingAttribute):
		fmt.Printf("  - [%s] %s: attribute %q removed\n", filePath, d.Resource, d.Attribute)
	case string(compare.DiffExtraAttribute):
		fmt.Printf("  + [%s] %s: attribute %q added\n", filePath, d.Resource, d.Attribute)
	case string(compare.DiffValueMismatch):
		fmt.Printf("  ~ [%s] %s: %q changed: %q -> %q\n", filePath, d.Resource, d.Attribute, d.Expected, d.Actual)
	case string(compare.DiffMissingBlock):
		fmt.Printf("  - [%s] %s: block %q removed\n", filePath, d.Resource, d.Attribute)
	case string(compare.DiffBlockMismatch):
		fmt.Printf("  ~ [%s] %s: block %q changed\n", filePath, d.Resource, d.Attribute)
	}
}

func writeJSONReport(path string, report *compare.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
