// Package export runs the schema-driven export pipeline: orchestrator →
// variable extraction → formatting → module generation. It is shared by the
// pingcli export command and the resource review-bundle tool, so both use
// one implementation of the export tail (the review tool passes a
// pre-filtered registry to export a dependency-closed subset).
package export

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/definitions"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/filter"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/module"
	pingoneplatform "github.com/pingidentity/pingcli-plugin-terraformer/internal/platform/pingone"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/variables"
)

// Logger is the subset of grpc.Logger the pipeline emits through. Declared
// locally so the package does not force the full plugin logger interface on
// embedders; grpc.Logger satisfies it.
type Logger interface {
	Message(message string, fields map[string]string) error
	Warn(message string, fields map[string]string) error
}

// RunOptions configures one export run.
type RunOptions struct {
	// Registry holds the definitions to export. Callers wanting the full
	// set pass a registry loaded from definitions.FS; the review tool passes
	// a dependency-closed subset.
	Registry *schema.Registry

	// Client is the constructed platform API client.
	Client *pingoneplatform.Client

	// Logger receives progress and warning messages.
	Logger Logger

	// EnvironmentID is the PingOne environment to export.
	EnvironmentID string

	// OutputDir receives the generated module ("." when empty).
	OutputDir string

	// ModuleDir is the child module directory name.
	ModuleDir string

	// ModuleName prefixes root-module files and the module block.
	ModuleName string

	// IncludeImports generates import blocks in the root module.
	IncludeImports bool

	// IncludeValues populates variable values in module.tf.
	IncludeValues bool

	// OutputFormat is "hcl" or "tfjson".
	OutputFormat string

	// SkipDependencies skips dependency resolution.
	SkipDependencies bool

	// IncludeResources / ExcludeResources are glob/regex patterns matching
	// resource_type.label addresses (applied after listing).
	IncludeResources []string
	ExcludeResources []string

	// ListOnly prints resource addresses to stdout instead of generating a
	// module.
	ListOnly bool

	// IncludeUpstream expands the filter to upstream dependencies.
	IncludeUpstream bool

	// OutputAttributes / OutputAttributeFile collect output-block paths.
	OutputAttributes    []string
	OutputAttributeFile string
}

// Run executes the full export pipeline and generates the module.
func Run(ctx context.Context, opts RunOptions) error {
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "."
	}

	if err := opts.Logger.Message(fmt.Sprintf("Generating Terraform module in: %s/%s", outputDir, opts.ModuleDir), nil); err != nil {
		return fmt.Errorf("failed to log message: %w", err)
	}

	// 1. Definitions come from the caller's registry (loaded by the caller
	// so restricted registries are possible).

	// 2. Create custom handler registry and load platform-specific handlers.
	customReg := core.NewCustomHandlerRegistry()
	pingoneplatform.RegisterCustomHandlers(customReg)

	// 3. Create processor.
	proc := core.NewProcessor(opts.Registry, core.WithCustomHandlers(customReg))

	// 4. Build resource filter from include/exclude patterns.
	var resourceFilter *filter.ResourceFilter
	if len(opts.IncludeResources) > 0 || len(opts.ExcludeResources) > 0 {
		var err error
		resourceFilter, err = filter.NewResourceFilter(opts.IncludeResources, opts.ExcludeResources)
		if err != nil {
			return fmt.Errorf("invalid resource filter pattern: %w", err)
		}
	}

	// 5. Create and run orchestrator.
	embeddedRefs := pingoneplatform.NewEmbeddedReferenceRegistry()
	orch := core.NewExportOrchestrator(opts.Registry, proc, opts.Client, core.WithProgressFunc(func(msg string) {
		_ = opts.Logger.Message(msg, nil)
	}), core.WithEmbeddedReferences(embeddedRefs))

	result, err := orch.Export(ctx, core.ExportOptions{
		SkipDependencies: opts.SkipDependencies,
		GenerateImports:  opts.IncludeImports,
		EnvironmentID:    opts.EnvironmentID,
		ResourceFilter:   resourceFilter,
		ListOnly:         opts.ListOnly,
		IncludeUpstream:  opts.IncludeUpstream,
	})
	if err != nil {
		return fmt.Errorf("orchestrator export failed: %w", err)
	}

	// Log any warnings collected by the API client during export.
	for _, w := range opts.Client.Warnings() {
		_ = opts.Logger.Warn(w, nil)
	}

	// Handle list-only mode: print resource addresses and exit.
	// Addresses are enumerated output, so they go to stdout (not the logger)
	// to remain pipeable — matching list-outputs behaviour.
	if opts.ListOnly {
		if err := WriteResourceAddresses(os.Stdout, result); err != nil {
			return err
		}
		return nil
	}

	return generateModule(result, opts, outputDir)
}

