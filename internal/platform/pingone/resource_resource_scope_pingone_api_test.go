package pingone

import (
	"testing"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceScopePingOneAPIResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_resource_scope_pingone_api"))
}

func TestResourceScopePingOneAPIResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_resource_scope_pingone_api"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

func TestListResourceScopesPingOneAPI(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
				map[string]any{"id": "res-api", "name": "PingOne API", "type": "PINGONE_API"},
			},
		},
	}
	scopesByParent := map[string]map[string]any{
		"res-api": {
			"_embedded": map[string]any{
				"scopes": []any{
					map[string]any{"id": "scope-read-user", "name": "p1:read:user"},
				},
			},
		},
	}

	srv := newBuiltInResourceMux(t, resourcesBody, scopesByParent)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceScopesPingOneAPI(testCtx(), c, "")
	require.NoError(t, err)

	var gotIDs []string
	for _, item := range result {
		scope, ok := item.(*management.ResourceScope)
		require.True(t, ok, "expected *management.ResourceScope, got %T", item)
		gotIDs = append(gotIDs, scope.GetId())
	}
	assert.Equal(t, []string{"scope-read-user"}, gotIDs)
}

// TestListResourceScopesPingOneAPI_FiltersUnmanageableScopes confirms that
// built-in PINGONE_API scopes other than p1:read:user/p1:update:user (and
// their :{suffix} variants) are silently excluded rather than exported — the
// provider's own name validator rejects every other scope, so exporting one
// would produce HCL that fails terraform validate. Silent (no warning)
// because these built-ins are identical across every PingOne environment.
func TestListResourceScopesPingOneAPI_FiltersUnmanageableScopes(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-api", "name": "PingOne API", "type": "PINGONE_API"},
			},
		},
	}
	scopesByParent := map[string]map[string]any{
		"res-api": {
			"_embedded": map[string]any{
				"scopes": []any{
					map[string]any{"id": "scope-read-user", "name": "p1:read:user"},
					map[string]any{"id": "scope-update-user-suffix", "name": "p1:update:user:custom"},
					map[string]any{"id": "scope-read-device", "name": "p1:read:device"},
					map[string]any{"id": "scope-create-pairingkey", "name": "p1:create:pairingKey"},
				},
			},
		},
	}

	srv := newBuiltInResourceMux(t, resourcesBody, scopesByParent)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceScopesPingOneAPI(testCtx(), c, "")
	require.NoError(t, err)

	var gotNames []string
	for _, item := range result {
		scope, ok := item.(*management.ResourceScope)
		require.True(t, ok, "expected *management.ResourceScope, got %T", item)
		gotNames = append(gotNames, scope.GetName())
	}
	assert.ElementsMatch(t, []string{"p1:read:user", "p1:update:user:custom"}, gotNames)
	assert.Empty(t, c.Warnings())
}

func TestListResourceScopesPingOneAPI_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listResourceScopesPingOneAPI(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetResourceScopePingOneAPI(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-api", "name": "PingOne API", "type": "PINGONE_API"},
			},
		},
	}
	scopesByParent := map[string]map[string]any{
		"res-api": {"id": "scope-read-user", "name": "p1:read:user"},
	}
	srv := newBuiltInResourceMux(t, resourcesBody, scopesByParent)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getResourceScopePingOneAPI(testCtx(), c, "", "scope-read-user")
	require.NoError(t, err)
	scope, ok := result.(*management.ResourceScope)
	require.True(t, ok)
	assert.Equal(t, "p1:read:user", scope.GetName())
}

func TestGetResourceScopePingOneAPI_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getResourceScopePingOneAPI(testCtx(), c, "", "scope-read-user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
