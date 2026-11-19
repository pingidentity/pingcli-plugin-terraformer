// Package review generates the resource review bundle: a human-reviewable
// report accompanying every resource-adding or resource-changing PR. It
// detects which resource types (and which definition fields) changed between
// a base ref and the working tree, renders a definition-driven summary and
// review checklist, produces a restricted live export and raw API snapshot
// when credentials are available, and packages everything into one
// self-contained HTML page.
package review

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// TypeChange describes everything detected about one resource type's change.
type TypeChange struct {
	// ResourceType is the resource_type from the definition ("" when only a
	// handler file changed and the type could not be recovered from it).
	ResourceType string `json:"resource_type"`

	// IsNew is true when the definition file does not exist in the base ref.
	IsNew bool `json:"is_new"`

	// Removed is true when the definition file exists in the base ref but not
	// in the working tree.
	Removed bool `json:"removed"`

	// HandlerChanged is true when the type's Go handler file changed.
	HandlerChanged bool `json:"handler_changed"`

	// DefinitionPath is the worktree-relative path of the definition YAML.
	DefinitionPath string `json:"definition_path"`

	// FieldChanges lists the definition-level and per-attribute changes.
	FieldChanges []FieldChange `json:"field_changes,omitempty"`
}

// FieldChange is one semantic difference between the base and worktree
// definitions. For attribute-level changes, Attr is the attribute's
// terraform_name ("" for section-level changes) and Section names the
// definition section the change lives in.
type FieldChange struct {
	Section string `json:"section"` // metadata | api | attributes | dependencies | variables | custom_handlers | conditional_defaults
	Attr    string `json:"attr,omitempty"`
	Kind    string `json:"kind"` // added | removed | changed
	Field   string `json:"field,omitempty"`
	Base    string `json:"base,omitempty"`
	PR      string `json:"pr,omitempty"`
}

// DetectOptions configures change detection.
type DetectOptions struct {
	// BaseRef is the git ref to diff against (branch name or SHA).
	BaseRef string

	// RepoDir is the working directory for git invocations.
	RepoDir string
}

// gitFile lists files under the given paths that differ between baseRef and
// the worktree (or HEAD when the worktree is clean).
func gitDiffFiles(repoDir, baseRef string, paths ...string) ([]string, error) {
	args := []string{"diff", "--name-only", baseRef + "...HEAD", "--"}
	args = append(args, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only %s: %w", baseRef, err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// gitShowFile reads a file's contents at baseRef. Returns ("", false) when
// the file does not exist at baseRef (e.g. a newly added definition).
func gitShowFile(repoDir, baseRef, path string) (string, bool, error) {
	cmd := exec.Command("git", "show", baseRef+":"+path)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if len(out) == 0 {
			if e, ok := err.(*exec.ExitError); ok {
				exitErr = e
				_ = exitErr
			}
		}
		// git exits 1 with empty output when the path is absent at the ref.
		if len(out) == 0 {
			return "", false, nil
		}
		return "", false, fmt.Errorf("git show %s:%s: %w", baseRef, path, err)
	}
	return string(out), true, nil
}

// readWorktreeFile reads a file from the working tree.
func readWorktreeFile(repoDir, path string) (string, bool, error) {
	data, err := os.ReadFile(repoDir + "/" + path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), true, nil
}

// parseDefinition parses YAML text into a ResourceDefinition without
// validation (the bundle reports on definitions as authored, and validation
// is the export pipeline's job).
func parseDefinition(text string) (*schema.ResourceDefinition, error) {
	var def schema.ResourceDefinition
	if err := yaml.Unmarshal([]byte(text), &def); err != nil {
		return nil, fmt.Errorf("parse definition YAML: %w", err)
	}
	return &def, nil
}

// resourceTypeFromHandler extracts the registerResource("<type>" argument
// from handler source text. Returns "" when no registration is present.
func resourceTypeFromHandler(source string) string {
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "registerResource(") || strings.Contains(trimmed, "registerResource(\"") {
			if i := strings.Index(trimmed, "registerResource(\""); i >= 0 {
				rest := trimmed[i+len("registerResource(\""):]
				if j := strings.Index(rest, "\""); j >= 0 {
					return rest[:j]
				}
			}
		}
	}
	return ""
}

// platformCoreFiles are platform-package files whose change affects the
// export pipeline as a whole rather than one resource type.
var platformCoreFiles = map[string]bool{
	"internal/platform/pingone/client.go":   true,
	"internal/platform/pingone/dispatch.go": true,
}

