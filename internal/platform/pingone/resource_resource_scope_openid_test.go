package pingone

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceScopeOpenIDResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_resource_scope_openid"))
}

func TestResourceScopeOpenIDResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_resource_scope_openid"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newBuiltInResourceMux builds a test server that serves a fixed resources
// list (used to resolve the built-in resource's ID) and a fixed scopes list
// keyed by the resolved parent resource ID.
func newBuiltInResourceMux(t *testing.T, resourcesBody map[string]any, scopesByParent map[string]map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if parentID, ok := parentIDFromScopesPath(r.URL.Path); ok {
			body, ok := scopesByParent[parentID]
			require.True(t, ok, "unexpected scopes request for parent %s", parentID)
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(resourcesBody)
	})
	return httptest.NewServer(mux)
}

func TestListResourceScopesOpenID(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
				map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
				map[string]any{"id": "res-api", "name": "PingOne API", "type": "PINGONE_API"},
			},
		},
	}
	scopesByParent := map[string]map[string]any{
		"res-oidc": {
			"_embedded": map[string]any{
				"scopes": []any{
					map[string]any{"id": "scope-email", "name": "email"},
					map[string]any{"id": "scope-profile", "name": "profile"},
				},
			},
		},
	}

	srv := newBuiltInResourceMux(t, resourcesBody, scopesByParent)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceScopesOpenID(testCtx(), c, "")
	require.NoError(t, err)

	var gotIDs []string
	for _, item := range result {
		scope, ok := item.(*management.ResourceScope)
		require.True(t, ok, "expected *management.ResourceScope, got %T", item)
		gotIDs = append(gotIDs, scope.GetId())
	}
	assert.ElementsMatch(t, []string{"scope-email", "scope-profile"}, gotIDs)
}

func TestListResourceScopesOpenID_BuiltInResourceNotFound(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
			},
		},
	}
	srv := newBuiltInResourceMux(t, resourcesBody, nil)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceScopesOpenID(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "built-in resource of type OPENID_CONNECT not found")
	assert.Nil(t, result)
}

func TestListResourceScopesOpenID_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listResourceScopesOpenID(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetResourceScopeOpenID(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
			},
		},
	}
	scopesByParent := map[string]map[string]any{
		"res-oidc": {"id": "scope-email", "name": "email"},
	}
	srv := newBuiltInResourceMux(t, resourcesBody, scopesByParent)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getResourceScopeOpenID(testCtx(), c, "", "scope-email")
	require.NoError(t, err)
	scope, ok := result.(*management.ResourceScope)
	require.True(t, ok)
	assert.Equal(t, "email", scope.GetName())
}

func TestGetResourceScopeOpenID_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getResourceScopeOpenID(testCtx(), c, "", "scope-email")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestFindBuiltInResourceID_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	id, err := findBuiltInResourceID(context.Background(), c, management.ENUMRESOURCETYPE_OPENID_CONNECT)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Empty(t, id)
}
