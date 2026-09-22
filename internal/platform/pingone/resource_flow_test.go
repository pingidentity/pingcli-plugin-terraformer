package pingone

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	pingone "github.com/pingidentity/pingone-go-client/pingone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/graph"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
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

// ── Embedded theme reference behavioral tests ──────────────────
//
// These exercise the two pingone_branding_theme rules registered in init()
// (theme.value / case 1, themeId.value / case 3) end-to-end via
// core.ResolveEmbeddedReferences, using realistic showForm node JSON shaped
// like the three evidence samples in context.md.

// themeFlowResourceDef returns a minimal *schema.ResourceDefinition for
// "pingone_davinci_flow", matching the shape used by the generic core tests.
func themeFlowResourceDef() *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			ResourceType: "pingone_davinci_flow",
		},
	}
}

// buildShowFormPropertiesJSON builds a jsonencode(...) RawHCLValue blob for a
// single showForm node, given raw (unescaped) JSON fragments for the "form",
// "theme", and "themeId" keys. Any of formJSON/themeJSON/themeIDJSON may be
// empty, in which case that key is omitted entirely (case 2 / customForm).
func buildShowFormPropertiesJSON(formJSON, themeJSON, themeIDJSON, nodeTitle string) core.RawHCLValue {
	var parts []string
	if formJSON != "" {
		parts = append(parts, `"form": `+formJSON)
	}
	if themeJSON != "" {
		parts = append(parts, `"theme": `+themeJSON)
	}
	if themeIDJSON != "" {
		parts = append(parts, `"themeId": `+themeIDJSON)
	}
	if nodeTitle != "" {
		parts = append(parts, `"nodeTitle": {"value": "`+nodeTitle+`"}`)
	}
	return core.RawHCLValue("jsonencode({" + strings.Join(parts, ", ") + "})")
}

// richTextWrapperJSON builds the escaped-string JSON value for a Slate-style
// rich-text wrapper embedding uuidStr, matching the double-JSON-encoding
// transformJSONEncodeRaw produces in production (mirrors context.md's
// themeId.value evidence sample).
func richTextWrapperJSON(t *testing.T, uuidStr string) string {
	t.Helper()
	wrapper := `[{"children":[{"text":"` + uuidStr + `"}]}]`
	escaped, err := json.Marshal(wrapper)
	require.NoError(t, err)
	return string(escaped)
}

// resolveShowFormNode runs core.ResolveEmbeddedReferences against a single
// showForm node's properties using the real registered pingone rules, and
// returns the resolved RawHCLValue plus any FallbackVariables produced.
func resolveShowFormNode(t *testing.T, g *graph.DependencyGraph, properties core.RawHCLValue) (core.RawHCLValue, []core.FallbackVariable) {
	t.Helper()

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": properties,
						},
					},
				},
			},
		},
	}

	resourceData := &core.ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &core.ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   themeFlowResourceDef(),
		Resources:    []*core.ResourceData{resourceData},
	}

	fallbackVars := core.ResolveEmbeddedReferences([]*core.ExportedResourceData{exportedData}, g, embeddedRefRules)

	resolved := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(core.RawHCLValue)
	return resolved, fallbackVars
}

// TestThemeRule_Case1_UUIDNotInGraph_FallsBackToVariable covers acceptance
// criterion: a showForm node with theme.value set to a UUID not present in
// the graph resolves to a Terraform variable fallback.
func TestThemeRule_Case1_UUIDNotInGraph_FallsBackToVariable(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	const themeUUID = "e6fd37f9-11dd-40f3-90f6-eaeb971ee3db"
	properties := buildShowFormPropertiesJSON(
		"",
		`{"value": "`+themeUUID+`"}`,
		"",
		"Sign On",
	)

	resolved, fallbackVars := resolveShowFormNode(t, g, properties)

	assert.Contains(t, string(resolved), "${var.davinci_theme_sign_on}")
	assert.NotContains(t, string(resolved), themeUUID)
	require.Len(t, fallbackVars, 1)
	assert.Equal(t, "davinci_theme_sign_on", fallbackVars[0].Name)
	assert.Equal(t, "pingone_branding_theme", fallbackVars[0].ResourceType)
	assert.Equal(t, themeUUID, fallbackVars[0].Default)
}

