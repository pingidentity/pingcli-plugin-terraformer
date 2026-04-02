package clients

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements APIClient for testing interface contract.
type mockClient struct {
	platform string
	data     map[string][]interface{}
}

func (m *mockClient) Platform() string { return m.platform }

func (m *mockClient) Warnings() []string { return nil }

func (m *mockClient) ListResources(_ context.Context, resourceType string, _ string) ([]interface{}, error) {
	if d, ok := m.data[resourceType]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
}

func (m *mockClient) GetResource(_ context.Context, resourceType string, _ string, resourceID string) (interface{}, error) {
	if d, ok := m.data[resourceType]; ok {
		for _, item := range d {
			if r, ok := item.(map[string]interface{}); ok && r["id"] == resourceID {
				return item, nil
			}
		}
		return nil, fmt.Errorf("resource not found: %s/%s", resourceType, resourceID)
	}
	return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
}

func TestAPIClientInterface(t *testing.T) {
	client := &mockClient{
		platform: "pingone",
		data: map[string][]interface{}{
			"pingone_davinci_variable": {
				map[string]interface{}{"id": "v1", "name": "myVar"},
				map[string]interface{}{"id": "v2", "name": "otherVar"},
			},
		},
	}

	// Verify interface conformance.
	var _ APIClient = client

	assert.Equal(t, "pingone", client.Platform())

	// ListResources
	resources, err := client.ListResources(context.Background(), "pingone_davinci_variable", "env-1")
	require.NoError(t, err)
	assert.Len(t, resources, 2)

	// GetResource found
	res, err := client.GetResource(context.Background(), "pingone_davinci_variable", "env-1", "v1")
	require.NoError(t, err)
	resMap, ok := res.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "myVar", resMap["name"])

	// GetResource not found
	_, err = client.GetResource(context.Background(), "pingone_davinci_variable", "env-1", "missing")
	require.Error(t, err)

	// ListResources unsupported type
	_, err = client.ListResources(context.Background(), "unsupported_type", "env-1")
	require.Error(t, err)
}
