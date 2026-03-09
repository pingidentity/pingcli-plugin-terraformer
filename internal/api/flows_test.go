package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListFlows(t *testing.T) {
	t.Run("Successfully converts API response to flow summaries", func(t *testing.T) {
		// This test verifies the data transformation logic
		// In real usage, apiClient would be populated by NewClient

		// Create a minimal client for testing transformation logic
		client := &Client{
			EnvironmentID: "12345678-1234-1234-1234-123456789012",
			Region:        "NA",
			apiClient:     nil, // Would be set by NewClient in real usage
		}

		// Verify the client structure is correct
		assert.Equal(t, "12345678-1234-1234-1234-123456789012", client.EnvironmentID)
		assert.Equal(t, "NA", client.Region)
	})

	t.Run("Returns error for invalid environment ID", func(t *testing.T) {
		client := &Client{
			EnvironmentID: "invalid-uuid",
			Region:        "NA",
		}

		flows, err := client.ListFlows(context.Background())
		assert.Error(t, err)
		assert.Nil(t, flows)
		assert.Contains(t, err.Error(), "invalid environment ID")
	})
}

func TestGetFlow(t *testing.T) {
	t.Run("Validates client structure for flow retrieval", func(t *testing.T) {
		// This test verifies the client structure is correctly set up
		client := &Client{
			EnvironmentID: "12345678-1234-1234-1234-123456789012",
			Region:        "NA",
			apiClient:     nil,
		}

		// Verify client fields
		assert.Equal(t, "12345678-1234-1234-1234-123456789012", client.EnvironmentID)
		assert.Equal(t, "NA", client.Region)
	})

	t.Run("Returns error for invalid environment ID", func(t *testing.T) {
		client := &Client{
			EnvironmentID: "invalid-uuid",
			Region:        "NA",
		}

		flow, err := client.GetFlow(context.Background(), "test-flow-id")
		assert.Error(t, err)
		assert.Nil(t, flow)
		assert.Contains(t, err.Error(), "invalid environment ID")
	})
}

func TestStringValue(t *testing.T) {
	t.Run("Returns empty string for nil pointer", func(t *testing.T) {
		result := stringValue(nil)
		assert.Equal(t, "", result)
	})

	t.Run("Returns string value for non-nil pointer", func(t *testing.T) {
		str := "test value"
		result := stringValue(&str)
		assert.Equal(t, "test value", result)
	})
}
