package pingone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationAttributeMappingResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_application_attribute_mapping"))
}

func TestApplicationAttributeMappingResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_application_attribute_mapping"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newApplicationAttributeMappingMux serves /applications (list) and
// /applications/{id}/attributeMappings[/...] from the same test server.
func newApplicationAttributeMappingMux(applicationsBody map[string]any, mappingsByApp map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if appID, ok := appIDFromAttributeMappingsPath(r.URL.Path); ok {
			body, ok := mappingsByApp[appID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(applicationsBody)
	})
	return httptest.NewServer(mux)
}

// appIDFromAttributeMappingsPath extracts the application ID from
// /environments/{envID}/applications/{applicationID}/attributes[/{mappingID}]
// (the SDK's actual wire path for ApplicationAttributeMappingApi).
func appIDFromAttributeMappingsPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "applications" && i+2 < len(segments) && segments[i+2] == "attributes" {
			return segments[i+1], true
		}
	}
	return "", false
}

func TestListApplicationAttributeMappings(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1", "app-2")
	mappingsByApp := map[string]any{
		"app-1": map[string]any{
			"_embedded": map[string]any{
				"attributes": []any{
					map[string]any{
						"id": "map-1", "name": "email", "value": "${user.email}", "required": false,
						"application": map[string]any{"id": "app-1"},
					},
				},
			},
		},
		"app-2": map[string]any{
			"_embedded": map[string]any{
				"attributes": []any{
					map[string]any{
						"id": "map-2", "name": "sub", "value": "${user.id}", "required": true,
						"application": map[string]any{"id": "app-2"},
					},
				},
			},
		},
	}

	srv := newApplicationAttributeMappingMux(applicationsBody, mappingsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationAttributeMappings(testCtx(), c, "")
	require.NoError(t, err)

	var gotIDs []string
	for _, item := range result {
		mapping, ok := item.(*management.ApplicationAttributeMapping)
		require.True(t, ok, "expected *management.ApplicationAttributeMapping, got %T", item)
		gotIDs = append(gotIDs, mapping.GetId())
	}
	assert.ElementsMatch(t, []string{"map-1", "map-2"}, gotIDs)
}

func TestListApplicationAttributeMappings_NoMappings(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1")
	mappingsByApp := map[string]any{
		"app-1": map[string]any{"count": 0},
	}

	srv := newApplicationAttributeMappingMux(applicationsBody, mappingsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationAttributeMappings(testCtx(), c, "")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestListApplicationAttributeMappings_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listApplicationAttributeMappings(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetApplicationAttributeMapping(t *testing.T) {
	srv := newApplicationAttributeMappingMux(oidcApplicationsBody(), map[string]any{
		"app-1": map[string]any{"id": "map-1", "name": "email", "value": "${user.email}"},
	})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getApplicationAttributeMapping(testCtx(), c, "", "app-1/map-1")
	require.NoError(t, err)

	mapping, ok := result.(*management.ApplicationAttributeMapping)
	require.True(t, ok)
	assert.Equal(t, "email", mapping.GetName())
}

func TestGetApplicationAttributeMapping_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getApplicationAttributeMapping(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be applicationID/mappingID")
	assert.Nil(t, result)
}

func TestGetApplicationAttributeMapping_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getApplicationAttributeMapping(testCtx(), c, "", "app-1/map-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
