package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBareRefPlaceholder(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"variable token", "var.davinci_form_login_form", `"@@TFREF@@var.davinci_form_login_form@@TFREF@@"`},
		{"resource traversal", "pingone_davinci_flow.pingcli__My-0020-Flow.id", `"@@TFREF@@pingone_davinci_flow.pingcli__My-0020-Flow.id@@TFREF@@"`},
		{"empty token", "", `"@@TFREF@@@@TFREF@@"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BareRefPlaceholder(tt.token))
		})
	}
}

// TestBareRefPlaceholder_JSONSafe verifies the placeholder is a valid JSON
// string that round-trips — the property the substitution mechanism depends
// on to keep intermediate blobs parseable by later rule applications.
func TestBareRefPlaceholder_JSONSafe(t *testing.T) {
	tokens := []string{
		"var.davinci_form_login_form",
		"pingone_davinci_flow.pingcli__My-0020-Flow.id",
		"var.external_id_sign_on_node",
	}
	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			ph := BareRefPlaceholder(token)

			var decoded string
			require.NoError(t, json.Unmarshal([]byte(ph), &decoded))
			assert.Equal(t, bareRefSentinel+token+bareRefSentinel, decoded)
		})
	}
}

func TestMaterializeBareRefs(t *testing.T) {
	tests := []struct {
		name string
		blob string
		want string
	}{
		{"no placeholder", `jsonencode({"a": "b"})`, `jsonencode({"a": "b"})`},
		{"single placeholder", `jsonencode({"value": "@@TFREF@@var.x@@TFREF@@"})`, `jsonencode({"value": var.x})`},
		{"multiple placeholders", `jsonencode({"a": "@@TFREF@@var.x@@TFREF@@", "b": "@@TFREF@@type.label.field@@TFREF@@"})`, `jsonencode({"a": var.x, "b": type.label.field})`},
		{"placeholder inside longer string unchanged", `jsonencode({"note": "prefix @@TFREF@@var.x@@TFREF@@ suffix"})`, `jsonencode({"note": "prefix @@TFREF@@var.x@@TFREF@@ suffix"})`},
		{"bare sentinel without quotes unchanged", `jsonencode({"a": @@TFREF@@var.x@@TFREF@@})`, `jsonencode({"a": @@TFREF@@var.x@@TFREF@@})`},
		{"unterminated placeholder unchanged", `jsonencode({"a": "@@TFREF@@var.x})`, `jsonencode({"a": "@@TFREF@@var.x})`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MaterializeBareRefs(tt.blob))
		})
	}
}

// TestMaterializeBareRefs_MarshalRoundTrip verifies the placeholder survives
// a full JSON marshal round-trip (mirroring what json.MarshalIndent does on
// the connector complex-property path in Task 2) and that materialization
// then yields the exact expected bare-token rendering. The value stored in
// the Go map is the *unquoted* sentinel core — json.MarshalIndent supplies
// the surrounding quotes, and BareRefPlaceholder's quotes would be
// double-encoded by marshaling.
func TestMaterializeBareRefs_MarshalRoundTrip(t *testing.T) {
	props := map[string]interface{}{
		"clientId": map[string]interface{}{"value": BareRefSentinelCore("var.davinci_connection_Test_clientId")},
		"issuer":   "https://example.com",
	}

	marshaled, err := json.MarshalIndent(props, "", "  ")
	require.NoError(t, err)

	blob := MaterializeBareRefs(string(marshaled))
	assert.NotContains(t, blob, "@@TFREF@@")
	assert.NotContains(t, blob, `"${var.`)
	assert.Equal(t, "{\n  \"clientId\": {\n    \"value\": var.davinci_connection_Test_clientId\n  },\n  \"issuer\": \"https://example.com\"\n}", blob)
}

// TestMaterializeBareRefsInAttributes walks nested attribute shapes (mirroring
// walkAndProcessPath traversal) and materializes only RawHCLValue leaves.
func TestMaterializeBareRefsInAttributes(t *testing.T) {
	attrs := map[string]interface{}{
		"plain": "not-a-blob",
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"form": {"value": "@@TFREF@@var.x@@TFREF@@"}})`),
						},
					},
				},
			},
		},
		"list": []interface{}{
			RawHCLValue(`jsonencode({"a": "@@TFREF@@var.y@@TFREF@@"})`),
			"string-element",
		},
	}

	materializeBareRefsInAttributes(attrs)

	props := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
	assert.Contains(t, string(props), `"value": var.x`)
	assert.Equal(t, "not-a-blob", attrs["plain"])
	assert.Equal(t, "string-element", attrs["list"].([]interface{})[1])

	listBlob := attrs["list"].([]interface{})[0].(RawHCLValue)
	assert.Contains(t, string(listBlob), `"a": var.y`)
}

// TestMaterializeBareRefs_AdjacentPlaceholders verifies two neighbouring
// placeholders materialize independently without splicing across each other.
// Tokens come from Terraform traversals and sanitized names ([A-Za-z0-9_.-]),
// so '@' never occurs inside a real token — the materializer relies on that.
func TestMaterializeBareRefs_AdjacentPlaceholders(t *testing.T) {
	blob := `"@@TFREF@@var.a@@TFREF@@","@@TFREF@@var.b@@TFREF@@"`
	assert.Equal(t, `var.a,var.b`, MaterializeBareRefs(blob))
	assert.False(t, strings.Contains(MaterializeBareRefs(blob), "@@TFREF@@"))
}
