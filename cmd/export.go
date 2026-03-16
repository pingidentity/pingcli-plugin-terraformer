// Copyright © 2025 Ping Identity Corporation

// Package cmd provides the command implementation for the Terraform converter.
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/pingidentity/pingcli-plugin-terraformer/definitions"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/api"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/filter"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/module"
	pingoneplatform "github.com/pingidentity/pingcli-plugin-terraformer/internal/platform/pingone"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/variables"
	"github.com/pingidentity/pingcli/shared/grpc"
	"github.com/spf13/pflag"
)

// Command metadata for the export subcommand
var (
	// ExportExample provides usage examples for the command
	ExportExample = `  # Export PingOne resources to Terraform
  pingcli tf export \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --pingone-region-code NA \
    --out ./environment.tf

  # Export from different environment than worker app
  pingcli tf export \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-export-environment-id <target-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --pingone-region-code NA \
    --out ./environment.tf

  # Export without Terraform dependencies (raw UUIDs)
  pingcli tf export \
    --pingone-worker-environment-id <uuid> \
    --skip-dependencies

  # List all resource addresses
  pingcli tf export \
    --pingone-worker-environment-id <uuid> \
    --list-resources

  # Export only DaVinci flow resources
  pingcli tf export \
    --include-resources "pingone_davinci_flow*" \
    --out ./output

  # Export everything except test resources
  pingcli tf export \
    --exclude-resources "*Test*" \
    --out ./output

  # Use environment variables for credentials
  export PINGCLI_PINGONE_ENVIRONMENT_ID="..."
  export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID="..."
  export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET="..."
  export PINGCLI_PINGONE_REGION_CODE="NA"
  pingcli tf export --out ./environment.tf`

	// ExportLong provides a detailed description of the command
	ExportLong = `Export PingOne resources to Terraform configuration.

Connects to PingOne APIs and exports all registered resource types for the target environment.

The generated output includes proper Terraform resource references and dependency ordering.

Output formats:
  • hcl    - Terraform HCL syntax (.tf files, default)
  • tfjson - Terraform JSON configuration syntax (.tf.json files)

Authentication can be provided via flags or environment variables:
  PINGCLI_PINGONE_ENVIRONMENT_ID                    - Environment containing the worker app
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID      - Worker app client ID
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET   - Worker app client secret
  PINGCLI_PINGONE_REGION_CODE                       - Region code (AP, AU, CA, EU, NA)
  PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID  - Target environment to export (optional, defaults to worker environment)

Resource Filtering:
  Use --include-resources and --exclude-resources to filter resources by pattern.
  Patterns match against: resource_type.terraform_label (e.g., pingone_davinci_flow.pingcli__Login)
  Glob wildcards (* and ?) are supported. Use 'regex:' prefix for regex patterns.
  Matching is case-insensitive. Multiple patterns combine via OR (union).
  Use --list-resources to discover available resource addresses before filtering.`

	// ExportShort provides a brief, one-line description of the command
	ExportShort = "Export Ping Identity resources to Terraform configuration"

	// ExportUse defines the command's name and its arguments/flags syntax
	ExportUse = "export [flags]"
)

// ExportCommand is the implementation of the export subcommand.
// It encapsulates the logic for exporting PingOne environments to Terraform.
type ExportCommand struct{}

// A compile-time check to ensure ExportCommand correctly implements the
// grpc.PingCliCommand interface.
var _ grpc.PingCliCommand = (*ExportCommand)(nil)

// Configuration is called by the pingcli host to retrieve the command's
// metadata, such as its name, description, and usage examples.
func (c *ExportCommand) Configuration() (*grpc.PingCliCommandConfiguration, error) {
	cmdConfig := &grpc.PingCliCommandConfiguration{
		Example: ExportExample,
		Long:    ExportLong,
		Short:   ExportShort,
		Use:     ExportUse,
	}

	return cmdConfig, nil
}