// generateModule formats orchestrator results and generates the module files.
// Split from Run so the orchestrator phase is testable in isolation.
func generateModule(result *core.ExportResult, opts RunOptions, outputDir string) error {
	outFmt, fmtErr := formatters.NewFormatter(opts.OutputFormat)
	if fmtErr != nil {
		return fmtErr
	}
	varExtractor := variables.NewVariableExtractor(opts.Registry)

	var allOutputParts []string
	var allImportBlocks []module.ImportBlock
	var allVariables []module.Variable

	for _, erd := range result.ResourcesByType {
		// Extract variables FIRST and inject references into resource attributes
		// so the formatter renders var.X expressions instead of raw values.
		for _, rd := range erd.Resources {
			// Schema-driven variable extraction (variable_eligible attributes).
			extracted, extErr := varExtractor.Extract(erd.ResourceType, rd.Attributes, rd.Name)
			if extErr != nil {
				return fmt.Errorf("extract variables %s: %w", erd.ResourceType, extErr)
			}

			for _, ev := range extracted {
				allVariables = append(allVariables, module.Variable{
					Name:         ev.VariableName,
					Type:         ev.VariableType,
					Description:  ev.Description,
					Default:      ev.CurrentValue,
					Sensitive:    ev.Sensitive,
					IsSecret:     ev.IsSecret,
					ResourceType: ev.ResourceType,
					ResourceName: ev.ResourceName,
				})

				// Inject variable reference into resource attributes.
				// For type_discriminated_block (single-key map), inject inside the map.
				ref := core.ResolvedReference{
					IsVariable:   true,
					VariableName: ev.VariableName,
				}
				if ev.CurrentValue != nil {
					if s, ok := ev.CurrentValue.(string); ok {
						ref.OriginalValue = s
					} else {
						ref.OriginalValue = fmt.Sprintf("%v", ev.CurrentValue)
					}
				}

				if m, ok := rd.Attributes[ev.AttributePath].(map[string]interface{}); ok && len(m) == 1 {
					for k := range m {
						m[k] = ref
					}
				} else {
					rd.Attributes[ev.AttributePath] = ref
				}
			}

			// Custom-transform-provided variables (e.g., connector properties).
			// These already have var. references baked into their RawHCLValue — do NOT inject.
			for i := range rd.ExtractedVariables {
				ev := &rd.ExtractedVariables[i]
				allVariables = append(allVariables, module.Variable{
					Name:         ev.VariableName,
					Type:         ev.VariableType,
					Description:  ev.Description,
					Default:      ev.CurrentValue,
					Sensitive:    ev.Sensitive,
					IsSecret:     ev.IsSecret,
					ResourceType: ev.ResourceType,
					ResourceName: ev.ResourceName,
				})
			}
		}

		// Format resource output (now with variable references injected).
		formattedOutput, fmtErr := outFmt.FormatList(erd.Resources, erd.Definition, formatters.FormatOptions{
			SkipDependencies: opts.SkipDependencies,
			EnvironmentID:    opts.EnvironmentID,
		})
		if fmtErr != nil {
			return fmt.Errorf("format %s: %w", erd.ResourceType, fmtErr)
		}
		allOutputParts = append(allOutputParts, formattedOutput)

		// Generate import blocks.
		if opts.IncludeImports && erd.Definition.Dependencies.ImportIDFormat != "" {
			for _, rd := range erd.Resources {
				label := ImportResourceLabel(rd, erd.Definition)
				allImportBlocks = append(allImportBlocks, module.ImportBlock{
					To: fmt.Sprintf("module.%s.%s.%s", opts.ModuleName, erd.ResourceType, label),
					ID: BuildImportID(erd.Definition, rd, opts.EnvironmentID),
				})
			}
		}
	}

	// Include fallback variables from filter-excluded reference resolution.
	// These are references to resources that were excluded by the filter and
	// resolved to variable fallbacks with label-derived names.
	for _, fv := range result.FallbackVariables {
		allVariables = append(allVariables, module.Variable{
			Name:         fv.Name,
			Type:         fv.Type,
			Description:  fv.Description,
			ResourceType: fv.ResourceType,
			Default:      fv.Default,
		})
	}

	// Resolve --output-attribute / --output-attribute-file paths into output blocks.
	outputPaths, pathErr := CollectOutputPaths(opts.OutputAttributes, opts.OutputAttributeFile)
	if pathErr != nil {
		return pathErr
	}
	var allOutputs []module.Output
	if len(outputPaths) > 0 {
		allOutputs = BuildOutputs(outputPaths, result, opts.Logger)
	}

	// Build module structure.
	moduleConfig := module.ModuleConfig{
		OutputDir:      outputDir,
		ModuleDirName:  opts.ModuleDir,
		ModuleName:     opts.ModuleName,
		IncludeImports: opts.IncludeImports,
		IncludeValues:  opts.IncludeValues,
		EnvironmentID:  opts.EnvironmentID,
		OutputFormat:   opts.OutputFormat,
	}

	// Combine formatted output by resource type into ModuleResources.
	resources := buildModuleResources(result, allOutputParts)

	structure := &module.ModuleStructure{
		Config:       moduleConfig,
		Variables:    allVariables,
		Outputs:      allOutputs,
		Resources:    resources,
		ImportBlocks: allImportBlocks,
	}

	// Generate module files.
	generator := module.NewGenerator(moduleConfig)
	if err := generator.Generate(structure); err != nil {
		return fmt.Errorf("failed to generate module: %w", err)
	}

	if err := opts.Logger.Message(fmt.Sprintf("✓ Module successfully generated in: %s", outputDir), map[string]string{
		"module_dir":      opts.ModuleDir,
		"include_imports": fmt.Sprintf("%v", opts.IncludeImports),
		"include_values":  fmt.Sprintf("%v", opts.IncludeValues),
	}); err != nil {
		return fmt.Errorf("failed to log success: %w", err)
	}

	return nil
}

