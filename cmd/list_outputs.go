// Copyright © 2025 Ping Identity Corporation

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/pingidentity/pingcli-plugin-terraformer/definitions"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/filter"
	pingoneplatform "github.com/pingidentity/pingcli-plugin-terraformer/internal/platform/pingone"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli/shared/grpc"
	"github.com/spf13/pflag"
)

var (
	ListOutputsExample = `  # List all attribute paths across all exported resources
  pingcli tf list-outputs \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --pingone-region-code NA

  # List attribute paths for DaVinci flows only (piped to a file for later use)
  pingcli tf list-outputs \
    --include-resources "pingone_davinci_flow.*" \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --pingone-region-code NA > flow-outputs.txt

  # Use the output file with export
  pingcli tf export \
    --output-attribute-file flow-outputs.txt \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --out ./output

  # List two levels of nesting
  pingcli tf list-outputs --depth 2 \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret>`

	ListOutputsLong = `List all possible output attribute paths for exported resources.

Connects to PingOne and fetches resource labels (same as --list-resources), then
enumerates schema-defined attribute paths up to the requested depth.

Each line of output is a path in resource_type.label.attr format that can be
passed directly to --output-attribute or written to a file for --output-attribute-file.

Computed attributes (e.g. id, current_version) are always included — these are
often the most useful for Terraform module outputs.

Use --depth 2 to include one level of nested object attributes (e.g. settings.csp).`

	ListOutputsShort = "List all possible output attribute paths for exported resources"

	ListOutputsUse = "list-outputs [flags]"
)

// ListOutputsCommand implements the list-outputs subcommand.
type ListOutputsCommand struct{}

var _ grpc.PingCliCommand = (*ListOutputsCommand)(nil)

func (c *ListOutputsCommand) Configuration() (*grpc.PingCliCommandConfiguration, error) {
	return &grpc.PingCliCommandConfiguration{
		Use:     ListOutputsUse,
		Short:   ListOutputsShort,
		Long:    ListOutputsLong,
		Example: ListOutputsExample,
	}, nil
}

func (c *ListOutputsCommand) Run(args []string, logger grpc.Logger) error {
	flags := pflag.NewFlagSet("list-outputs", pflag.ContinueOnError)

	workerEnvironmentID := flags.String("pingone-worker-environment-id", "", "PingOne environment ID containing the worker app")
	exportEnvironmentID := flags.String("pingone-export-environment-id", "", "PingOne environment ID to export resources from (defaults to worker environment)")
	regionCode := flags.String("pingone-region-code", "", "PingOne region code (NA, EU, AP, CA, AU, SG)")
	clientID := flags.String("pingone-worker-client-id", "", "OAuth worker app client ID")
	clientSecret := flags.String("pingone-worker-client-secret", "", "OAuth worker app client secret")
	depth := flags.Int("depth", 1, "Attribute enumeration depth (1 = top-level only; 2 = one level of nesting)")
	includeResources := flags.StringSlice("include-resources", []string{}, "Include resources matching glob pattern(s)")
	excludeResources := flags.StringSlice("exclude-resources", []string{}, "Exclude resources matching glob pattern(s)")
	includeUpstream := flags.Bool("include-upstream", false, "Include upstream dependencies of filtered resources")

	if err := flags.Parse(args); err != nil {
		return err
	}

	return c.runListOutputs(logger, *workerEnvironmentID, *exportEnvironmentID, *regionCode, *clientID, *clientSecret, *depth, *includeResources, *excludeResources, *includeUpstream)
}

func (c *ListOutputsCommand) runListOutputs(logger grpc.Logger, workerEnvironmentID, exportEnvironmentID, regionCode, clientID, clientSecret string, depth int, includeResources, excludeResources []string, includeUpstream bool) error {
	if workerEnvironmentID == "" {
		workerEnvironmentID = os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID")
	}
	if exportEnvironmentID == "" {
		exportEnvironmentID = os.Getenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID")
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

	if workerEnvironmentID == "" {
		return fmt.Errorf("worker environment ID is required: use --pingone-worker-environment-id flag or PINGCLI_PINGONE_ENVIRONMENT_ID env var")
	}
	if clientID == "" {
		return fmt.Errorf("client ID is required: use --pingone-worker-client-id flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID env var")
	}
	if clientSecret == "" {
		return fmt.Errorf("client secret is required: use --pingone-worker-client-secret flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET env var")
	}

	if regionCode == "" {
		regionCode = "NA"
	}

	ctx := context.Background()
	client, err := pingoneplatform.NewFromCredentials(ctx, workerEnvironmentID, exportEnvironmentID, regionCode, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	reg := schema.NewRegistry()
	if err := reg.LoadFromFS(definitions.FS, "pingone"); err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	customReg := core.NewCustomHandlerRegistry()
	pingoneplatform.RegisterCustomHandlers(customReg)
	proc := core.NewProcessor(reg, core.WithCustomHandlers(customReg))

	var resourceFilter *filter.ResourceFilter
	if len(includeResources) > 0 || len(excludeResources) > 0 {
		var err error
		resourceFilter, err = filter.NewResourceFilter(includeResources, excludeResources)
		if err != nil {
			return fmt.Errorf("invalid resource filter pattern: %w", err)
		}
	}

	embeddedRefs := pingoneplatform.NewEmbeddedReferenceRegistry()
	orch := core.NewExportOrchestrator(reg, proc, client, core.WithEmbeddedReferences(embeddedRefs))

	result, err := orch.Export(ctx, core.ExportOptions{
		EnvironmentID:   exportEnvironmentID,
		ListOnly:        true,
		ResourceFilter:  resourceFilter,
		IncludeUpstream: includeUpstream,
	})
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	var lines []string
	for _, erd := range result.ResourcesByType {
		for _, rd := range erd.Resources {
			schema.WalkAttributes(erd.Definition.Attributes, depth, "", func(attrPath string, _ schema.AttributeDefinition) {
				lines = append(lines, fmt.Sprintf("%s.%s.%s", erd.ResourceType, rd.Label, attrPath))
			})
		}
	}
	sort.Strings(lines)

	// Write paths to stdout so the output is pipeable (e.g. grep | file).
	// Progress/error messages from the logger go to stderr, keeping the two streams separate.
	w := bufio.NewWriter(os.Stdout)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}
