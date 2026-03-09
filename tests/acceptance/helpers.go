//go:build acceptance

package acceptance

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/api"
	"github.com/stretchr/testify/require"
)

// createTestClient creates an API client using environment variables
// Supports separate auth and target environments
func createTestClient(t *testing.T) *api.Client {
	clientID := requireEnv(t, "PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID")
	clientSecret := requireEnv(t, "PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
	authEnvID := requireEnv(t, "PINGCLI_PINGONE_ENVIRONMENT_ID")
	targetEnvID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", authEnvID) // Default to auth env
	region := getEnvOrDefault("PINGONE_REGION", "NA")

	client, err := api.NewClient(context.Background(), authEnvID, targetEnvID, region, clientID, clientSecret)
	require.NoError(t, err, "Failed to create API client")
	return client
}

// requireEnv gets an environment variable or fails the test if not set
func requireEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping acceptance test: %s not set", key)
	}
	return value
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// skipIfNoCredentials skips the test if required credentials are not set
func skipIfNoCredentials(t *testing.T) {
	if os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID") == "" {
		t.Skip("Skipping acceptance test: PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID not set")
	}
}

// countResourcesInHCL counts resources of each type in generated HCL
func countResourcesInHCL(hcl string) map[string]int {
	counts := make(map[string]int)

	// Match resource blocks: resource "type" "name" {
	re := regexp.MustCompile(`resource\s+"([^"]+)"\s+"[^"]+"\s+{`)
	matches := re.FindAllStringSubmatch(hcl, -1)

	for _, match := range matches {
		if len(match) > 1 {
			resourceType := match[1]
			counts[resourceType]++
		}
	}

	return counts
}

// verifyResourceOrdering checks that resources appear in the correct dependency order
func verifyResourceOrdering(t *testing.T, hcl string) {
	// Find all resource positions
	positions := make(map[string]int)

	resourceTypes := []string{
		"pingone_davinci_variable",
		"pingone_davinci_connection",
		"pingone_davinci_flow",
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
	}

	for _, resourceType := range resourceTypes {
		index := strings.Index(hcl, `resource "`+resourceType+`"`)
		if index != -1 {
			positions[resourceType] = index
		}
	}

	// Verify ordering: variables < connections < flows < applications < flow policies
	if varPos, varOk := positions["pingone_davinci_variable"]; varOk {
		if connPos, connOk := positions["pingone_davinci_connection"]; connOk {
			require.Less(t, varPos, connPos, "Variables should appear before connections")
		}
	}

	if connPos, connOk := positions["pingone_davinci_connection"]; connOk {
		if flowPos, flowOk := positions["pingone_davinci_flow"]; flowOk {
			require.Less(t, connPos, flowPos, "Connections should appear before flows")
		}
	}

	if flowPos, flowOk := positions["pingone_davinci_flow"]; flowOk {
		if appPos, appOk := positions["pingone_davinci_application"]; appOk {
			require.Less(t, flowPos, appPos, "Flows should appear before applications")
		}
	}

	if appPos, appOk := positions["pingone_davinci_application"]; appOk {
		if policyPos, policyOk := positions["pingone_davinci_application_flow_policy"]; policyOk {
			require.Less(t, appPos, policyPos, "Applications should appear before flow policies")
		}
	}
}

// validateHCLSyntax performs basic HCL syntax validation
// validateFlowResourceHCL validates the structure of flow resource HCL
func validateFlowResourceHCL(t *testing.T, hcl string) {
	// Check for balanced braces
	openBraces := strings.Count(hcl, "{")
	closeBraces := strings.Count(hcl, "}")
	require.Equal(t, openBraces, closeBraces, "HCL has unbalanced braces")

	// Check for basic resource structure
	require.Contains(t, hcl, "resource \"", "HCL should contain resource blocks")
	require.Contains(t, hcl, "environment_id", "Flow resources should have environment_id")
}

// validateHCLSyntax validates complete HCL file structure including terraform blocks
func validateHCLSyntax(t *testing.T, hcl string) {
	// Check for balanced braces
	openBraces := strings.Count(hcl, "{")
	closeBraces := strings.Count(hcl, "}")
	require.Equal(t, openBraces, closeBraces, "HCL has unbalanced braces")

	// Check for basic resource structure
	require.Contains(t, hcl, "resource \"", "HCL should contain resource blocks")

	// Check for terraform required block (for complete Terraform configurations)
	require.Contains(t, hcl, "terraform {", "HCL should contain terraform configuration")
	require.Contains(t, hcl, "required_providers", "HCL should specify required providers")
}
