package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListVariables(t *testing.T) {
	client := &Client{
		EnvironmentID: "00000000-0000-0000-0000-000000000000",
		Region:        "NA",
	}
	ctx := context.Background()

	t.Run("EmptyEnvironmentID", func(t *testing.T) {
		_, err := client.ListVariables(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment ID is required")
	})

	t.Run("InvalidEnvironmentIDFormat", func(t *testing.T) {
		_, err := client.ListVariables(ctx, "invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment ID format")
	})
}

func TestGetVariable(t *testing.T) {
	client := &Client{
		EnvironmentID: "00000000-0000-0000-0000-000000000000",
		Region:        "NA",
	}
	ctx := context.Background()
	testVariableID := "00000000-0000-0000-0000-000000000001"
	testEnvID := "00000000-0000-0000-0000-000000000000"

	t.Run("EmptyEnvironmentID", func(t *testing.T) {
		_, err := client.GetVariable(ctx, "", testVariableID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment ID is required")
	})

	t.Run("EmptyVariableID", func(t *testing.T) {
		_, err := client.GetVariable(ctx, testEnvID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "variable ID is required")
	})

	t.Run("InvalidEnvironmentIDFormat", func(t *testing.T) {
		_, err := client.GetVariable(ctx, "invalid-uuid", testVariableID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment ID format")
	})

	t.Run("InvalidVariableIDFormat", func(t *testing.T) {
		_, err := client.GetVariable(ctx, testEnvID, "invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid variable ID format")
	})
}