// Run is the execution entry point for the export subcommand.
// It parses flags and executes the export logic.
func (c *ExportCommand) Run(args []string, logger grpc.Logger) error {
	// Create a new FlagSet for parsing command-line flags
	flags := pflag.NewFlagSet("export", pflag.ContinueOnError)

	// Define API export flags matching Ping CLI standards
	workerEnvironmentID := flags.String("pingone-worker-environment-id", "", "PingOne environment ID containing the worker app")
	exportEnvironmentID := flags.String("pingone-export-environment-id", "", "PingOne environment ID to export resources from (defaults to worker environment)")
	regionCode := flags.String("pingone-region-code", "", "PingOne region code (NA, EU, AP, CA, AU)")
	clientID := flags.String("pingone-worker-client-id", "", "OAuth worker app client ID")
	clientSecret := flags.String("pingone-worker-client-secret", "", "OAuth worker app client secret")
	out := flags.StringP("out", "o", "", "Output file path (default: stdout)")
	skipDependencies := flags.Bool("skip-dependencies", false, "Skip dependency resolution")
	skipImports := flags.Bool("skip-imports", false, "Skip generating Terraform import blocks (imports generated by default, requires Terraform 1.5+)")

	// Module generation flags (module mode is always enabled)
	moduleDir := flags.String("module-dir", "ping-export-module", "Name of the child module directory")
	moduleName := flags.String("module-name", "ping-export", "Used to define Terraform module and prefix generated content (default \"ping-export\")")
	includeImports := flags.Bool("include-imports", false, "Generate import blocks in root module")
	includeValues := flags.Bool("include-values", false, "Populate variable values in module.tf from export")
	// Output format selection.
	outputFormat := flags.String("output-format", formatters.FormatHCL, "Output format: hcl (default) or tfjson (Terraform JSON configuration syntax)")

	// Resource filtering flags
	includeResources := flags.StringSlice("include-resources", []string{}, "Include resources matching glob pattern(s) (repeatable). Use 'regex:' prefix for regex. Pattern matches against resource_type.terraform_label")
	excludeResources := flags.StringSlice("exclude-resources", []string{}, "Exclude resources matching glob pattern(s) (repeatable). Use 'regex:' prefix for regex. Pattern matches against resource_type.terraform_label")
	listResources := flags.Bool("list-resources", false, "List all resource addresses (resource_type.terraform_label) and exit")

	// Parse the provided arguments
	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate output format early.
	switch *outputFormat {
	case formatters.FormatHCL, formatters.FormatTFJSON:
		// valid
	default:
		return fmt.Errorf("unsupported output format: %s. Supported: hcl, tfjson", *outputFormat)
	}

	// Execute export (invert skipImports to get generateImports)
	return c.runExport(logger, *workerEnvironmentID, *exportEnvironmentID, *regionCode, *clientID, *clientSecret, *out, *skipDependencies, !*skipImports, *moduleDir, *moduleName, *includeImports, *includeValues, *outputFormat, *includeResources, *excludeResources, *listResources)
}

// runExport handles API export of all resources from an environment
// All exports now generate Terraform module structure
func (c *ExportCommand) runExport(logger grpc.Logger, workerEnvironmentID, exportEnvironmentID, regionCode, clientID, clientSecret, out string, skipDeps bool, generateImports bool, moduleDir string, moduleName string, includeImports bool, includeValues bool, outputFormat string, includeResources []string, excludeResources []string, listResources bool) error {
	// Get credentials from environment variables if not provided via flags
	if workerEnvironmentID == "" {
		workerEnvironmentID = os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID")
	}
	if exportEnvironmentID == "" {
		exportEnvironmentID = os.Getenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID")
		// Default export environment to worker environment if not specified
		if exportEnvironmentID == "" {
			exportEnvironmentID = workerEnvironmentID
		}
	}
	if regionCode == "" {
		regionCode = os.Getenv("PINGCLI_PINGONE_REGION_CODE")
	}
	if clientID == "" {
		clientID = os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
	}

	// Validate required credentials
	if workerEnvironmentID == "" {
		return fmt.Errorf("worker environment ID is required: use --pingone-worker-environment-id flag or PINGCLI_PINGONE_ENVIRONMENT_ID env var")
	}
	if clientID == "" {
		return fmt.Errorf("client ID is required: use --pingone-worker-client-id flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID env var")
	}
	if clientSecret == "" {
		return fmt.Errorf("client secret is required: use --pingone-worker-client-secret flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET env var")
	}

	// Default region to NA if not specified
	if regionCode == "" {
		regionCode = "NA"
	}

	// Log export start
	if err := logger.Message(fmt.Sprintf("Exporting PingOne resources from environment: %s (Region: %s)", exportEnvironmentID, regionCode), nil); err != nil {
		return err
	}

	// Create API client
	// Use NewClient to support two-environment model: worker environment for auth, export environment for resources
	ctx := context.Background()
	client, err := api.NewClient(ctx, workerEnvironmentID, exportEnvironmentID, regionCode, clientID, clientSecret)
	if err != nil {
		if logErr := logger.PluginError("Failed to create API client", map[string]string{
			"worker_environment_id": workerEnvironmentID,
			"export_environment_id": exportEnvironmentID,
			"region_code":           regionCode,
			"error":                 err.Error(),
		}); logErr != nil {
			return fmt.Errorf("failed to log error: %w", logErr)
		}
		return fmt.Errorf("failed to create API client: %w", err)
	}

	return c.exportAsModule(ctx, client, logger, skipDeps, includeImports, includeValues, moduleDir, moduleName, out, exportEnvironmentID, outputFormat, includeResources, excludeResources, listResources)
}

