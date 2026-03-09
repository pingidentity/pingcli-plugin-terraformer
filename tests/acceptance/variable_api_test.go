//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableAPI(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	exportEnvID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID"))

	t.Run("ListVariables", func(t *testing.T) {
		variables, err := client.ListVariables(ctx, exportEnvID)
		require.NoError(t, err)
		assert.NotNil(t, variables)

		t.Logf("Found %d variables in environment %s", len(variables), exportEnvID)
		for _, variable := range variables {
			assert.NotEmpty(t, variable.GetId())
			assert.NotEmpty(t, variable.GetName())
			t.Logf("  Variable: ID=%s, Name=%s, DataType=%v, Context=%s",
				variable.GetId(),
				variable.GetName(),
				variable.GetDataType(),
				variable.GetContext())
		}
	})
}

func TestGetVariableById(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	exportEnvID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID"))

	// First get list of variables
	variables, err := client.ListVariables(ctx, exportEnvID)
	require.NoError(t, err)
	require.NotEmpty(t, variables, "No variables found for GetVariable test")

	testVariable := variables[0]
	variableID := testVariable.GetId().String()

	t.Run("GetExistingVariable", func(t *testing.T) {
		variable, err := client.GetVariable(ctx, exportEnvID, variableID)
		require.NoError(t, err)
		assert.NotNil(t, variable)
		assert.Equal(t, variableID, variable.GetId().String())
		assert.Equal(t, testVariable.GetName(), variable.GetName())

		t.Logf("Retrieved variable: ID=%s, Name=%s, DataType=%v, Context=%s",
			variable.GetId(),
			variable.GetName(),
			variable.GetDataType(),
			variable.GetContext())
	})

	t.Run("GetNonexistentVariable", func(t *testing.T) {
		_, err := client.GetVariable(ctx, exportEnvID, "00000000-0000-0000-0000-000000000001")
		require.Error(t, err)
		t.Logf("Expected error for nonexistent variable: %v", err)
	})
}

func TestListVariablesEmpty(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	workerEnvID := os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID")

	// Use worker environment which should have no variables
	variables, err := client.ListVariables(ctx, workerEnvID)
	require.NoError(t, err)

	t.Logf("Found %d variables in worker environment %s (expected 0)", len(variables), workerEnvID)
}