// LoadFullRegistry loads every embedded definition into a fresh registry —
// the standard full-export setup.
func LoadFullRegistry() (*schema.Registry, error) {
	registry := schema.NewRegistry()
	if err := registry.LoadFromFS(definitions.FS, "pingone"); err != nil {
		return nil, fmt.Errorf("failed to load definitions: %w", err)
	}
	return registry, nil
}

// buildModuleResources maps orchestrator results to module.ModuleResources.
// Each resource type key maps to concatenated HCL for that type.
func buildModuleResources(result *core.ExportResult, outputParts []string) module.ModuleResources {
	mr := make(module.ModuleResources)
	for i, erd := range result.ResourcesByType {
		if i >= len(outputParts) {
			break
		}
		mr[erd.ResourceType] += outputParts[i]
	}
	return mr
}

// ImportResourceLabel derives the sanitized resource label for import blocks,
// matching the logic used by the imports generator and HCL/tfjson formatters.
func ImportResourceLabel(data *core.ResourceData, def *schema.ResourceDefinition) string {
	if data.Label != "" {
		return data.Label
	}
	if len(def.API.LabelFields) > 0 {
		keys := make([]string, 0, len(def.API.LabelFields))
		for _, field := range def.API.LabelFields {
			if v, ok := data.Attributes[field]; ok && v != nil {
				if s, ok := v.(string); ok && s != "" {
					keys = append(keys, s)
				}
			}
		}
		if len(keys) > 0 {
			return utils.SanitizeMultiKeyResourceName(keys...)
		}
	}
	if data.Name != "" {
		return utils.SanitizeResourceName(data.Name)
	}
	return data.ID
}

// outputAttrPath holds a parsed --output-attribute path.
type outputAttrPath struct {
	resourceType string
	labelPattern string
	attrPath     string // dot-notation, e.g. "id" or "settings.csp"
}

