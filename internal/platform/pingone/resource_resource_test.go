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

// newTestManagementClient builds a management.APIClient pointed at a test
// server, bypassing the real OAuth exchange. All requests to it carry a
// fake bearer token via management.ContextAccessToken.
func newTestManagementClient(serverURL string) *management.APIClient {
	cfg := management.NewConfiguration()
	cfg.Servers = management.ServerConfigurations{
		{URL: serverURL},
	}
	return management.NewAPIClient(cfg)
}

func testCtx() context.Context {
	return context.WithValue(context.Background(), management.ContextAccessToken, "test-token")
}

func TestListResources(t *testing.T) {
	tests := []struct {
		name         string
		responseBody map[string]any
		responseCode int
		wantIDs      []string
		wantErr      string
	}{
		{
			name: "only CUSTOM resources returned, built-ins filtered out",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"resources": []any{
						map[string]any{"id": "res-custom-1", "name": "Custom One", "type": "CUSTOM"},
						map[string]any{"id": "res-oidc", "name": "openid", "type": "OPENID_CONNECT"},
						map[string]any{"id": "res-pingone-api", "name": "PingOne API", "type": "PINGONE_API"},
						map[string]any{"id": "res-custom-2", "name": "Custom Two", "type": "CUSTOM"},
					},
				},
			},
			responseCode: http.StatusOK,
			wantIDs:      []string{"res-custom-1", "res-custom-2"},
		},
		{
			name: "no embedded field returns empty",
			responseBody: map[string]any{
				"count": 0,
			},
			responseCode: http.StatusOK,
			wantIDs:      nil,
		},
		{
			name: "empty resources list returns empty",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"resources": []any{},
				},
			},
			responseCode: http.StatusOK,
			wantIDs:      nil,
		},
		{
			name: "resource missing type field is skipped",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"resources": []any{
						map[string]any{"id": "res-no-type", "name": "No Type"},
					},
				},
			},
			responseCode: http.StatusOK,
			wantIDs:      nil,
		},
		{
			// 400 is used (not 500/503/429) because those status codes
			// trigger the management SDK's built-in exponential-backoff
			// retry loop (up to 10 attempts, tens of seconds), which would
			// make this test extremely slow.
			name:         "400 error returns wrapped error",
			responseBody: map[string]any{"message": "bad request"},
			responseCode: http.StatusBadRequest,
			wantErr:      "list resources",
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

			mgmt := newTestManagementClient(srv.URL)
			c := NewWithManagementClient(nil, mgmt, uuid.New())

			result, err := listResources(testCtx(), c, "")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			var gotIDs []string
			for _, item := range result {
				res, ok := item.(*management.Resource)
				require.True(t, ok, "expected *management.Resource, got %T", item)
				gotIDs = append(gotIDs, res.GetId())
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestListResources_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listResources(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetResource(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		responseBody map[string]any
		responseCode int
		wantName     string
		wantErr      string
	}{
		{
			name:       "valid resource returned",
			resourceID: "res-custom-1",
			responseBody: map[string]any{
				"id":   "res-custom-1",
				"name": "Custom One",
				"type": "CUSTOM",
			},
			responseCode: http.StatusOK,
			wantName:     "Custom One",
		},
		{
			// 404 is not in the SDK's retryable-status list, so this fails
			// fast rather than triggering the backoff-retry loop.
			name:         "404 not found returns wrapped error",
			resourceID:   "missing",
			responseBody: map[string]any{"message": "not found"},
			responseCode: http.StatusNotFound,
			wantErr:      "get resource",
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

			mgmt := newTestManagementClient(srv.URL)
			c := NewWithManagementClient(nil, mgmt, uuid.New())

			result, err := getResource(testCtx(), c, "", tt.resourceID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			res, ok := result.(*management.Resource)
			require.True(t, ok)
			assert.Equal(t, tt.wantName, res.GetName())
		})
	}
}

func TestGetResource_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getResource(testCtx(), c, "", "some-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
