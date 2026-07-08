package pingone

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientImplementsInterface(t *testing.T) {
	var _ clients.APIClient = (*Client)(nil)
}

func TestPlatform(t *testing.T) {
	c := &Client{}
	assert.Equal(t, "pingone", c.Platform())
}

func TestNewClient(t *testing.T) {
	envID := uuid.New()
	c := New(nil, envID)
	assert.NotNil(t, c)
	assert.Nil(t, c.apiClient)
	assert.Equal(t, envID, c.environmentID)
}

func TestNewFromCredentials(t *testing.T) {
	ctx := context.Background()
	// A valid UUID to use as exportEnvID in success cases.
	validExportEnvID := "00000000-0000-0000-0000-000000000001"

	tests := []struct {
		name          string
		workerEnvID   string
		exportEnvID   string
		region        string
		clientID      string
		clientSecret  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing auth environment ID",
			workerEnvID:   "",
			exportEnvID:   validExportEnvID,
			region:        "NA",
			clientID:      "client-123",
			clientSecret:  "secret-123",
			expectError:   true,
			errorContains: "auth environment ID is required",
		},
		{
			name:          "missing target environment ID",
			workerEnvID:   "auth-env-123",
			exportEnvID:   "",
			region:        "NA",
			clientID:      "client-123",
			clientSecret:  "secret-123",
			expectError:   true,
			errorContains: "target environment ID is required",
		},
		{
			name:          "missing region",
			workerEnvID:   "auth-env-123",
			exportEnvID:   validExportEnvID,
			region:        "",
			clientID:      "client-123",
			clientSecret:  "secret-123",
			expectError:   true,
			errorContains: "region is required",
		},
		{
			name:          "invalid region",
			workerEnvID:   "auth-env-123",
			exportEnvID:   validExportEnvID,
			region:        "XX",
			clientID:      "client-123",
			clientSecret:  "secret-123",
			expectError:   true,
			errorContains: "invalid region: XX",
		},
		{
			name:          "missing client ID",
			workerEnvID:   "auth-env-123",
			exportEnvID:   validExportEnvID,
			region:        "NA",
			clientID:      "",
			clientSecret:  "secret-123",
			expectError:   true,
			errorContains: "client ID is required",
		},
		{
			name:          "missing client secret",
			workerEnvID:   "auth-env-123",
			exportEnvID:   validExportEnvID,
			region:        "NA",
			clientID:      "client-123",
			clientSecret:  "",
			expectError:   true,
			errorContains: "client secret is required",
		},
		{
			name:          "invalid export environment ID format",
			workerEnvID:   "auth-env-123",
			exportEnvID:   "not-a-uuid",
			region:        "NA",
			clientID:      "client-123",
			clientSecret:  "secret-123",
			expectError:   true,
			errorContains: "invalid export environment ID format",
		},
		{
			name:         "valid credentials",
			workerEnvID:  "auth-env-123",
			exportEnvID:  validExportEnvID,
			region:       "NA",
			clientID:     "client-123",
			clientSecret: "secret-123",
			expectError:  false,
		},
		{
			name:         "valid credentials AU region",
			workerEnvID:  "auth-env-123",
			exportEnvID:  "00000000-0000-0000-0000-000000000002",
			region:       "AU",
			clientID:     "client-123",
			clientSecret: "secret-123",
			expectError:  false,
		},
		{
			name:         "valid credentials SG region",
			workerEnvID:  "auth-env-123",
			exportEnvID:  "00000000-0000-0000-0000-000000000003",
			region:       "SG",
			clientID:     "client-123",
			clientSecret: "secret-123",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewFromCredentials(ctx, tt.workerEnvID, tt.exportEnvID, tt.region, tt.clientID, tt.clientSecret, "dev")

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				// exportEnvID must be stored as environmentID (not workerEnvID)
				expectedUUID, _ := uuid.Parse(tt.exportEnvID)
				assert.Equal(t, expectedUUID, client.environmentID)
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
		{"AU", true},
		{"SG", true},
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
	require.Len(t, regions, 6)
	assert.Contains(t, regions, "NA")
	assert.Contains(t, regions, "EU")
	assert.Contains(t, regions, "AP")
	assert.Contains(t, regions, "CA")
	assert.Contains(t, regions, "AU")
	assert.Contains(t, regions, "SG")
}
