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

func TestApplicationResourceGrantResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_application_resource_grant"))
}

func TestApplicationResourceGrantResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_application_resource_grant"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newApplicationResourceGrantMux serves /applications (list),
// /applications/{id}/grants[/{grantID}], and /resources/{id} (for the
// supplemental resource-type lookup) from the same test server.
func newApplicationResourceGrantMux(applicationsBody map[string]any, grantsByApp map[string]any, resourcesByID map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if resourceID, ok := resourceIDFromResourcesPath(r.URL.Path); ok {
			body, ok := resourcesByID[resourceID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		if appID, ok := appIDFromGrantsPath(r.URL.Path); ok {
			body, ok := grantsByApp[appID]
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

// appIDFromGrantsPath extracts the application ID from
// /environments/{envID}/applications/{applicationID}/grants[/{grantID}].
func appIDFromGrantsPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "applications" && i+2 < len(segments) && segments[i+2] == "grants" {
			return segments[i+1], true
		}
	}
	return "", false
}

// resourceIDFromResourcesPath extracts the resource ID from a direct
// single-resource path: /environments/{envID}/resources/{resourceID}
// (exactly 4 segments — distinguishes this from the resources-list path and
// from resource-scoped sub-paths like /resources/{id}/scopes).
func resourceIDFromResourcesPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 4 && segments[2] == "resources" {
		return segments[3], true
	}
	return "", false
}

func TestListApplicationResourceGrants(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1")
	grantsByApp := map[string]any{
		"app-1": map[string]any{
			"_embedded": map[string]any{
				"grants": []any{
					map[string]any{
						"id":       "grant-custom",
						"resource": map[string]any{"id": "res-custom"},
						"scopes":   []any{map[string]any{"id": "scope-1"}},
					},
					map[string]any{
						"id":       "grant-oidc",
						"resource": map[string]any{"id": "res-oidc"},
						"scopes":   []any{map[string]any{"id": "scope-2"}},
					},
				},
			},
		},
	}
	resourcesByID := map[string]any{
		"res-custom": map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
		"res-oidc":   map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
	}

	srv := newApplicationResourceGrantMux(applicationsBody, grantsByApp, resourcesByID)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationResourceGrants(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	byID := map[string]*applicationResourceGrantData{}
	for _, item := range result {
		data, ok := item.(*applicationResourceGrantData)
		require.True(t, ok, "expected *applicationResourceGrantData, got %T", item)
		byID[data.ID] = data
	}

	custom := byID["grant-custom"]
	require.NotNil(t, custom)
	assert.Equal(t, "CUSTOM", custom.ResourceType)
	assert.Equal(t, "res-custom", custom.CustomResourceID)
	assert.Equal(t, "res-custom", custom.ResourceID)
	assert.Equal(t, []string{"scope-1"}, custom.Scopes)

	oidc := byID["grant-oidc"]
	require.NotNil(t, oidc)
	assert.Equal(t, "OPENID_CONNECT", oidc.ResourceType)
	assert.Empty(t, oidc.CustomResourceID)
	assert.Equal(t, "res-oidc", oidc.ResourceID)
}

func TestListApplicationResourceGrants_SkipsGrantWhenResourceTypeLookupFails(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1")
	grantsByApp := map[string]any{
		"app-1": map[string]any{
			"_embedded": map[string]any{
				"grants": []any{
					map[string]any{
						"id":       "grant-missing-resource",
						"resource": map[string]any{"id": "res-gone"},
						"scopes":   []any{map[string]any{"id": "scope-1"}},
					},
				},
			},
		},
	}
	// res-gone intentionally omitted from resourcesByID -> 404 -> should be
	// skipped with a warning rather than failing the whole export.
	srv := newApplicationResourceGrantMux(applicationsBody, grantsByApp, map[string]any{})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationResourceGrants(testCtx(), c, "")
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NotEmpty(t, c.Warnings())
}

func TestListApplicationResourceGrants_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listApplicationResourceGrants(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetApplicationResourceGrant(t *testing.T) {
	srv := newApplicationResourceGrantMux(oidcApplicationsBody(), map[string]any{
		"app-1": map[string]any{
			"id":       "grant-custom",
			"resource": map[string]any{"id": "res-custom"},
			"scopes":   []any{map[string]any{"id": "scope-1"}},
		},
	}, map[string]any{
		"res-custom": map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
	})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getApplicationResourceGrant(testCtx(), c, "", "app-1/grant-custom")
	require.NoError(t, err)

	data, ok := result.(*applicationResourceGrantData)
	require.True(t, ok)
	assert.Equal(t, "CUSTOM", data.ResourceType)
	assert.Equal(t, "res-custom", data.CustomResourceID)
}

func TestGetApplicationResourceGrant_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getApplicationResourceGrant(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be applicationID/grantID")
	assert.Nil(t, result)
}

func TestGetApplicationResourceGrant_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getApplicationResourceGrant(testCtx(), c, "", "app-1/grant-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestResolveResourceType_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	_, err := c.management(testCtx())
	require.Error(t, err)
}

func TestToApplicationResourceGrantData_NonCustomOmitsCustomResourceID(t *testing.T) {
	grant := &management.ApplicationResourceGrant{
		Id:       strPtr("grant-1"),
		Resource: management.ApplicationResourceGrantResource{Id: "res-api"},
		Scopes: []management.ApplicationResourceGrantScopesInner{
			{Id: "scope-1"},
		},
	}
	data := toApplicationResourceGrantData("app-1", "env-1", grant, management.ENUMRESOURCETYPE_PINGONE_API)
	assert.Equal(t, "PINGONE_API", data.ResourceType)
	assert.Empty(t, data.CustomResourceID)
	assert.Equal(t, "res-api", data.ResourceID)
	assert.Equal(t, []string{"scope-1"}, data.Scopes)
}
