package pingone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceSecretResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_resource_secret"))
}

func TestResourceSecretResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_resource_secret"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newResourceSecretMux serves /resources (list) and
// /resources/{resourceID}/secret (via ReadResourceSecret) from the same test
// server.
func newResourceSecretMux(resourcesBody map[string]any, secretsByResource map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if resourceID, ok := resourceIDFromSecretPath(r.URL.Path); ok {
			body, ok := secretsByResource[resourceID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(resourcesBody)
	})
	return httptest.NewServer(mux)
}

// resourceIDFromSecretPath extracts the resource ID from
// /environments/{envID}/resources/{resourceID}/secret.
func resourceIDFromSecretPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "resources" && i+2 < len(segments) && segments[i+2] == "secret" {
			return segments[i+1], true
		}
	}
	return "", false
}

func customResourcesBody(ids ...string) map[string]any {
	var resources []any
	for _, id := range ids {
		resources = append(resources, map[string]any{"id": id, "name": "Resource " + id, "type": "CUSTOM"})
	}
	return map[string]any{
		"_embedded": map[string]any{
			"resources": resources,
		},
	}
}

func TestListResourceSecrets(t *testing.T) {
	resourcesBody := customResourcesBody("res-1", "res-2")
	secretsByResource := map[string]any{
		"res-1": map[string]any{"secret": "s3cr3t-1"},
		"res-2": map[string]any{"secret": "s3cr3t-2"},
	}

	srv := newResourceSecretMux(resourcesBody, secretsByResource)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceSecrets(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	var gotIDs []string
	for _, item := range result {
		data, ok := item.(*resourceSecretData)
		require.True(t, ok, "expected *resourceSecretData, got %T", item)
		gotIDs = append(gotIDs, data.ID)
	}
	assert.ElementsMatch(t, []string{"res-1", "res-2"}, gotIDs)
}

func TestListResourceSecrets_SkipsResourcesWithoutSecret(t *testing.T) {
	resourcesBody := customResourcesBody("res-1", "res-2")
	secretsByResource := map[string]any{
		"res-1": map[string]any{"secret": "s3cr3t-1"},
		// res-2 intentionally omitted -> 404 -> should be skipped with a warning.
	}

	srv := newResourceSecretMux(resourcesBody, secretsByResource)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceSecrets(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	data, ok := result[0].(*resourceSecretData)
	require.True(t, ok)
	assert.Equal(t, "res-1", data.ID)
	assert.NotEmpty(t, c.Warnings())
}

func TestListResourceSecrets_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listResourceSecrets(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetResourceSecret(t *testing.T) {
	srv := newResourceSecretMux(map[string]any{}, map[string]any{
		"res-1": map[string]any{
			"secret": "s3cr3t-1",
			"previous": map[string]any{
				"secret":    "old-secret",
				"expiresAt": "2026-01-01T00:00:00Z",
				"lastUsed":  "2025-12-01T00:00:00Z",
			},
		},
	})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getResourceSecret(testCtx(), c, "", "res-1")
	require.NoError(t, err)

	data, ok := result.(*resourceSecretData)
	require.True(t, ok)
	assert.Equal(t, "res-1", data.ID)
	assert.Equal(t, "res-1", data.ResourceID)
	require.NotNil(t, data.Secret)
	assert.Equal(t, "s3cr3t-1", *data.Secret)
	require.NotNil(t, data.Previous)
	require.NotNil(t, data.Previous.Secret)
	assert.Equal(t, "old-secret", *data.Previous.Secret)
	assert.NotEmpty(t, data.Previous.ExpiresAt)
	require.NotNil(t, data.Previous.LastUsed)
}

func TestGetResourceSecret_NotFound(t *testing.T) {
	srv := newResourceSecretMux(map[string]any{}, map[string]any{})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getResourceSecret(testCtx(), c, "", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get resource secret")
	assert.Nil(t, result)
}

func TestGetResourceSecret_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getResourceSecret(testCtx(), c, "", "res-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
