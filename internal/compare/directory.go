package compare

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileKind describes how a file was handled during directory comparison.
type FileKind string

const (
	// FileMissing means the file was in base but not in PR.
	FileMissing FileKind = "missing_file"

	// FileExtra means the file was in PR but not in base.
	FileExtra FileKind = "extra_file"

	// FileCompared means the file was found in both and compared.
	FileCompared FileKind = "compared"
)

// FileResult holds the comparison result for a single file.
type FileResult struct {
	Path   string
	Result *Result
	Kind   FileKind
}

// DirectoryResult holds the outcome of a directory comparison.
type DirectoryResult struct {
	Files []FileResult
}

// HasBreakingDiffs returns true if any file is missing or has breaking content diffs.
func (r *DirectoryResult) HasBreakingDiffs() bool {
	for _, f := range r.Files {
		if f.Kind == FileMissing {
			return true
		}
		if f.Kind == FileCompared && f.Result != nil && f.Result.HasBreakingDiffs() {
			return true
		}
	}
	return false
}

// Summary returns a human-readable summary of directory comparison results.
func (r *DirectoryResult) Summary() string {
	if len(r.Files) == 0 {
		return "no files compared"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d file(s) compared:\n", len(r.Files)))
	for _, f := range r.Files {
		switch f.Kind {
		case FileMissing:
			sb.WriteString(fmt.Sprintf("  - MISSING %s\n", f.Path))
		case FileExtra:
			sb.WriteString(fmt.Sprintf("  + EXTRA   %s\n", f.Path))
		case FileCompared:
			if f.Result != nil && f.Result.HasDiffs() {
				sb.WriteString(fmt.Sprintf("  ~ %s: %s", f.Path, f.Result.Summary()))
			} else {
				sb.WriteString(fmt.Sprintf("  = %s: identical\n", f.Path))
			}
		}
	}
	return sb.String()
}

// CompareDirectories walks two directory trees and compares files.
func CompareDirectories(baseDir, prDir string) (*DirectoryResult, error) {
	baseFiles, err := collectFiles(baseDir)
	if err != nil {
		return nil, fmt.Errorf("walk base dir: %w", err)
	}
	prFiles, err := collectFiles(prDir)
	if err != nil {
		return nil, fmt.Errorf("walk pr dir: %w", err)
	}

	result := &DirectoryResult{}

	// Check files in base.
	for _, rel := range sortedMapKeys(baseFiles) {
		prPath, exists := prFiles[rel]
		if !exists {
			result.Files = append(result.Files, FileResult{
				Path: rel,
				Kind: FileMissing,
			})
			continue
		}
		basePath := baseFiles[rel]
		cmpResult, err := compareFile(basePath, prPath, rel)
		if err != nil {
			return nil, fmt.Errorf("compare %s: %w", rel, err)
		}
		result.Files = append(result.Files, FileResult{
			Path:   rel,
			Kind:   FileCompared,
			Result: cmpResult,
		})
	}

	// Check for extra files in PR.
	for _, rel := range sortedMapKeys(prFiles) {
		if _, exists := baseFiles[rel]; !exists {
			result.Files = append(result.Files, FileResult{
				Path: rel,
				Kind: FileExtra,
			})
		}
	}

	return result, nil
}

func collectFiles(dir string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[rel] = path
		return nil
	})
	return files, err
}

func compareFile(basePath, prPath, relPath string) (*Result, error) {
	baseContent, err := os.ReadFile(basePath)
	if err != nil {
		return nil, err
	}
	prContent, err := os.ReadFile(prPath)
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasSuffix(relPath, ".tf.json"):
		return CompareJSON(string(baseContent), string(prContent))
	case strings.HasSuffix(relPath, ".tf"), strings.HasSuffix(relPath, ".tfvars"):
		return CompareHCLGeneric(string(baseContent), string(prContent))
	default:
		// Byte comparison for other files.
		result := &Result{}
		if string(baseContent) != string(prContent) {
			// Truncate large content to avoid memory pressure in reports.
			expStr := truncate(string(baseContent), 200)
			actStr := truncate(string(prContent), 200)
			result.Diffs = append(result.Diffs, Diff{
				Resource:  relPath,
				Kind:      DiffValueMismatch,
				Attribute: "content",
				Expected:  expStr,
				Actual:    actStr,
			})
		}
		return result, nil
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}
