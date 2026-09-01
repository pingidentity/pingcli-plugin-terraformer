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
	// Valid UUIDs — real PingOne client/environment IDs are UUIDs, and the
	// management SDK's Config.Validate() rejects non-UUID-shaped values.
	validExportEnvID := "00000000-0000-0000-0000-000000000001"
	validWorkerEnvID := "00000000-0000-0000-0000-000000000010"
	validClientID := "00000000-0000-0000-0000-000000000020"
	validClientSecret := "secret-123"

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
			clientID:      validClientID,
			clientSecret:  validClientSecret,
			expectError:   true,
			errorContains: "auth environment ID is required",
		},
		{
			name:          "missing target environment ID",
			workerEnvID:   validWorkerEnvID,
			exportEnvID:   "",
			region:        "NA",
			clientID:      validClientID,
			clientSecret:  validClientSecret,
			expectError:   true,
			errorContains: "target environment ID is required",
		},
		{
			name:          "missing region",
			workerEnvID:   validWorkerEnvID,
			exportEnvID:   validExportEnvID,
			region:        "",
			clientID:      validClientID,
			clientSecret:  validClientSecret,
			expectError:   true,
			errorContains: "region is required",
		},
		{
			name:          "invalid region",
			workerEnvID:   validWorkerEnvID,
			exportEnvID:   validExportEnvID,
			region:        "XX",
			clientID:      validClientID,
			clientSecret:  validClientSecret,
			expectError:   true,
			errorContains: "invalid region: XX",
		},
		{
			name:          "missing client ID",
			workerEnvID:   validWorkerEnvID,
			exportEnvID:   validExportEnvID,
			region:        "NA",
			clientID:      "",
			clientSecret:  validClientSecret,
			expectError:   true,
			errorContains: "client ID is required",
		},
		{
			name:          "missing client secret",
			workerEnvID:   validWorkerEnvID,
			exportEnvID:   validExportEnvID,
			region:        "NA",
			clientID:      validClientID,
			clientSecret:  "",
			expectError:   true,
			errorContains: "client secret is required",
		},
		{
			name:          "invalid export environment ID format",
			workerEnvID:   validWorkerEnvID,
			exportEnvID:   "not-a-uuid",
			region:        "NA",
			clientID:      validClientID,
			clientSecret:  validClientSecret,
			expectError:   true,
			errorContains: "invalid export environment ID format",
		},
		{
			name:         "valid credentials",
			workerEnvID:  validWorkerEnvID,
			exportEnvID:  validExportEnvID,
			region:       "NA",
			clientID:     validClientID,
			clientSecret: validClientSecret,
			expectError:  false,
		},
		{
			name:         "valid credentials AU region",
			workerEnvID:  validWorkerEnvID,
			exportEnvID:  "00000000-0000-0000-0000-000000000002",
			region:       "AU",
			clientID:     validClientID,
			clientSecret: validClientSecret,
			expectError:  false,
		},
		{
			name:         "valid credentials SG region",
			workerEnvID:  validWorkerEnvID,
			exportEnvID:  "00000000-0000-0000-0000-000000000003",
			region:       "SG",
			clientID:     validClientID,
			clientSecret: validClientSecret,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewFromCredentials(ctx, tt.workerEnvID, tt.exportEnvID, tt.region, tt.clientID, tt.clientSecret)

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

func TestAddWarning_DedupesExactDuplicates(t *testing.T) {
	c := &Client{}
	c.AddWarning("skipping application app-1: PingOne system application types are not exportable")
	c.AddWarning("skipping application app-1: PingOne system application types are not exportable")
	c.AddWarning("skipping application app-1: PingOne system application types are not exportable")

	assert.Equal(t, []string{"skipping application app-1: PingOne system application types are not exportable"}, c.Warnings())
}

func TestAddWarning_KeepsDistinctMessages(t *testing.T) {
	c := &Client{}
	c.AddWarning("skipping application app-1: not exportable")
	c.AddWarning("skipping application app-2: not exportable")

	assert.Equal(t, []string{
		"skipping application app-1: not exportable",
		"skipping application app-2: not exportable",
	}, c.Warnings())
}

func TestAddWarning_NoWarningsReturnsNil(t *testing.T) {
	c := &Client{}
	assert.Empty(t, c.Warnings())
}
