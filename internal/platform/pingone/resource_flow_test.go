package pingone

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	pingone "github.com/pingidentity/pingone-go-client/pingone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exportRespBody builds a minimal valid DaVinciExportFlowVersionResponse JSON body.
func exportRespBody(variables []any) map[string]any {
	halLink := map[string]any{"href": "https://example.com"}
	return map[string]any{
		"_links": map[string]any{
			"environment": halLink,
			"self":        halLink,
		},
		"environment":      map[string]any{"id": "env-1"},
		"flow":             map[string]any{"id": "flow-1", "name": "Test"},
		"publishedVersion": 1,
		"version":          1,
		"variables":        variables,
	}
}

func TestExportFlowVersion(t *testing.T) {
	tests := []struct {
		name         string
		responseBody any
		responseCode int
		wantVarIDs   []string
		wantErr      string
	}{
		{
			name: "company and flowInstance variables returned",
			responseBody: exportRespBody([]any{
				map[string]any{"id": "var-a", "context": "company"},
				map[string]any{"id": "var-b", "context": "flowInstance"},
				map[string]any{"id": "var-c", "context": "node"},
			}),
			responseCode: http.StatusOK,
			wantVarIDs:   []string{"var-a", "var-b"},
		},
		{
			name: "no variables field",
			responseBody: func() map[string]any {
				b := exportRespBody(nil)
				delete(b, "variables")
				return b
			}(),
			responseCode: http.StatusOK,
			wantVarIDs:   nil,
		},
		{
			name:         "403 forbidden returns access denied error",
			responseBody: map[string]any{"message": "forbidden"},
			responseCode: http.StatusForbidden,
			wantErr:      "access denied",
		},
		{
			name:         "non-200 non-403 returns unexpected status error",
			responseBody: map[string]any{"message": "server error"},
			responseCode: http.StatusInternalServerError,
			wantErr:      "unexpected status 500",
		},
		{
			name:         "invalid json returns parse error",
			responseBody: nil,
			responseCode: http.StatusOK,
			wantErr:      "parse response JSON",
		},
		{
			name: "duplicate variable IDs returned as-is from SDK",
			responseBody: exportRespBody([]any{
				map[string]any{"id": "var-x", "context": "company"},
				map[string]any{"id": "var-x", "context": "company"},
			}),
			responseCode: http.StatusOK,
			wantVarIDs:   []string{"var-x", "var-x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/vnd.pingidentity.flowversion.export+json", r.Header.Get("Content-Type"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				} else {
					_, _ = w.Write([]byte("not json"))
				}
			}))
			defer srv.Close()

			u, err := url.Parse(srv.URL)
			require.NoError(t, err)

			variables, err := exportFlowVersion(context.Background(), u.Scheme, u.Host, "env-1", "flow-1", "1", srv.Client())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)

			var gotIDs []string
			for _, v := range variables {
				if id, _ := v["id"].(string); id != "" {
					ctx, _ := v["context"].(string)
					if ctx == "company" || ctx == "flowInstance" {
						gotIDs = append(gotIDs, id)
					}
				}
			}
			assert.Equal(t, tt.wantVarIDs, gotIDs)
		})
	}
}

func TestExtractVariableDeps(t *testing.T) {
	tests := []struct {
		name      string
		variables []map[string]interface{}
		wantIDs   []string
	}{
		{
			name:      "nil input",
			variables: nil,
			wantIDs:   nil,
		},
		{
			name:      "empty slice",
			variables: []map[string]interface{}{},
			wantIDs:   nil,
		},
		{
			name: "company and flowInstance included, node excluded",
			variables: []map[string]interface{}{
				{"id": "v1", "context": "company"},
				{"id": "v2", "context": "flowInstance"},
				{"id": "v3", "context": "node"},
			},
			wantIDs: []string{"v1", "v2"},
		},
		{
			name: "duplicates deduplicated",
			variables: []map[string]interface{}{
				{"id": "v1", "context": "company"},
				{"id": "v1", "context": "company"},
			},
			wantIDs: []string{"v1"},
		},
		{
			name: "empty id skipped",
			variables: []map[string]interface{}{
				{"id": "", "context": "company"},
				{"id": "v1", "context": "company"},
			},
			wantIDs: []string{"v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := extractVariableDeps(tt.variables)
			if tt.wantIDs == nil {
				assert.Nil(t, deps)
				return
			}
			var gotIDs []string
			for _, d := range deps {
				gotIDs = append(gotIDs, d.ResourceID)
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

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
// TestClearInputSchemaIfAuthentication verifies that clearInputSchemaIfAuthentication
// zeroes InputSchema when trigger type is AUTHENTICATION, and is otherwise a no-op.
func TestClearInputSchemaIfAuthentication(t *testing.T) {
	// A non-empty InputSchema slice used to confirm preservation in non-AUTHENTICATION cases.
	someSchema := []pingone.DaVinciFlowInputSchemaResponseItem{
		*pingone.NewDaVinciFlowInputSchemaResponseItem(
			"textField",
			pingone.DAVINCIFLOWINPUTSCHEMARESPONSEITEMPREFERREDDATATYPE_STRING,
			"username",
		),
	}

	tests := []struct {
		name            string
		detail          *pingone.DaVinciFlowResponse
		wantInputSchema []pingone.DaVinciFlowInputSchemaResponseItem
	}{
		{
			name: "AUTHENTICATION trigger clears InputSchema",
			detail: func() *pingone.DaVinciFlowResponse {
				d := pingone.NewDaVinciFlowResponseWithDefaults()
				trigger := pingone.NewDaVinciFlowTriggerResponse(pingone.DAVINCIFLOWTRIGGERRESPONSETYPE_AUTHENTICATION)
				d.SetTrigger(*trigger)
				d.SetInputSchema(someSchema)
				return d
			}(),
			wantInputSchema: nil,
		},
		{
			name: "SCHEDULE trigger leaves InputSchema unchanged",
			detail: func() *pingone.DaVinciFlowResponse {
				d := pingone.NewDaVinciFlowResponseWithDefaults()
				trigger := pingone.NewDaVinciFlowTriggerResponse(pingone.DAVINCIFLOWTRIGGERRESPONSETYPE_SCHEDULE)
				d.SetTrigger(*trigger)
				d.SetInputSchema(someSchema)
				return d
			}(),
			wantInputSchema: someSchema,
		},
		{
			name: "nil trigger leaves InputSchema unchanged",
			detail: func() *pingone.DaVinciFlowResponse {
				d := pingone.NewDaVinciFlowResponseWithDefaults()
				d.SetInputSchema(someSchema)
				// No trigger set — GetTriggerOk returns (nil, false).
				return d
			}(),
			wantInputSchema: someSchema,
		},
		{
			name:            "nil detail does not panic",
			detail:          nil,
			wantInputSchema: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearInputSchemaIfAuthentication(tt.detail)
			if tt.detail == nil {
				// Nothing to assert — just confirming no panic above.
				return
			}
			assert.Equal(t, tt.wantInputSchema, tt.detail.InputSchema)
		})
	}
}

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