// ParseOutputPath splits "resource_type.label_or_glob.attr" or
// "resource_type.label_or_glob.nested.attr" into its three logical parts.
// The first dot-segment is resource_type, the second is label, and everything
// after the second dot is the attribute path.
func ParseOutputPath(raw string) (outputAttrPath, bool) {
	parts := strings.SplitN(raw, ".", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return outputAttrPath{}, false
	}
	return outputAttrPath{
		resourceType: parts[0],
		labelPattern: parts[1],
		attrPath:     parts[2],
	}, true
}

// WriteResourceAddresses writes one "resource_type.label" address per line to
// w. Used by --list-resources; addresses are enumerated output and belong on
// stdout so they stay pipeable, matching the list-outputs command.
func WriteResourceAddresses(w io.Writer, result *core.ExportResult) error {
	buf := bufio.NewWriter(w)
	for _, erd := range result.ResourcesByType {
		for _, rd := range erd.Resources {
			if _, err := fmt.Fprintf(buf, "%s.%s\n", erd.ResourceType, rd.Label); err != nil {
				return fmt.Errorf("failed to write resource address: %w", err)
			}
		}
	}
	if err := buf.Flush(); err != nil {
		return fmt.Errorf("failed to flush resource addresses: %w", err)
	}
	return nil
}

// CollectOutputPaths merges --output-attribute values with lines read from
// --output-attribute-file, deduplicates, and returns the combined slice.
func CollectOutputPaths(flags []string, filePath string) ([]string, error) {
	seen := make(map[string]struct{})
	var paths []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") {
			return
		}
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			paths = append(paths, p)
		}
	}
	for _, f := range flags {
		add(f)
	}
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("cannot open --output-attribute-file %q: %w", filePath, err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			add(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading --output-attribute-file: %w", err)
		}
	}
	return paths, nil
}

// BuildOutputs converts collected path strings into module.Output values by
// matching each (resourceType, labelPattern, attrPath) against exported results.
func BuildOutputs(paths []string, result *core.ExportResult, logger Logger) []module.Output {
	// Index results by resource type for O(1) lookup.
	byType := make(map[string]*core.ExportedResourceData, len(result.ResourcesByType))
	for _, erd := range result.ResourcesByType {
		byType[erd.ResourceType] = erd
	}

	var outputs []module.Output
	for _, raw := range paths {
		p, ok := ParseOutputPath(raw)
		if !ok {
			_ = logger.Warn(fmt.Sprintf("skipping malformed --output-attribute path %q: expected resource_type.label.attr", raw), nil)
			continue
		}
		erd, found := byType[p.resourceType]
		if !found {
			_ = logger.Warn(fmt.Sprintf("--output-attribute: resource type %q not found in export results", p.resourceType), nil)
			continue
		}
		matched := 0
		for _, rd := range erd.Resources {
			ok, err := filepath.Match(strings.ToLower(p.labelPattern), strings.ToLower(rd.Label))
			if err != nil || !ok {
				continue
			}
			matched++
			name := p.resourceType + "__" + rd.Label + "__" + strings.ReplaceAll(p.attrPath, ".", "__")
			outputs = append(outputs, module.Output{
				Name:        name,
				Description: "The " + p.attrPath + " of " + p.resourceType + " " + rd.Label,
				Value:       p.resourceType + "." + rd.Label + "." + p.attrPath,
			})
		}
		if matched == 0 {
			_ = logger.Warn(fmt.Sprintf("--output-attribute: pattern %q matched no %s resources", p.labelPattern, p.resourceType), nil)
		}
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Name < outputs[j].Name })
	return outputs
}

// BuildImportID expands the definition's import ID format for a single resource.
func BuildImportID(def *schema.ResourceDefinition, rd *core.ResourceData, environmentID string) string {
	format := def.Dependencies.ImportIDFormat
	format = strings.ReplaceAll(format, "{env_id}", environmentID)
	format = strings.ReplaceAll(format, "{resource_id}", rd.ID)
	// Expand any remaining {attr_name} placeholders.
	for k, v := range rd.Attributes {
		if s, ok := v.(string); ok {
			format = strings.ReplaceAll(format, "{"+k+"}", s)
		} else if ref, ok := v.(core.ResolvedReference); ok && ref.OriginalValue != "" {
			format = strings.ReplaceAll(format, "{"+k+"}", ref.OriginalValue)
		}
	}
	return format
}
