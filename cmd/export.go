// Copyright © 2025 Ping Identity Corporation

// Package cmd provides the command implementation for the Terraform converter.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/export"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters"
	pingoneplatform "github.com/pingidentity/pingcli-plugin-terraformer/internal/platform/pingone"
	"github.com/pingidentity/pingcli/shared/grpc"
	"github.com/spf13/pflag"
)

var (
	// ExportExample provides usage examples for the export command
	ExportExample = `  # Export all resources
  pingcli tf export

  # Export to a specific directory in HCL format
  pingcli tf export --out ./output

  # Export in Terraform JSON format
  pingcli tf export --output-format tfjson --out ./output

  # Export with import blocks and values
  pingcli tf export --include-imports --include-values --out ./output

  # Export only specific resources
  pingcli tf export \
    --include-resources "pingone_davinci_flow*" \
    --out ./output

  # Export everything except test resources
  pingcli tf export \
    --exclude-resources "*Test*" \
    --out ./output

  # Export a specific flow and its upstream dependencies (e.g., connector instances)
  pingcli tf export \
    --include-resources "pingone_davinci_flow.pingcli__Login*" \
    --include-upstream \
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
  PINGCLI_PINGONE_REGION_CODE                       - Region code (AP, AU, CA, EU, NA, SG)
  PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID  - Target environment to export (optional, defaults to worker environment)

Resource Filtering:
  Use --include-resources and --exclude-resources to filter resources by pattern.
  Patterns match against: resource_type.terraform_label (e.g., pingone_davinci_flow.pingcli__Login)
  Glob wildcards (* and ?) are supported. Use 'regex:' prefix for regex patterns.
  Matching is case-insensitive. Multiple patterns combine via OR (union).
  Use --include-upstream with --include-resources to automatically include upstream dependencies
  (resources referenced by matched resources). This is transitive: dependencies of dependencies are
  also included. Explicit --exclude-resources patterns always take priority.
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
// It parses flags and delegates the pipeline to internal/export.
func (c *ExportCommand) Run(args []string, logger grpc.Logger) error {
	// Create a new FlagSet for parsing command-line flags
	flags := pflag.NewFlagSet("export", pflag.ContinueOnError)

	// Define API export flags matching Ping CLI standards
	workerEnvironmentID := flags.String("pingone-worker-environment-id", "", "PingOne environment ID containing the worker app")
	exportEnvironmentID := flags.String("pingone-export-environment-id", "", "PingOne environment ID to export resources from (defaults to worker environment)")
	regionCode := flags.String("pingone-region-code", "", "PingOne region code (NA, EU, AP, CA, AU, SG)")
	clientID := flags.String("pingone-worker-client-id", "", "OAuth worker app client ID")
	clientSecret := flags.String("pingone-worker-client-secret", "", "OAuth worker app client secret")
	out := flags.StringP("out", "o", "", "Output file path (default: stdout)")
	skipDependencies := flags.Bool("skip-dependencies", false, "Skip dependency resolution")

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
	includeUpstream := flags.Bool("include-upstream", false, "Automatically include upstream dependencies of filtered resources")

	// Output attribute flags
	outputAttributes := flags.StringSlice("output-attribute", []string{}, "Add an output block for resource_type.label.attr (repeatable; glob * supported in label position)")
	outputAttributeFile := flags.String("output-attribute-file", "", "File with one resource_type.label.attr path per line (same format as --output-attribute)")

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

	// Get credentials from environment variables if not provided via flags
	workerEnv := *workerEnvironmentID
	if workerEnv == "" {
		workerEnv = os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID")
	}
	exportEnv := *exportEnvironmentID
	if exportEnv == "" {
		exportEnv = os.Getenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID")
		if exportEnv == "" {
			exportEnv = workerEnv
		}
	}
	region := *regionCode
	if region == "" {
		region = os.Getenv("PINGCLI_PINGONE_REGION_CODE")
	}
	cid := *clientID
	if cid == "" {
		cid = os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID")
	}
	secret := *clientSecret
	if secret == "" {
		secret = os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
	}

	// Validate required credentials
	if workerEnv == "" {
		return fmt.Errorf("worker environment ID is required: use --pingone-worker-environment-id flag or PINGCLI_PINGONE_ENVIRONMENT_ID env var")
	}
	if cid == "" {
		return fmt.Errorf("client ID is required: use --pingone-worker-client-id flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID env var")
	}
	if secret == "" {
		return fmt.Errorf("client secret is required: use --pingone-worker-client-secret flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET env var")
	}

	// Default region to NA if not specified
	if region == "" {
		region = "NA"
	}

	// Log export start
	if err := logger.Message(fmt.Sprintf("Exporting PingOne resources from environment: %s (Region: %s)", exportEnv, region), nil); err != nil {
		return err
	}

	// Create API client
	// Use NewFromCredentials to support two-environment model: worker environment for auth, export environment for resources
	ctx := context.Background()
	client, err := pingoneplatform.NewFromCredentials(ctx, workerEnv, exportEnv, region, cid, secret)
	if err != nil {
		if logErr := logger.PluginError("Failed to create API client", map[string]string{
			"worker_environment_id": workerEnv,
			"export_environment_id": exportEnv,
			"region_code":           region,
			"error":                 err.Error(),
		}); logErr != nil {
			return fmt.Errorf("failed to log error: %w", logErr)
		}
		return fmt.Errorf("failed to create API client: %w", err)
	}

	registry, err := export.LoadFullRegistry()
	if err != nil {
		return err
	}

	return export.Run(ctx, export.RunOptions{
		Registry:            registry,
		Client:              client,
		Logger:              logger,
		EnvironmentID:       exportEnv,
		OutputDir:           *out,
		ModuleDir:           *moduleDir,
		ModuleName:          *moduleName,
		IncludeImports:      *includeImports,
		IncludeValues:       *includeValues,
		OutputFormat:        *outputFormat,
		SkipDependencies:    *skipDependencies,
		IncludeResources:    *includeResources,
		ExcludeResources:    *excludeResources,
		ListOnly:            *listResources,
		IncludeUpstream:     *includeUpstream,
		OutputAttributes:    *outputAttributes,
		OutputAttributeFile: *outputAttributeFile,
	})
}
