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

// TestGroupResourceRegistered verifies the group handler is in the dispatch table.
func TestGroupResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_group"))
}

// TestGroupResourceHandlerFunctions verifies list and get functions are set.
func TestGroupResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_group"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newTestGroupManagementClient builds a management.APIClient pointed at the given
// httptest.Server so listGroups/getGroup can be exercised without a real
// OAuth exchange or network call.
func newTestGroupManagementClient(serverURL string) *management.APIClient {
	cfg := management.NewConfiguration()
	cfg.Servers = management.ServerConfigurations{
		{URL: serverURL, Variables: map[string]management.ServerVariable{}},
	}
	return management.NewAPIClient(cfg)
}

func TestListGroups(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      string
		wantGroupIDs []string
	}{
		{
			name: "single page of groups",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"_embedded": map[string]interface{}{
						"groups": []map[string]interface{}{
							{"id": "grp-1", "name": "Group One"},
							{"id": "grp-2", "name": "Group Two"},
						},
					},
				})
			},
			wantGroupIDs: []string{"grp-1", "grp-2"},
		},
		{
			name: "empty embedded returns nil result",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{})
			},
			wantGroupIDs: nil,
		},
		{
			name: "embedded present but no groups key",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"_embedded": map[string]interface{}{
						"populations": []map[string]interface{}{
							{"id": "pop-1"},
						},
					},
				})
			},
			wantGroupIDs: nil,
		},
		{
			// 400 is not in the SDK's retryable status code list (unlike 5xx/429),
			// so this fails fast instead of exercising the exponential backoff loop.
			name: "client error surfaces as list error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"boom"}`))
			},
			wantErr: "list groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := NewWithManagementClient(nil, newTestGroupManagementClient(srv.URL), uuid.New())

			result, err := listGroups(context.Background(), c, "")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)

			var gotIDs []string
			for _, item := range result {
				grp, ok := item.(*management.Group)
				require.True(t, ok, "expected *management.Group, got %T", item)
				gotIDs = append(gotIDs, grp.GetId())
			}
			assert.Equal(t, tt.wantGroupIDs, gotIDs)
		})
	}
}

func TestGetGroup(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		handler    http.HandlerFunc
		wantErr    string
		wantName   string
	}{
		{
			name:       "successful get",
			resourceID: "grp-1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":   "grp-1",
					"name": "Group One",
				})
			},
			wantName: "Group One",
		},
		{
			// 404 is not in the SDK's retryable status code list, so this
			// fails fast instead of exercising the exponential backoff loop.
			name:       "not found returns error",
			resourceID: "missing",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"not found"}`))
			},
			wantErr: "get group",
		},
		{
			name:       "empty resource ID still dispatches to API",
			resourceID: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":   "",
					"name": "",
				})
			},
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := NewWithManagementClient(nil, newTestGroupManagementClient(srv.URL), uuid.New())

			result, err := getGroup(context.Background(), c, "", tt.resourceID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)

			grp, ok := result.(*management.Group)
			require.True(t, ok, "expected *management.Group, got %T", result)
			assert.Equal(t, tt.wantName, grp.GetName())
		})
	}
}

func TestListGroups_ManagementClientError(t *testing.T) {
	// A Client with neither managementClient nor managementCfg set should
	// surface the "not configured" error from c.management(), not panic.
	c := &Client{environmentID: uuid.New()}

	_, err := listGroups(context.Background(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
}

func TestGetGroup_ManagementClientError(t *testing.T) {
	c := &Client{environmentID: uuid.New()}

	_, err := getGroup(context.Background(), c, "", "grp-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
}
