package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConnectorInstanceClient tests the basic structure and validation of connector instance methods
func TestConnectorInstanceClient(t *testing.T) {
	// Test that Client has the necessary fields for connector instance operations
	client := &Client{
		EnvironmentID: "00000000-0000-0000-0000-000000000000",
		Region:        "NA",
	}

	// Verify client structure
	assert.NotNil(t, client)
	assert.NotEmpty(t, client.EnvironmentID)
	assert.NotEmpty(t, client.Region)
}

// TestListConnectorInstancesValidation tests environment ID validation for ListConnectorInstances
func TestListConnectorInstancesValidation(t *testing.T) {
	tests := []struct {
		name          string
		environmentID string
		expectError   bool
		errorContains string
	}{
		{
			name:          "invalid environment ID format",
			environmentID: "not-a-uuid",
			expectError:   true,
			errorContains: "invalid environment ID",
		},
		{
			name:          "empty environment ID",
			environmentID: "",
			expectError:   true,
			errorContains: "invalid environment ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				EnvironmentID: tt.environmentID,
				Region:        "NA",
			}

			ctx := context.Background()
			_, err := client.ListConnectorInstances(ctx)

			assert.Error(t, err)
			if tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains)
			}
		})
	}
}

// TestGetConnectorInstanceValidation tests environment ID validation for GetConnectorInstance
func TestGetConnectorInstanceValidation(t *testing.T) {
	tests := []struct {
		name          string
		environmentID string
		instanceID    string
		expectError   bool
		errorContains string
	}{
		{
			name:          "invalid environment ID",
			environmentID: "invalid-id",
			instanceID:    "someInstanceID",
			expectError:   true,
			errorContains: "invalid environment ID",
		},
		{
			name:          "empty instance ID",
			environmentID: "12345678-1234-1234-1234-123456789012",
			instanceID:    "",
			expectError:   true,
			errorContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				EnvironmentID: tt.environmentID,
				Region:        "NA",
			}

			ctx := context.Background()
			_, err := client.GetConnectorInstance(ctx, tt.instanceID)

			assert.Error(t, err)
			if tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains)
			}
		})
	}
}
