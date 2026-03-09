package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestListFlowPolicies tests the flow policy listing functionality
func TestListFlowPolicies(t *testing.T) {
	tests := []struct {
		name          string
		client        *Client
		expectError   bool
		errorContains string
	}{
		{
			name: "Validates environment ID format",
			client: &Client{
				EnvironmentID: "invalid-uuid",
			},
			expectError:   true,
			errorContains: "invalid environment ID",
		},
		{
			name: "Requires valid client structure",
			client: &Client{
				EnvironmentID: "12345678-1234-1234-1234-123456789012",
				Region:        "NA",
			},
			expectError: false, // Will be skipped due to nil apiClient
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Skip test if expecting API call error with nil client
			if tt.client.apiClient == nil && !tt.expectError {
				t.Skip("Skipping - requires full API client initialization, covered by acceptance tests")
				return
			}

			_, err := tt.client.ListFlowPolicies(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetFlowPolicy tests the flow policy detail retrieval functionality
func TestGetFlowPolicy(t *testing.T) {
	tests := []struct {
		name          string
		client        *Client
		applicationID string
		policyID      string
		expectError   bool
		errorContains string
	}{
		{
			name: "Validates environment ID format",
			client: &Client{
				EnvironmentID: "invalid-uuid",
			},
			applicationID: "app123",
			policyID:      "policy123",
			expectError:   true,
			errorContains: "invalid environment ID",
		},
		{
			name: "Validates application ID is not empty",
			client: &Client{
				EnvironmentID: "12345678-1234-1234-1234-123456789012",
			},
			applicationID: "",
			policyID:      "policy123",
			expectError:   true,
			errorContains: "application ID cannot be empty",
		},
		{
			name: "Validates policy ID is not empty",
			client: &Client{
				EnvironmentID: "12345678-1234-1234-1234-123456789012",
			},
			applicationID: "app123",
			policyID:      "",
			expectError:   true,
			errorContains: "flow policy ID cannot be empty",
		},
		{
			name: "Requires valid client structure for API call",
			client: &Client{
				EnvironmentID: "12345678-1234-1234-1234-123456789012",
				Region:        "NA",
			},
			applicationID: "app123",
			policyID:      "policy123",
			expectError:   false, // Will be skipped due to nil apiClient
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Skip test if expecting API call error with nil client
			if tt.client.apiClient == nil && !tt.expectError {
				t.Skip("Skipping - requires full API client initialization, covered by acceptance tests")
				return
			}

			_, err := tt.client.GetFlowPolicy(ctx, tt.applicationID, tt.policyID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
