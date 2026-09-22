package pingone

import (
	"context"
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

func TestResourceScopeResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_resource_scope"))
}

func TestResourceScopeResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_resource_scope"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// TestListResourceScopes exercises the list-then-scan flow: an initial
// ResourcesApi call to find CUSTOM resources, then a ResourceScopesApi call
// per resource. Uses a single mux since both calls hit the same test server.
func TestListResourceScopes(t *testing.T) {
	tests := []struct {
		name           string
		resourcesBody  map[string]any
		scopesByParent map[string]map[string]any
		wantScopeIDs   []string
		wantErr        string
	}{
		{
			name: "scopes collected across multiple CUSTOM resources, built-ins skipped",
			resourcesBody: map[string]any{
				"_embedded": map[string]any{
					"resources": []any{
						map[string]any{"id": "res-1", "name": "Custom One", "type": "CUSTOM"},
						map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
						map[string]any{"id": "res-2", "name": "Custom Two", "type": "CUSTOM"},
					},
				},
			},
			scopesByParent: map[string]map[string]any{
				"res-1": {
					"_embedded": map[string]any{
						"scopes": []any{
							map[string]any{"id": "scope-1", "name": "read"},
						},
					},
				},
				"res-2": {
					"_embedded": map[string]any{
						"scopes": []any{
							map[string]any{"id": "scope-2", "name": "write"},
						},
					},
				},
			},
			wantScopeIDs: []string{"scope-1", "scope-2"},
		},
		{
			name: "no custom resources returns empty",
			resourcesBody: map[string]any{
				"_embedded": map[string]any{
					"resources": []any{
						map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
					},
				},
			},
			wantScopeIDs: nil,
		},
		{
			name: "custom resource with no scopes",
			resourcesBody: map[string]any{
				"_embedded": map[string]any{
					"resources": []any{
						map[string]any{"id": "res-1", "name": "Custom One", "type": "CUSTOM"},
					},
				},
			},
			scopesByParent: map[string]map[string]any{
				"res-1": {"count": 0},
			},
			wantScopeIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// Distinguish the resources list call from a scope-list call
				// by whether the path contains "/resources/{id}/scopes".
				if parentID, ok := parentIDFromScopesPath(r.URL.Path); ok {
					body, ok := tt.scopesByParent[parentID]
					require.True(t, ok, "unexpected scopes request for parent %s", parentID)
					_ = json.NewEncoder(w).Encode(body)
					return
				}
				_ = json.NewEncoder(w).Encode(tt.resourcesBody)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			mgmt := newTestResourceManagementClient(srv.URL)
			c := NewWithManagementClient(nil, mgmt, uuid.New())

			result, err := listResourceScopes(testCtx(), c, "")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			var gotIDs []string
			for _, item := range result {
				scope, ok := item.(*management.ResourceScope)
				require.True(t, ok, "expected *management.ResourceScope, got %T", item)
				gotIDs = append(gotIDs, scope.GetId())
			}
			assert.ElementsMatch(t, tt.wantScopeIDs, gotIDs)
		})
	}
}

// parentIDFromScopesPath extracts the resource ID from a scopes path, either
// the list form (/environments/{envID}/resources/{resourceID}/scopes) or the
// single-scope form (/environments/{envID}/resources/{resourceID}/scopes/{scopeID}).
// Returns ok=false for paths that aren't under a "/resources/.../scopes" segment
// (e.g. the plain resources-list path used to resolve built-in resource IDs).
func parentIDFromScopesPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "resources" && i+2 < len(segments) && segments[i+2] == "scopes" {
			return segments[i+1], true
		}
	}
	return "", false
}

func TestListResourceScopes_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listResourceScopes(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetResourceScope(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		responseBody map[string]any
		responseCode int
		wantName     string
		wantErr      string
	}{
		{
			name:       "valid composite ID returns scope",
			resourceID: "res-1/scope-1",
			responseBody: map[string]any{
				"id":   "scope-1",
				"name": "read",
			},
			responseCode: http.StatusOK,
			wantName:     "read",
		},
		{
			name:       "malformed composite ID returns error",
			resourceID: "no-slash",
			wantErr:    "resourceID must be resourceID/scopeID",
		},
		{
			name:         "404 not found returns wrapped error",
			resourceID:   "res-1/missing",
			responseBody: map[string]any{"message": "not found"},
			responseCode: http.StatusNotFound,
			wantErr:      "get resource scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseCode)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer srv.Close()

			mgmt := newTestResourceManagementClient(srv.URL)
			c := NewWithManagementClient(nil, mgmt, uuid.New())

			result, err := getResourceScope(testCtx(), c, "", tt.resourceID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			scope, ok := result.(*management.ResourceScope)
			require.True(t, ok)
			assert.Equal(t, tt.wantName, scope.GetName())
		})
	}
}

func TestGetResourceScope_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getResourceScope(testCtx(), c, "", "res-1/scope-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestListCustomResourceIDs_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listCustomResourceIDs(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