// TestThemeRule_Case1_UUIDInGraph_ResolvesToReference covers acceptance
// criterion: the same case-1 scenario but with a matching pingone_branding_theme
// resource present in the graph resolves to a resource reference with a graph
// edge, and no FallbackVariable.
func TestThemeRule_Case1_UUIDInGraph_ResolvesToReference(t *testing.T) {
	g := graph.New()
	const themeUUID = "e6fd37f9-11dd-40f3-90f6-eaeb971ee3db"
	g.AddResource("pingone_branding_theme", themeUUID, "pingcli__My_Theme")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	properties := buildShowFormPropertiesJSON(
		"",
		`{"value": "`+themeUUID+`"}`,
		"",
		"Sign On",
	)

	resolved, fallbackVars := resolveShowFormNode(t, g, properties)

	assert.Contains(t, string(resolved), "${pingone_branding_theme.pingcli__My_Theme.id}")
	assert.NotContains(t, string(resolved), themeUUID)
	assert.Empty(t, fallbackVars)

	deps := g.GetDependencies("pingone_davinci_flow", "parent-flow-id")
	require.NotEmpty(t, deps)
	found := false
	for _, d := range deps {
		if d.To.ResourceType == "pingone_branding_theme" && d.To.ID == themeUUID {
			found = true
		}
	}
	assert.True(t, found, "expected a graph edge to the pingone_branding_theme resource")
}

// TestThemeRule_Case2_ThemeAbsent_Unchanged covers acceptance criterion: a
// showForm node where theme is absent entirely is emitted with byte-for-byte
// unchanged properties JSON.
func TestThemeRule_Case2_ThemeAbsent_Unchanged(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	properties := buildShowFormPropertiesJSON(
		"",
		"",
		"",
		"Sign On",
	)

	resolved, fallbackVars := resolveShowFormNode(t, g, properties)

	assert.Equal(t, properties, resolved, "expected properties JSON to remain byte-for-byte unchanged")
	assert.Empty(t, fallbackVars)
}

// TestThemeRule_Case3_UUIDNotInGraph_FallsBackToVariable covers acceptance
// criterion: theme.value == "useThemeId" with themeId.value holding the
// rich-text-wrapped UUID resolves the UUID inside the wrapper to a variable
// fallback when not in the graph, preserves the wrapper structure, and
// leaves theme.value unchanged.
func TestThemeRule_Case3_UUIDNotInGraph_FallsBackToVariable(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	const themeUUID = "abc12300-0000-4000-8000-000000000123"
	properties := buildShowFormPropertiesJSON(
		"",
		`{"value": "useThemeId"}`,
		`{"value": `+richTextWrapperJSON(t, themeUUID)+`}`,
		"Sign On",
	)

	resolved, fallbackVars := resolveShowFormNode(t, g, properties)

	assert.Contains(t, string(resolved), `"useThemeId"`, "theme.value must remain the literal string \"useThemeId\"")
	assert.Contains(t, string(resolved), "${var.davinci_theme_sign_on}")
	assert.NotContains(t, string(resolved), themeUUID)
	require.Len(t, fallbackVars, 1)
	assert.Equal(t, "davinci_theme_sign_on", fallbackVars[0].Name)

	// Verify the wrapper's outer JSON structure round-trips with only the
	// inner value swapped.
	jsonStr := extractPropertiesJSON(t, resolved)
	var outer map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &outer))
	assert.Equal(t, "useThemeId", outer["theme"].(map[string]interface{})["value"])

	themeIDValue := outer["themeId"].(map[string]interface{})["value"].(string)
	var wrapperArr []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(themeIDValue), &wrapperArr))
	require.Len(t, wrapperArr, 1)
	children := wrapperArr[0]["children"].([]interface{})
	text := children[0].(map[string]interface{})["text"].(string)
	assert.Equal(t, "${var.davinci_theme_sign_on}", text)
}

// TestThemeRule_Case3_UUIDInGraph_ResolvesToReference covers acceptance
// criterion: the same case-3 scenario but with a matching pingone_branding_theme
// resource in the graph resolves the wrapped UUID to a resource reference
// with a graph edge, and theme.value still remains "useThemeId".
func TestThemeRule_Case3_UUIDInGraph_ResolvesToReference(t *testing.T) {
	g := graph.New()
	const themeUUID = "abc12300-0000-4000-8000-000000000456"
	g.AddResource("pingone_branding_theme", themeUUID, "pingcli__My_Theme")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	properties := buildShowFormPropertiesJSON(
		"",
		`{"value": "useThemeId"}`,
		`{"value": `+richTextWrapperJSON(t, themeUUID)+`}`,
		"Sign On",
	)

	resolved, fallbackVars := resolveShowFormNode(t, g, properties)

	assert.Contains(t, string(resolved), `"useThemeId"`)
	assert.Contains(t, string(resolved), "${pingone_branding_theme.pingcli__My_Theme.id}")
	assert.NotContains(t, string(resolved), themeUUID)
	assert.Empty(t, fallbackVars)

	deps := g.GetDependencies("pingone_davinci_flow", "parent-flow-id")
	found := false
	for _, d := range deps {
		if d.To.ResourceType == "pingone_branding_theme" && d.To.ID == themeUUID {
			found = true
		}
	}
	assert.True(t, found, "expected a graph edge to the pingone_branding_theme resource")
}

