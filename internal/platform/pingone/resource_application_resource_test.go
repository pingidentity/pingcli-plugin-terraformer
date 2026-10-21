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

func TestApplicationResourceResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_application_resource"))
}

func TestApplicationResourceResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_application_resource"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newApplicationResourceMux serves /resources (list of CUSTOM resources) and
// /resources/{id}/applicationResources[/{appResourceID}] from the same test
// server.
func newApplicationResourceMux(t *testing.T, resourcesBody map[string]any, appResourcesByParent map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if parentID, ok := parentIDFromApplicationResourcesPath(r.URL.Path); ok {
			body, ok := appResourcesByParent[parentID]
			require.True(t, ok, "unexpected applicationResources request for parent %s", parentID)
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(resourcesBody)
	})
	return httptest.NewServer(mux)
}

// parentIDFromApplicationResourcesPath extracts the resource ID from
// /environments/{envID}/resources/{resourceID}/applicationResources[/{appResourceID}].
func parentIDFromApplicationResourcesPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "resources" && i+2 < len(segments) && segments[i+2] == "applicationResources" {
			return segments[i+1], true
		}
	}
	return "", false
}

func TestListApplicationResources(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-1", "name": "Custom One", "type": "CUSTOM"},
				map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
			},
		},
	}
	appResourcesByParent := map[string]any{
		"res-1": map[string]any{
			"_embedded": map[string]any{
				"resources": []any{
					map[string]any{"id": "app-res-1", "name": "Invoices", "parent": map[string]any{"id": "res-1", "type": "PING_ONE_RESOURCE"}},
				},
			},
		},
	}

	srv := newApplicationResourceMux(t, resourcesBody, appResourcesByParent)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationResources(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	appRes, ok := result[0].(*management.ResourceApplicationResource)
	require.True(t, ok, "expected *management.ResourceApplicationResource, got %T", result[0])
	assert.Equal(t, "app-res-1", appRes.GetId())
	assert.Equal(t, "Invoices", appRes.GetName())
}

func TestListApplicationResources_NoCustomResources(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
			},
		},
	}
	srv := newApplicationResourceMux(t, resourcesBody, nil)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationResources(testCtx(), c, "")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestListApplicationResources_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listApplicationResources(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetApplicationResource(t *testing.T) {
	srv := newApplicationResourceMux(t, map[string]any{}, map[string]any{
		"res-1": map[string]any{"id": "app-res-1", "name": "Invoices"},
	})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getApplicationResource(testCtx(), c, "", "res-1/app-res-1")
	require.NoError(t, err)

	appRes, ok := result.(*management.ResourceApplicationResource)
	require.True(t, ok)
	assert.Equal(t, "Invoices", appRes.GetName())
}

func TestGetApplicationResource_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getApplicationResource(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be resourceID/applicationResourceID")
	assert.Nil(t, result)
}

func TestGetApplicationResource_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getApplicationResource(testCtx(), c, "", "res-1/app-res-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