// exportAsModule uses the schema-driven orchestrator pipeline to export
// resources and generate a Terraform module.
func (c *ExportCommand) exportAsModule(ctx context.Context, client *api.Client, logger grpc.Logger, skipDeps, includeImports, includeValues bool, moduleDir, moduleName, out, environmentID, outputFormat string, includeResources []string, excludeResources []string, listResources bool) error {
	outputDir := out
	if outputDir == "" {
		outputDir = "."
	}

	if err := logger.Message(fmt.Sprintf("Generating Terraform module in: %s/%s", outputDir, moduleDir), nil); err != nil {
		return fmt.Errorf("failed to log message: %w", err)
	}

	// 1. Load schema definitions from embedded FS.
	registry := schema.NewRegistry()
	if err := registry.LoadFromFS(definitions.FS, "pingone"); err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// 2. Create custom handler registry and load platform-specific handlers.
	customReg := core.NewCustomHandlerRegistry()
	pingoneplatform.RegisterCustomHandlers(customReg)

	// 3. Create processor.
	proc := core.NewProcessor(registry, core.WithCustomHandlers(customReg))

	// 4. Create API client adapter.
	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return fmt.Errorf("invalid environment ID format: %w", err)
	}
	apiClient := pingoneplatform.New(client.APIClient(), envUUID)

	// Build resource filter from include/exclude patterns.
	var resourceFilter *filter.ResourceFilter
	if len(includeResources) > 0 || len(excludeResources) > 0 {
		var err error
		resourceFilter, err = filter.NewResourceFilter(includeResources, excludeResources)
		if err != nil {
			return fmt.Errorf("invalid resource filter pattern: %w", err)
		}
	}

	// 5. Create and run orchestrator.
	orch := core.NewExportOrchestrator(registry, proc, apiClient, core.WithProgressFunc(func(msg string) {
		_ = logger.Message(msg, nil)
	}))

	result, err := orch.Export(ctx, core.ExportOptions{
		SkipDependencies: skipDeps,
		GenerateImports:  includeImports,
		EnvironmentID:    environmentID,
		ResourceFilter:   resourceFilter,
		ListOnly:         listResources,
	})
	if err != nil {
		return fmt.Errorf("orchestrator export failed: %w", err)
	}

	// Handle --list-resources mode: print resource addresses and exit
	if listResources {
		for _, erd := range result.ResourcesByType {
			for _, rd := range erd.Resources {
				if err := logger.Message(fmt.Sprintf("%s.%s", erd.ResourceType, rd.Label), nil); err != nil {
					return fmt.Errorf("failed to log resource: %w", err)
				}
			}
		}
		return nil  // Exit early, don't generate module
	}

	// 6. Format results into output, imports, and variables.
	outFmt, fmtErr := formatters.NewFormatter(outputFormat)
	if fmtErr != nil {
		return fmtErr
	}
	varExtractor := variables.NewVariableExtractor(registry)

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
			SkipDependencies: skipDeps,
			EnvironmentID:    environmentID,
		})
		if fmtErr != nil {
			return fmt.Errorf("format %s: %w", erd.ResourceType, fmtErr)
		}
		allOutputParts = append(allOutputParts, formattedOutput)

		// Generate import blocks.
		if includeImports && erd.Definition.Dependencies.ImportIDFormat != "" {
			for _, rd := range erd.Resources {
				label := importResourceLabel(rd, erd.Definition)
				allImportBlocks = append(allImportBlocks, module.ImportBlock{
					To: fmt.Sprintf("module.%s.%s.%s", moduleName, erd.ResourceType, label),
					ID: buildImportID(erd.Definition, rd, environmentID),
				})
			}
		}
	}

	// 7. Build module structure.
	moduleConfig := module.ModuleConfig{
		OutputDir:      outputDir,
		ModuleDirName:  moduleDir,
		ModuleName:     moduleName,
		IncludeImports: includeImports,
		IncludeValues:  includeValues,
		EnvironmentID:  environmentID,
		OutputFormat:   outputFormat,
	}

	// Combine formatted output by resource type into ModuleResources.
	resources := buildModuleResources(result, allOutputParts)

	structure := &module.ModuleStructure{
		Config:       moduleConfig,
		Variables:    allVariables,
		Resources:    resources,
		ImportBlocks: allImportBlocks,
	}

	// 8. Generate module files.
	generator := module.NewGenerator(moduleConfig)
	if err := generator.Generate(structure); err != nil {
		return fmt.Errorf("failed to generate module: %w", err)
	}

	if err := logger.Message(fmt.Sprintf("✓ Module successfully generated in: %s", outputDir), map[string]string{
		"module_dir":      moduleDir,
		"include_imports": fmt.Sprintf("%v", includeImports),
		"include_values":  fmt.Sprintf("%v", includeValues),
	}); err != nil {
		return fmt.Errorf("failed to log success: %w", err)
	}

	return nil
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

// importResourceLabel derives the sanitized resource label for import blocks,
// matching the logic used by the imports generator and HCL/tfjson formatters.
func importResourceLabel(data *core.ResourceData, def *schema.ResourceDefinition) string {
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

// buildImportID expands the definition's import ID format for a single resource.
func buildImportID(def *schema.ResourceDefinition, rd *core.ResourceData, environmentID string) string {
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