// TestThemeRule_ActiveThemeSentinel_NoThemeId_Unchanged covers acceptance
// criterion: theme.value == "activeTheme" with no themeId key present is
// emitted completely unchanged, because "activeTheme" is not UUID-shaped and
// is rejected by the unconditional format guard, not an enumerated skip list.
func TestThemeRule_ActiveThemeSentinel_NoThemeId_Unchanged(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")
	// Deliberately register a resource keyed by the literal sentinel string:
	// if the format guard were bypassed, this would let the rule resolve to
	// a reference, which would make the test fail loudly rather than
	// silently passing for the wrong reason.
	g.AddResource("pingone_branding_theme", "activeTheme", "pingcli__Should_Not_Resolve")

	properties := buildShowFormPropertiesJSON(
		"",
		`{"value": "activeTheme"}`,
		"",
		"Sign On",
	)

	resolved, fallbackVars := resolveShowFormNode(t, g, properties)

	assert.Equal(t, properties, resolved, "expected properties JSON to remain byte-for-byte unchanged")
	assert.Empty(t, fallbackVars)
}

// TestThemeRule_FormValueUnaffectedByThemeCase covers acceptance criterion: a
// node with both form.value and any of the three theme shapes resolves
// form.value identically regardless of which theme case is present.
func TestThemeRule_FormValueUnaffectedByThemeCase(t *testing.T) {
	const formUUID = "860b5cd5-45cc-466d-abbd-64298bb90bed"

	cases := []struct {
		name       string
		themeJSON  string
		themeIDVal string
	}{
		{"case1 direct theme UUID", `{"value": "e6fd37f9-11dd-40f3-90f6-eaeb971ee3db"}`, ""},
		{"case2 theme absent", "", ""},
		{"case3 useThemeId indirect", `{"value": "useThemeId"}`, "abc12300-0000-4000-8000-000000000789"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.New()
			g.AddResource("pingone_davinci_form", formUUID, "pingcli__Example_Sign_On")
			g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

			themeIDJSON := ""
			if tc.themeIDVal != "" {
				themeIDJSON = `{"value": ` + richTextWrapperJSON(t, tc.themeIDVal) + `}`
			}

			properties := buildShowFormPropertiesJSON(
				`{"value": "`+formUUID+`"}`,
				tc.themeJSON,
				themeIDJSON,
				"Sign On",
			)

			resolved, _ := resolveShowFormNode(t, g, properties)

			assert.Contains(t, string(resolved), "${pingone_davinci_form.pingcli__Example_Sign_On.id}")
			assert.NotContains(t, string(resolved), formUUID)
		})
	}
}

// TestThemeRule_CustomFormCapability_Unchanged covers acceptance criterion:
// a customForm-capability node (no theme/themeId keys) is emitted unchanged
// — no rule fires, no capability-aware branching.
func TestThemeRule_CustomFormCapability_Unchanged(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")
	g.AddResource("pingone_davinci_form", "ba30f833-e6a5-4fda-9ff1-2576ece5108c", "pingcli__Legacy_Form")

	// customForm's property set has no theme/themeId keys.
	properties := buildShowFormPropertiesJSON(
		`{"value": "ba30f833-e6a5-4fda-9ff1-2576ece5108c"}`,
		"",
		"",
		"",
	)

	resolved, fallbackVars := resolveShowFormNode(t, g, properties)

	// form.value still resolves (unrelated to the theme rules); theme rules
	// simply never fire because theme/themeId keys don't exist.
	assert.Contains(t, string(resolved), "${pingone_davinci_form.pingcli__Legacy_Form.id}")
	assert.NotContains(t, string(resolved), "davinci_theme")
	assert.NotContains(t, string(resolved), "pingone_branding_theme")
	assert.Empty(t, fallbackVars)
}

// extractPropertiesJSON strips the jsonencode(...) wrapper from a resolved
// RawHCLValue, returning the inner JSON string.
func extractPropertiesJSON(t *testing.T, value core.RawHCLValue) string {
	t.Helper()
	str := string(value)
	const prefix = "jsonencode("
	require.True(t, strings.HasPrefix(str, prefix), "expected jsonencode(...) wrapper, got: %s", str)
	return str[len(prefix) : len(str)-1]
}

// TestWrapFlowFetchError verifies that both getFlow and listFlows route
// GetFlowById errors through a single shared helper, so a legacy jsLinks
// unmarshal failure always produces the actionable hint (issue #10) instead
// of the raw SDK error, regardless of which call site hit it.
func TestWrapFlowFetchError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "legacy jsLinks unmarshal error is replaced with the hint",
			err:     errors.New(`json: cannot unmarshal string into Go struct field _DaVinciFlowResponse.settings.jsLinks of type map[string]interface {}`),
			wantMsg: jsLinksLegacyHint,
		},
		{
			name:    "unrelated error is passed through unchanged",
			err:     errors.New("unexpected status 500"),
			wantMsg: "unexpected status 500",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapFlowFetchError(tt.err)
			require.Error(t, got)
			assert.Equal(t, tt.wantMsg, got.Error())
		})
	}
}
