package core

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attrDef(name string) *schema.AttributeDefinition {
	return &schema.AttributeDefinition{Name: name}
}

func TestTransformPassthrough(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{"string", "hello"},
		{"int", 42},
		{"bool", true},
		{"nil", nil},
		{"map", map[string]interface{}{"a": 1}},
		{"slice", []interface{}{1, 2, 3}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformPassthrough(tc.input, attrDef("test"))
			require.NoError(t, err)
			assert.Equal(t, tc.input, result)
		})
	}
}

func TestTransformBase64Encode(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{"simple", "hello", base64.StdEncoding.EncodeToString([]byte("hello")), false},
		{"empty", "", base64.StdEncoding.EncodeToString([]byte("")), false},
		{"unicode", "日本語", base64.StdEncoding.EncodeToString([]byte("日本語")), false},
		{"not string", 42, "", true},
		{"nil value", nil, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformBase64Encode(tc.input, attrDef("test"))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformBase64Decode(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{"simple", base64.StdEncoding.EncodeToString([]byte("hello")), "hello", false},
		{"empty", base64.StdEncoding.EncodeToString([]byte("")), "", false},
		{"invalid base64", "not-valid-base64!!", "", true},
		{"not string", 42, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformBase64Decode(tc.input, attrDef("test"))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformJSONEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{"string", "hello", `"hello"`, false},
		{"number", 42, "42", false},
		{"bool", true, "true", false},
		{"map", map[string]interface{}{"key": "val"}, `{"key":"val"}`, false},
		{"slice", []interface{}{1, 2, 3}, "[1,2,3]", false},
		{"nil", nil, "null", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformJSONEncode(tc.input, attrDef("test"))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformJSONDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{"string", `"hello"`, "hello", false},
		{"number", "42", float64(42), false},
		{"bool", "true", true, false},
		{"object", `{"key":"val"}`, map[string]interface{}{"key": "val"}, false},
		{"invalid json", "{invalid", nil, true},
		{"not string", 42, nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformJSONDecode(tc.input, attrDef("test"))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"float", 3.14, "3.14"},
		{"nil", nil, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformToString(tc.input, attrDef("test"))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestApplyTransform(t *testing.T) {
	tests := []struct {
		name      string
		transform string
		input     interface{}
		wantErr   bool
	}{
		{"empty name passthrough", "", "hello", false},
		{"explicit passthrough", "passthrough", "test", false},
		{"custom passthrough", "custom", "data", false},
		{"base64_encode", "base64_encode", "hello", false},
		{"json_encode", "json_encode", map[string]interface{}{"k": "v"}, false},
		{"to_string", "to_string", 42, false},
		{"jsonencode_raw", "jsonencode_raw", map[string]interface{}{"k": "v"}, false},
		{"unknown transform", "nonexistent", "x", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ApplyTransform(tc.transform, tc.input, attrDef("test"))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestRegisterTransform(t *testing.T) {
	RegisterTransform("double", func(value interface{}, _ *schema.AttributeDefinition) (interface{}, error) {
		n, ok := value.(int)
		if !ok {
			return nil, nil
		}
		return n * 2, nil
	})

	result, err := ApplyTransform("double", 5, attrDef("test"))
	require.NoError(t, err)
	assert.Equal(t, 10, result)

	// Clean up
	delete(StandardTransforms, "double")
}

func TestBase64RoundTrip(t *testing.T) {
	original := "The quick brown fox jumps over the lazy dog"
	attr := attrDef("test")

	encoded, err := transformBase64Encode(original, attr)
	require.NoError(t, err)

	decoded, err := transformBase64Decode(encoded, attr)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestJSONRoundTrip(t *testing.T) {
	original := map[string]interface{}{
		"name":   "test",
		"count":  float64(42),
		"active": true,
	}
	attr := attrDef("test")

	encoded, err := transformJSONEncode(original, attr)
	require.NoError(t, err)

	decoded, err := transformJSONDecode(encoded, attr)
	require.NoError(t, err)

	decodedMap, ok := decoded.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, original["name"], decodedMap["name"])
	assert.Equal(t, original["count"], decodedMap["count"])
	assert.Equal(t, original["active"], decodedMap["active"])
}

func TestTransformJSONEncodeRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected RawHCLValue
		wantErr  bool
	}{
		{
			name:     "simple map",
			input:    map[string]interface{}{"key": "val"},
			expected: RawHCLValue("jsonencode({\n  \"key\": \"val\"\n})"),
		},
		{
			name:     "nested map",
			input:    map[string]interface{}{"a": map[string]interface{}{"b": "c"}},
			expected: RawHCLValue("jsonencode({\n  \"a\": {\n    \"b\": \"c\"\n  }\n})"),
		},
		{
			name:     "slice",
			input:    []interface{}{1, 2, 3},
			expected: RawHCLValue("jsonencode([\n  1,\n  2,\n  3\n])"),
		},
		{
			name:     "string value",
			input:    "hello",
			expected: RawHCLValue(`jsonencode("hello")`),
		},
		{
			name:     "number",
			input:    42,
			expected: RawHCLValue(`jsonencode(42)`),
		},
		{
			name:     "bool",
			input:    true,
			expected: RawHCLValue(`jsonencode(true)`),
		},
		{
			name:     "nil",
			input:    nil,
			expected: RawHCLValue(`jsonencode(null)`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformJSONEncodeRaw(tc.input, attrDef("test"))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformBase64EncodeBinaryData(t *testing.T) {
	jsonData := `{"key":"value with \"quotes\" and\nnewlines"}`
	attr := attrDef("graph_data")

	result, err := transformBase64Encode(jsonData, attr)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result.(string))
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(decoded, &parsed))
	assert.Equal(t, "value with \"quotes\" and\nnewlines", parsed["key"])
}

func TestTransformValueMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		valueMap map[string]string
		dflt     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "mapped value",
			input:    "bool",
			valueMap: map[string]string{"bool": "boolean", "int": "number"},
			expected: "boolean",
		},
		{
			name:     "unmapped value preserved",
			input:    "string",
			valueMap: map[string]string{"bool": "boolean"},
			expected: "string",
		},
		{
			name:     "unmapped value with default",
			input:    "secret",
			valueMap: map[string]string{"bool": "boolean"},
			dflt:     "string",
			expected: "string",
		},
		{
			name:     "non-string input",
			input:    42,
			valueMap: map[string]string{},
			wantErr:  true,
		},
		{
			name:     "nil input",
			input:    nil,
			valueMap: map[string]string{},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attr := &schema.AttributeDefinition{
				Name:            "test",
				ValueMap:        tc.valueMap,
				ValueMapDefault: tc.dflt,
			}
			result, err := transformValueMap(tc.input, attr)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildMaskedSecretVarName(t *testing.T) {
	tests := []struct {
		name         string
		parts        []schema.VariableNamePart
		resourceName string
		attrKey      string
		expected     string
	}{
		{
			name: "resource_name and attribute_key",
			parts: []schema.VariableNamePart{
				{Source: "resource_name"},
				{Source: "attribute_key"},
			},
			resourceName: "my_connector",
			attrKey:      "api_key",
			expected:     "my_connector_api_key",
		},
		{
			name: "with literal prefix",
			parts: []schema.VariableNamePart{
				{Source: "literal", Value: "dv"},
				{Source: "resource_name"},
				{Source: "attribute_key"},
			},
			resourceName: "conn",
			attrKey:      "secret",
			expected:     "dv_conn_secret",
		},
		{
			name: "resource_name only",
			parts: []schema.VariableNamePart{
				{Source: "resource_name"},
			},
			resourceName: "myres",
			expected:     "myres",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildMaskedSecretVarName(tc.parts, tc.resourceName, tc.attrKey)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestReplaceMaskedSecrets(t *testing.T) {
	config := &schema.MaskedSecretConfig{
		Sentinel: "******",
		VariableNameParts: []schema.VariableNamePart{
			{Source: "resource_name"},
			{Source: "attribute_key"},
		},
	}

	t.Run("string sentinel replaced", func(t *testing.T) {
		result, vars := ReplaceMaskedSecrets("******", config, "my_conn")
		assert.Equal(t, "${var.my_conn_}", result)
		require.Len(t, vars, 1)
		assert.Equal(t, "my_conn_", vars[0].Name)
		assert.True(t, vars[0].Sensitive)
	})

	t.Run("string non-sentinel preserved", func(t *testing.T) {
		result, vars := ReplaceMaskedSecrets("real_value", config, "my_conn")
		assert.Equal(t, "real_value", result)
		assert.Empty(t, vars)
	})

	t.Run("map with mixed values", func(t *testing.T) {
		input := map[string]interface{}{
			"api_key":  "******",
			"base_url": "https://example.com",
			"token":    "******",
		}

		result, vars := ReplaceMaskedSecrets(input, config, "conn1")
		m, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "${var.conn1_api_key}", m["api_key"])
		assert.Equal(t, "https://example.com", m["base_url"])
		assert.Equal(t, "${var.conn1_token}", m["token"])
		assert.Len(t, vars, 2)
	})

	t.Run("nil config returns value unchanged", func(t *testing.T) {
		result, vars := ReplaceMaskedSecrets("******", nil, "x")
		assert.Equal(t, "******", result)
		assert.Nil(t, vars)
	})

	t.Run("non-string non-map returns unchanged", func(t *testing.T) {
		result, vars := ReplaceMaskedSecrets(42, config, "x")
		assert.Equal(t, 42, result)
		assert.Nil(t, vars)
	})
}