// DetectChangedTypes maps changed definition and handler files between
// baseRef and the worktree to resource types, and computes semantic field
// diffs for each changed definition. When platform-core files changed, the
// returned PlatformCoreChanged flag is set and the caller widens the review
// set to all non-DaVinci types.
func DetectChangedTypes(opts DetectOptions) (changes []TypeChange, platformCoreChanged bool, err error) {
	files, err := gitDiffFiles(opts.RepoDir, opts.BaseRef, "definitions/", "internal/platform/")
	if err != nil {
		return nil, false, err
	}

	type acc struct {
		tc *TypeChange
	}
	byType := map[string]*acc{}
	var order []string

	get := func(rt, defPath string) *TypeChange {
		if a, ok := byType[rt]; ok {
			return a.tc
		}
		tc := &TypeChange{ResourceType: rt, DefinitionPath: defPath}
		byType[rt] = &acc{tc}
		order = append(order, rt)
		return tc
	}

	// definitionPathByType lets handler changes join an existing type entry
	// and new entries find their definition path later.
	defPathByType := map[string]string{}

	for _, f := range files {
		switch {
		case strings.HasPrefix(f, "definitions/") && (strings.HasSuffix(f, ".yaml") || strings.HasSuffix(f, ".yml")):
			prText, prExists, err := readWorktreeFile(opts.RepoDir, f)
			if err != nil {
				return nil, false, err
			}
			baseText, baseExists, err := gitShowFile(opts.RepoDir, opts.BaseRef, f)
			if err != nil {
				return nil, false, err
			}

			var rt string
			switch {
			case prExists:
				def, err := parseDefinition(prText)
				if err != nil {
					return nil, false, fmt.Errorf("%s: %w", f, err)
				}
				rt = def.Metadata.ResourceType
			case baseExists:
				def, err := parseDefinition(baseText)
				if err != nil {
					return nil, false, fmt.Errorf("%s (base): %w", f, err)
				}
				rt = def.Metadata.ResourceType
			default:
				continue
			}
			if rt == "" {
				continue
			}

			tc := get(rt, f)
			tc.DefinitionPath = f
			defPathByType[rt] = f

			switch {
			case baseExists && prExists:
				baseDef, err := parseDefinition(baseText)
				if err != nil {
					return nil, false, fmt.Errorf("%s (base): %w", f, err)
				}
				prDef, err := parseDefinition(prText)
				if err != nil {
					return nil, false, fmt.Errorf("%s: %w", f, err)
				}
				tc.FieldChanges = DiffDefinitions(baseDef, prDef)
			case prExists:
				tc.IsNew = true
			case baseExists:
				tc.Removed = true
			}

		case strings.HasPrefix(f, "internal/platform/") && strings.HasSuffix(f, "_test.go"):
			// Test-only changes carry no resource review surface.

		case strings.HasPrefix(f, "internal/platform/"):
			if platformCoreFiles[f] {
				platformCoreChanged = true
				continue
			}
			var source string
			if text, ok, err := readWorktreeFile(opts.RepoDir, f); ok && err == nil {
				source = text
			} else if text, ok, err := gitShowFile(opts.RepoDir, "HEAD", f); ok && err == nil {
				source = text
			}
			if rt := resourceTypeFromHandler(source); rt != "" {
				tc := get(rt, defPathByType[rt])
				tc.HandlerChanged = true
				if tc.DefinitionPath == "" {
					// Handler changed with no definition change: find the
					// definition path from the full definition set.
					if p, err := findDefinitionPath(opts.RepoDir, rt); err == nil {
						tc.DefinitionPath = p
						defPathByType[rt] = p
					}
				}
			}
		}
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	for _, rt := range order {
		changes = append(changes, *byType[rt].tc)
	}
	return changes, platformCoreChanged, nil
}

// findDefinitionPath locates the definition YAML declaring resourceType by
// scanning the definitions tree in the worktree.
func findDefinitionPath(repoDir, resourceType string) (string, error) {
	matches, err := walkDefinitionFiles(repoDir)
	if err != nil {
		return "", err
	}
	for _, path := range matches {
		text, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var def schema.ResourceDefinition
		if err := yaml.Unmarshal(text, &def); err != nil {
			continue
		}
		if def.Metadata.ResourceType == resourceType {
			rel, err := relPath(repoDir, path)
			if err != nil {
				return "", err
			}
			return rel, nil
		}
	}
	return "", fmt.Errorf("no definition found for %s", resourceType)
}

func walkDefinitionFiles(repoDir string) ([]string, error) {
	var out []string
	root := repoDir + "/definitions"
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		files, err := os.ReadDir(root + "/" + e.Name())
		if err != nil {
			continue
		}
		for _, f := range files {
			name := f.Name()
			if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
				out = append(out, root+"/"+e.Name()+"/"+name)
			}
		}
	}
	return out, nil
}

func relPath(repoDir, path string) (string, error) {
	rel, err := filepath.Rel(repoDir, path)
	if err != nil {
		return "", err
	}
	return rel, nil
}
