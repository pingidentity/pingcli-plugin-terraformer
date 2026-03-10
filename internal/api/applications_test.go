package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListApplications(t *testing.T) {
	client := &Client{
		EnvironmentID: "00000000-0000-0000-0000-000000000000",
		Region:        "NA",
	}
	ctx := context.Background()

	t.Run("EmptyEnvironmentID", func(t *testing.T) {
		_, err := client.ListApplications(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment ID is required")
	})

	t.Run("InvalidEnvironmentIDFormat", func(t *testing.T) {
		_, err := client.ListApplications(ctx, "invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment ID format")
	})
}

func TestGetApplication(t *testing.T) {
	client := &Client{
		EnvironmentID: "00000000-0000-0000-0000-000000000000",
		Region:        "NA",
	}
	ctx := context.Background()
	testAppID := "test-app-id"
	testEnvID := "00000000-0000-0000-0000-000000000000"

	t.Run("EmptyEnvironmentID", func(t *testing.T) {
		_, err := client.GetApplication(ctx, "", testAppID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment ID is required")
	})

	t.Run("EmptyApplicationID", func(t *testing.T) {
		_, err := client.GetApplication(ctx, testEnvID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "application ID is required")
	})

	t.Run("InvalidEnvironmentIDFormat", func(t *testing.T) {
		_, err := client.GetApplication(ctx, "invalid-uuid", testAppID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment ID format")
	})
}
