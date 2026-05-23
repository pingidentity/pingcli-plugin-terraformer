package pingone

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlowResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_davinci_flow"))
}

func TestFlowResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_davinci_flow"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// TestFetchFlowIDs verifies that fetchFlowIDs parses id and name from the
// ?attributes=id,name list response and returns one stub per flow.
func TestFetchFlowIDs(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  any
		responseCode  int
		wantLen       int
		wantFirstID   string
		wantFirstName string
		wantErr       string
	}{
		{
			name: "two flows returned",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"flows": []any{
						map[string]any{"id": "abc123", "name": "Flow One"},
						map[string]any{"id": "def456", "name": "Flow Two"},
					},
				},
			},
			responseCode:  http.StatusOK,
			wantLen:       2,
			wantFirstID:   "abc123",
			wantFirstName: "Flow One",
		},
		{
			name: "empty flow list",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"flows": []any{},
				},
			},
			responseCode: http.StatusOK,
			wantLen:      0,
		},
		{
			name:         "server error",
			responseBody: map[string]any{"message": "internal error"},
			responseCode: http.StatusInternalServerError,
			wantErr:      "unexpected status 500",
		},
		{
			name: "flow missing id is skipped",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"flows": []any{
						map[string]any{"name": "No ID Flow"},
						map[string]any{"id": "ghi789", "name": "Has ID"},
					},
				},
			},
			responseCode:  http.StatusOK,
			wantLen:       1,
			wantFirstID:   "ghi789",
			wantFirstName: "Has ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "attributes=id,name")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseCode)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer srv.Close()

			u, err := url.Parse(srv.URL)
			require.NoError(t, err)

			stubs, err := fetchFlowIDs(context.Background(), u.Scheme, u.Host, "test-env-id", srv.Client())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Len(t, stubs, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantFirstID, stubs[0].ID)
				assert.Equal(t, tt.wantFirstName, stubs[0].Name)
			}
		})
	}
}
