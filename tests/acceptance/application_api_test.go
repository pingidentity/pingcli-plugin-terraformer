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

func TestApplicationAPI(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	exportEnvID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID"))

	t.Run("ListApplications", func(t *testing.T) {
		applications, err := client.ListApplications(ctx, exportEnvID)
		require.NoError(t, err)
		assert.NotNil(t, applications)

		t.Logf("Found %d applications in environment %s", len(applications), exportEnvID)
		for _, app := range applications {
			assert.NotEmpty(t, app.GetId())
			assert.NotEmpty(t, app.GetName())
			t.Logf("  Application: ID=%s, Name=%s",
				app.GetId(),
				app.GetName())
		}
	})
}

func TestGetApplicationById(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	exportEnvID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID"))

	// First get list of applications
	applications, err := client.ListApplications(ctx, exportEnvID)
	require.NoError(t, err)
	require.NotEmpty(t, applications, "No applications found for GetApplication test")

	testApp := applications[0]
	appID := testApp.GetId()

	t.Run("GetExistingApplication", func(t *testing.T) {
		application, err := client.GetApplication(ctx, exportEnvID, appID)
		require.NoError(t, err)
		assert.NotNil(t, application)
		assert.Equal(t, appID, application.GetId())
		assert.Equal(t, testApp.GetName(), application.GetName())

		t.Logf("Retrieved application: ID=%s, Name=%s",
			application.GetId(),
			application.GetName())

		// Check for API key or OAuth
		if apiKey, ok := application.GetApiKeyOk(); ok && apiKey != nil {
			t.Logf("  Application has API key configured")
		}
		if oauth, ok := application.GetOauthOk(); ok && oauth != nil {
			t.Logf("  Application has OAuth configured")
		}
	})

	t.Run("GetNonexistentApplication", func(t *testing.T) {
		_, err := client.GetApplication(ctx, exportEnvID, "nonexistent-app-id")
		require.Error(t, err)
		t.Logf("Expected error for nonexistent application: %v", err)
	})
}

func TestListApplicationsEmpty(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	workerEnvID := os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID")

	// Use worker environment which should have no applications
	applications, err := client.ListApplications(ctx, workerEnvID)
	require.NoError(t, err)

	t.Logf("Found %d applications in worker environment %s (expected 0)", len(applications), workerEnvID)
}
