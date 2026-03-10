package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                string
		authEnvironmentID   string
		targetEnvironmentID string
		region              string
		clientID            string
		clientSecret        string
		expectError         bool
		errorContains       string
	}{
		{
			name:                "Valid client configuration",
			authEnvironmentID:   "auth-env-123",
			targetEnvironmentID: "target-env-456",
			region:              "NA",
			clientID:            "client-123",
			clientSecret:        "secret-123",
			expectError:         false,
		},
		{
			name:                "Same auth and target environment",
			authEnvironmentID:   "env-123",
			targetEnvironmentID: "env-123",
			region:              "NA",
			clientID:            "client-123",
			clientSecret:        "secret-123",
			expectError:         false,
		},
		{
			name:                "Missing auth environment ID",
			authEnvironmentID:   "",
			targetEnvironmentID: "target-env-456",
			region:              "NA",
			clientID:            "client-123",
			clientSecret:        "secret-123",
			expectError:         true,
			errorContains:       "auth environment ID is required",
		},
		{
			name:                "Missing target environment ID",
			authEnvironmentID:   "auth-env-123",
			targetEnvironmentID: "",
			region:              "NA",
			clientID:            "client-123",
			clientSecret:        "secret-123",
			expectError:         true,
			errorContains:       "target environment ID is required",
		},
		{
			name:                "Missing region",
			authEnvironmentID:   "auth-env-123",
			targetEnvironmentID: "target-env-456",
			region:              "",
			clientID:            "client-123",
			clientSecret:        "secret-123",
			expectError:         true,
			errorContains:       "region is required",
		},
		{
			name:                "Missing client ID",
			authEnvironmentID:   "auth-env-123",
			targetEnvironmentID: "target-env-456",
			region:              "NA",
			clientID:            "",
			clientSecret:        "secret-123",
			expectError:         true,
			errorContains:       "client ID is required",
		},
		{
			name:                "Missing client secret",
			authEnvironmentID:   "auth-env-123",
			targetEnvironmentID: "target-env-456",
			region:              "NA",
			clientID:            "client-123",
			clientSecret:        "",
			expectError:         true,
			errorContains:       "client secret is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ctx, tt.authEnvironmentID, tt.targetEnvironmentID, tt.region, tt.clientID, tt.clientSecret)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				assert.Equal(t, tt.authEnvironmentID, client.AuthEnvironmentID)
				assert.Equal(t, tt.targetEnvironmentID, client.EnvironmentID)
				assert.Equal(t, tt.region, client.Region)
			}
		})
	}
}

func TestIsValidRegion(t *testing.T) {
	tests := []struct {
		region string
		valid  bool
	}{
		{"NA", true},
		{"EU", true},
		{"AP", true},
		{"CA", true},
		{"US", false},
		{"", false},
		{"INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidRegion(tt.region))
		})
	}
}

func TestValidRegions(t *testing.T) {
	regions := ValidRegions()
	assert.Len(t, regions, 4)
	assert.Contains(t, regions, "NA")
	assert.Contains(t, regions, "EU")
	assert.Contains(t, regions, "AP")
	assert.Contains(t, regions, "CA")
}

func TestNewClientSingleEnvironment(t *testing.T) {
	ctx := context.Background()

	envID := "env-123"
	region := "NA"
	clientID := "client-123"
	clientSecret := "secret-123"

	client, err := NewClientSingleEnvironment(ctx, envID, region, clientID, clientSecret)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, envID, client.AuthEnvironmentID, "Auth environment should match provided ID")
	assert.Equal(t, envID, client.EnvironmentID, "Target environment should match provided ID")
	assert.Equal(t, region, client.Region)
}
