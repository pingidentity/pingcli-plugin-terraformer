package pingone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignOnPolicyResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_sign_on_policy"))
}

func TestSignOnPolicyResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_sign_on_policy"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

func TestListSignOnPolicies(t *testing.T) {
	tests := []struct {
		name         string
		responseBody map[string]any
		responseCode int
		wantIDs      []string
		wantErr      string
	}{
		{
			name: "multiple policies returned",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"signOnPolicies": []any{
						map[string]any{"id": "sop-1", "name": "Policy One"},
						map[string]any{"id": "sop-2", "name": "Policy Two"},
					},
				},
			},
			responseCode: http.StatusOK,
			wantIDs:      []string{"sop-1", "sop-2"},
		},
		{
			name:         "no embedded field returns empty",
			responseBody: map[string]any{"count": 0},
			responseCode: http.StatusOK,
			wantIDs:      nil,
		},
		{
			name: "embedded present but no signOnPolicies key",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"groups": []any{map[string]any{"id": "grp-1"}},
				},
			},
			responseCode: http.StatusOK,
			wantIDs:      nil,
		},
		{
			// 400 (not 500/503/429) avoids the management SDK's built-in
			// exponential-backoff retry loop.
			name:         "400 error returns wrapped error",
			responseBody: map[string]any{"message": "bad request"},
			responseCode: http.StatusBadRequest,
			wantErr:      "list sign-on policies",
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

			result, err := listSignOnPolicies(testCtx(), c, "")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			var gotIDs []string
			for _, item := range result {
				policy, ok := item.(*management.SignOnPolicy)
				require.True(t, ok, "expected *management.SignOnPolicy, got %T", item)
				gotIDs = append(gotIDs, policy.GetId())
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestListSignOnPolicies_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listSignOnPolicies(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetSignOnPolicy(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		responseBody map[string]any
		responseCode int
		wantName     string
		wantErr      string
	}{
		{
			name:       "valid policy returned",
			resourceID: "sop-1",
			responseBody: map[string]any{
				"id":   "sop-1",
				"name": "Policy One",
			},
			responseCode: http.StatusOK,
			wantName:     "Policy One",
		},
		{
			// 404 is not in the SDK's retryable-status list, so this fails fast.
			name:         "404 not found returns wrapped error",
			resourceID:   "missing",
			responseBody: map[string]any{"message": "not found"},
			responseCode: http.StatusNotFound,
			wantErr:      "get sign-on policy",
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

			result, err := getSignOnPolicy(testCtx(), c, "", tt.resourceID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			policy, ok := result.(*management.SignOnPolicy)
			require.True(t, ok)
			assert.Equal(t, tt.wantName, policy.GetName())
		})
	}
}

func TestGetSignOnPolicy_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getSignOnPolicy(testCtx(), c, "", "some-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
