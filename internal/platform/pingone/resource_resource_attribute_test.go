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

func TestResourceAttributeResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_resource_attribute"))
}

func TestResourceAttributeResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_resource_attribute"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newResourceAttributeMux serves /resources (list), /resources/{id} (single
// get, used by the supplemental resource-type lookup), and
// /resources/{resourceID}/attributes[/{attrID}] from the same test server.
func newResourceAttributeMux(resourcesBody map[string]any, resourcesByID map[string]any, attributesByResource map[string]map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if resourceID, ok := parentIDFromAttributesPath(r.URL.Path); ok {
			body, ok := attributesByResource[resourceID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(body)
			return
		}
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
		_ = json.NewEncoder(w).Encode(resourcesBody)
	})
	return httptest.NewServer(mux)
}

// parentIDFromAttributesPath extracts the resource ID from
// /environments/{envID}/resources/{resourceID}/attributes[/{attrID}].
func parentIDFromAttributesPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "resources" && i+2 < len(segments) && segments[i+2] == "attributes" {
			return segments[i+1], true
		}
	}
	return "", false
}

func TestListResourceAttributes(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
				map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
			},
		},
	}
	attributesByResource := map[string]map[string]any{
		"res-custom": {
			"_embedded": map[string]any{
				"attributes": []any{
					map[string]any{
						"id": "attr-1", "name": "example_attribute", "value": "${user.name.family}",
						"type":     "CUSTOM",
						"resource": map[string]any{"id": "res-custom"},
					},
				},
			},
		},
		"res-oidc": {
			"_embedded": map[string]any{
				"attributes": []any{
					map[string]any{
						"id": "attr-2", "name": "example_oidc_attribute", "value": "${user.customAttribute}",
						"type":     "CUSTOM",
						"resource": map[string]any{"id": "res-oidc"},
					},
				},
			},
		},
	}

	srv := newResourceAttributeMux(resourcesBody, nil, attributesByResource)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceAttributes(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	byID := map[string]*resourceAttributeData{}
	for _, item := range result {
		data, ok := item.(*resourceAttributeData)
		require.True(t, ok, "expected *resourceAttributeData, got %T", item)
		byID[data.ID] = data
	}

	custom := byID["attr-1"]
	require.NotNil(t, custom)
	assert.Equal(t, "CUSTOM", custom.ResourceType)
	assert.Equal(t, "res-custom", custom.CustomResourceID)
	assert.Equal(t, "res-custom", custom.ResourceID)

	oidc := byID["attr-2"]
	require.NotNil(t, oidc)
	assert.Equal(t, "OPENID_CONNECT", oidc.ResourceType)
	assert.Empty(t, oidc.CustomResourceID)
	assert.Equal(t, "res-oidc", oidc.ResourceID)
}

// TestListResourceAttributes_SkipsCoreAndPredefined confirms that CORE and
// PREDEFINED attributes (e.g. "sub", "given_name") are excluded — they exist
// automatically on every resource without ever being created, and their
// value is a provider-side hardcoded default rather than something the API
// returns, so exporting them would either omit the required value field or
// export a meaningless empty string.
func TestListResourceAttributes_SkipsCoreAndPredefined(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
				map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
			},
		},
	}
	attributesByResource := map[string]map[string]any{
		"res-custom": {
			"_embedded": map[string]any{
				"attributes": []any{
					map[string]any{
						"id": "attr-core", "name": "sub", "value": "${user.id}",
						"type":     "CORE",
						"resource": map[string]any{"id": "res-custom"},
					},
					map[string]any{
						"id": "attr-custom", "name": "example_attribute", "value": "${user.name.family}",
						"type":     "CUSTOM",
						"resource": map[string]any{"id": "res-custom"},
					},
				},
			},
		},
		"res-oidc": {
			"_embedded": map[string]any{
				"attributes": []any{
					map[string]any{
						"id": "attr-predefined", "name": "given_name", "value": "${user.name.given}",
						"type":     "PREDEFINED",
						"resource": map[string]any{"id": "res-oidc"},
					},
				},
			},
		},
	}

	srv := newResourceAttributeMux(resourcesBody, nil, attributesByResource)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listResourceAttributes(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	data, ok := result[0].(*resourceAttributeData)
	require.True(t, ok)
	assert.Equal(t, "attr-custom", data.ID)
}

func TestListResourceAttributes_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listResourceAttributes(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetResourceAttribute(t *testing.T) {
	resourcesBody := map[string]any{
		"_embedded": map[string]any{
			"resources": []any{
				map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
			},
		},
	}
	attributesByResource := map[string]map[string]any{
		"res-custom": {
			"id": "attr-1", "name": "example_attribute", "value": "${user.name.family}",
			"resource": map[string]any{"id": "res-custom"},
		},
	}
	resourcesByID := map[string]any{
		"res-custom": map[string]any{"id": "res-custom", "name": "Custom", "type": "CUSTOM"},
	}
	srv := newResourceAttributeMux(resourcesBody, resourcesByID, attributesByResource)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getResourceAttribute(testCtx(), c, "", "res-custom/attr-1")
	require.NoError(t, err)

	data, ok := result.(*resourceAttributeData)
	require.True(t, ok)
	assert.Equal(t, "CUSTOM", data.ResourceType)
	assert.Equal(t, "res-custom", data.CustomResourceID)
}

func TestGetResourceAttribute_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getResourceAttribute(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be resourceID/attributeID")
	assert.Nil(t, result)
}

func TestGetResourceAttribute_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getResourceAttribute(testCtx(), c, "", "res-1/attr-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
