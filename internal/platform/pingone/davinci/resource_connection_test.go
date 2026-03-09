package davinci

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAPIData simulates a connector instance API response.
type mockAPIData struct {
	Id          string
	Environment struct{ Id string }
	Connector   struct{ Id string }
	Name        string
	Properties  map[string]interface{}
}

func TestHandleConnectorProperties(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		apiData  interface{}
		wantNil  bool
		contains []string
	}{
		{
			name:    "nil value returns nil",
			value:   nil,
			apiData: &mockAPIData{Name: "test"},
			wantNil: true,
		},
		{
			name:    "empty map returns nil",
			value:   map[string]interface{}{},
			apiData: &mockAPIData{Name: "test"},
			wantNil: true,
		},
		{
			name: "string property formatted correctly",
			value: map[string]interface{}{
				"clientId": map[string]interface{}{
					"type":  "string",
					"value": "my-client-id",
				},
			},
			apiData: &mockAPIData{Name: "Test Connector"},
			contains: []string{
				`"clientId"`,
				`"type": "string"`,
				`"value": "my-client-id"`,
				"jsonencode(",
			},
		},
		{
			name: "masked secret produces variable reference",
			value: map[string]interface{}{
				"clientSecret": map[string]interface{}{
					"type":  "string",
					"value": "******",
				},
			},
			apiData: &mockAPIData{Name: "PingOne Protect"},
			contains: []string{
				`"clientSecret"`,
				`${var.davinci_connection_PingOne-0020-Protect_clientSecret}`,
			},
		},
		{
			name: "bool property formatted correctly",
			value: map[string]interface{}{
				"enabled": map[string]interface{}{
					"type":  "boolean",
					"value": true,
				},
			},
			apiData: &mockAPIData{Name: "test"},
			contains: []string{
				`"value": true`,
			},
		},
		{
			name: "number property formatted as integer",
			value: map[string]interface{}{
				"timeout": map[string]interface{}{
					"type":  "number",
					"value": float64(30),
				},
			},
			apiData: &mockAPIData{Name: "test"},
			contains: []string{
				`"value": 30`,
			},
		},
		{
			name: "empty type omitted",
			value: map[string]interface{}{
				"noType": map[string]interface{}{
					"type":  "",
					"value": "val",
				},
			},
			apiData: &mockAPIData{Name: "test"},
			contains: []string{
				`"value": "val"`,
			},
		},
		{
			name: "null value rendered",
			value: map[string]interface{}{
				"empty": map[string]interface{}{
					"type":  "string",
					"value": nil,
				},
			},
			apiData: &mockAPIData{Name: "test"},
			contains: []string{
				`"value": null`,
			},
		},
		{
			name: "multiple properties sorted alphabetically",
			value: map[string]interface{}{
				"zebra": map[string]interface{}{
					"type":  "string",
					"value": "z",
				},
				"alpha": map[string]interface{}{
					"type":  "string",
					"value": "a",
				},
			},
			apiData: &mockAPIData{Name: "test"},
			contains: []string{
				`"alpha"`,
				`"zebra"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleConnectorProperties(tt.value, tt.apiData, &schema.AttributeDefinition{}, &schema.ResourceDefinition{
				Metadata: schema.ResourceMetadata{
					ResourceType: "pingone_davinci_connector_instance",
				},
			})
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			// The handler now returns TransformResultWithVariables wrapping the RawHCLValue.
			wrapped, ok := result.(core.TransformResultWithVariables)
			require.True(t, ok, "expected TransformResultWithVariables, got %T", result)

			rawHCL, ok := wrapped.Value.(core.RawHCLValue)
			require.True(t, ok, "expected RawHCLValue inside wrapper, got %T", wrapped.Value)

			for _, s := range tt.contains {
				assert.Contains(t, string(rawHCL), s)
			}
		})
	}
}

func TestHandleConnectorPropertiesExtractsVariables(t *testing.T) {
	apiData := struct{ Name string }{Name: "PingOne"}
	props := map[string]interface{}{
		"clientId": map[string]interface{}{
			"type":  "string",
			"value": "abc-123",
		},
		"clientSecret": map[string]interface{}{
			"type":  "string",
			"value": "******",
		},
		"region": map[string]interface{}{
			"type":  "string",
			"value": "NA",
		},
		"createdDate": map[string]interface{}{
			"type":  "string",
			"value": "2024-01-01",
		},
	}

	result, err := handleConnectorProperties(props, apiData, &schema.AttributeDefinition{}, &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			ResourceType: "pingone_davinci_connector_instance",
		},
	})
	require.NoError(t, err)

	wrapped, ok := result.(core.TransformResultWithVariables)
	require.True(t, ok)

	// All properties become variables (no exclusion list).
	require.Len(t, wrapped.Variables, 4)

	// Build a lookup by variable name.
	varMap := make(map[string]core.ExtractedVariable)
	for _, v := range wrapped.Variables {
		varMap[v.VariableName] = v
	}

	// clientSecret — masked, should be marked secret.
	cs, ok := varMap["davinci_connection_PingOne_clientSecret"]
	require.True(t, ok, "expected clientSecret variable")
	assert.True(t, cs.IsSecret)
	assert.True(t, cs.Sensitive)
	assert.Nil(t, cs.CurrentValue, "secrets should not expose values")

	// clientId — not secret, should carry current value.
	ci, ok := varMap["davinci_connection_PingOne_clientId"]
	require.True(t, ok, "expected clientId variable")
	assert.False(t, ci.IsSecret)
	assert.Equal(t, "abc-123", ci.CurrentValue)

	// region — not secret.
	rg, ok := varMap["davinci_connection_PingOne_region"]
	require.True(t, ok, "expected region variable")
	assert.False(t, rg.IsSecret)
	assert.Equal(t, "NA", rg.CurrentValue)

	// createdDate — included, not secret.
	cd, ok := varMap["davinci_connection_PingOne_createdDate"]
	require.True(t, ok, "expected createdDate variable")
	assert.False(t, cd.IsSecret)
	assert.Equal(t, "2024-01-01", cd.CurrentValue)
}

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		want    string
	}{
		{
			name: "struct with Name field",
			data: &mockAPIData{Name: "PingOne Protect"},
			want: "pingcli__PingOne-0020-Protect",
		},
		{
			name: "nil returns empty",
			data: nil,
			want: "",
		},
		{
			name: "string returns empty",
			data: "not a struct",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceName(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConnectorVariableName(t *testing.T) {
	tests := []struct {
		resource string
		property string
		want     string
	}{
		{
			resource: "pingcli__PingOne-0020-Protect",
			property: "clientSecret",
			want:     "davinci_connection_PingOne-0020-Protect_clientSecret",
		},
		{
			resource: "pingcli__simple",
			property: "apiKey",
			want:     "davinci_connection_simple_apiKey",
		},
		{
			resource: "davinci__myConn",
			property: "token",
			want:     "davinci_connection_myConn_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.resource+"_"+tt.property, func(t *testing.T) {
			got := connectorVariableName("davinci_connection_", tt.resource, tt.property)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsComplexProperty(t *testing.T) {
	tests := []struct {
		name string
		prop map[string]interface{}
		want bool
	}{
		{
			name: "simple property without nested properties",
			prop: map[string]interface{}{
				"type":  "string",
				"value": "my-client-id",
			},
			want: false,
		},
		{
			name: "complex property with nested properties",
			prop: map[string]interface{}{
				"type":        "array",
				"displayName": "Custom Parameters",
				"properties": map[string]interface{}{
					"providerName": map[string]interface{}{"type": "string", "value": "Login with Apple"},
				},
			},
			want: true,
		},
		{
			name: "empty map",
			prop: map[string]interface{}{},
			want: false,
		},
		{
			name: "shape2 value map with nested properties",
			prop: map[string]interface{}{
				"type": "object",
				"value": map[string]interface{}{
					"properties": map[string]interface{}{
						"providerName": map[string]interface{}{"type": "string", "value": "Login with Apple"},
					},
				},
			},
			want: true,
		},
		{
			name: "shape2 value JSON string with nested properties",
			prop: map[string]interface{}{
				"type":  "object",
				"value": `{"properties":{"providerName":{"type":"string","value":"Login with Apple"}}}`,
			},
			want: true,
		},
		{
			name: "shape2 value map without properties key",
			prop: map[string]interface{}{
				"type": "object",
				"value": map[string]interface{}{
					"someOther": "data",
				},
			},
			want: false,
		},
		{
			name: "shape2 value JSON string without properties key",
			prop: map[string]interface{}{
				"type":  "object",
				"value": `{"someOther":"data"}`,
			},
			want: false,
		},
		{
			name: "shape2 value is non-JSON string",
			prop: map[string]interface{}{
				"type":  "string",
				"value": "just-a-string",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isComplexProperty(tt.prop)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsNestedSecret(t *testing.T) {
	tests := []struct {
		name     string
		propName string
		propMap  map[string]interface{}
		want     bool
	}{
		{
			name:     "secure flag true",
			propName: "someField",
			propMap:  map[string]interface{}{"secure": true, "value": "plain"},
			want:     true,
		},
		{
			name:     "masked value",
			propName: "someField",
			propMap:  map[string]interface{}{"value": "******"},
			want:     true,
		},
		{
			name:     "name alone does not trigger secret",
			propName: "clientSecret",
			propMap:  map[string]interface{}{"value": "plain"},
			want:     false,
		},
		{
			name:     "password name alone does not trigger secret",
			propName: "userPassword",
			propMap:  map[string]interface{}{"value": "plain"},
			want:     false,
		},
		{
			name:     "token name alone does not trigger secret",
			propName: "bearerToken",
			propMap:  map[string]interface{}{"value": "plain"},
			want:     false,
		},
		{
			name:     "privatekey name alone does not trigger secret",
			propName: "myPrivateKey",
			propMap:  map[string]interface{}{"value": "plain"},
			want:     false,
		},
		{
			name:     "apikey name alone does not trigger secret",
			propName: "apiKey",
			propMap:  map[string]interface{}{"value": "plain"},
			want:     false,
		},
		{
			name:     "clientId is not a secret",
			propName: "clientId",
			propMap:  map[string]interface{}{"value": "abc-123"},
			want:     false,
		},
		{
			name:     "providerName is not a secret",
			propName: "providerName",
			propMap:  map[string]interface{}{"value": "Login with Apple"},
			want:     false,
		},
		{
			name:     "no indicators returns false",
			propName: "displayName",
			propMap:  map[string]interface{}{"value": "My Connector"},
			want:     false,
		},
		{
			name:     "secure flag false",
			propName: "someField",
			propMap:  map[string]interface{}{"secure": false, "value": "plain"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNestedSecret(tt.propName, tt.propMap)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHandleConnectorPropertiesComplex(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		apiData     interface{}
		contains    []string
		notContains []string
	}{
		{
			name: "customAuth complex property",
			value: map[string]interface{}{
				"customAuth": map[string]interface{}{
					"type":        "array",
					"displayName": "Custom Parameters",
					"properties": map[string]interface{}{
						"providerName": map[string]interface{}{
							"type":                 "string",
							"displayName":          "Provider Name",
							"preferredControlType": "textField",
							"value":                "Login with Apple",
						},
						"clientSecret": map[string]interface{}{
							"type":                 "string",
							"displayName":          "Private Key",
							"secure":               true,
							"preferredControlType": "textArea",
							"value":                "******",
						},
					},
				},
			},
			apiData: &mockAPIData{Name: "Apple Login"},
			contains: []string{
				`"customAuth"`,
				`"type": "json"`,
				"jsonencode(",
				`"properties"`,
				`"providerName"`,
				`"Login with Apple"`,
				`${var.davinci_connection_Apple-0020-Login_customAuth_clientSecret}`,
				`"secure": true`,
			},
			notContains: []string{
				`"displayName"`,
				`"preferredControlType"`,
			},
		},
		{
			name: "mixed simple and complex properties",
			value: map[string]interface{}{
				"clientId": map[string]interface{}{
					"type":  "string",
					"value": "my-client-id",
				},
				"customAuth": map[string]interface{}{
					"type": "array",
					"properties": map[string]interface{}{
						"providerName": map[string]interface{}{
							"type":  "string",
							"value": "My Provider",
						},
					},
				},
			},
			apiData: &mockAPIData{Name: "Mixed"},
			contains: []string{
				`"clientId"`,
				`"type": "string"`,
				`"value": "my-client-id"`,
				`"customAuth"`,
				`"type": "json"`,
				`"My Provider"`,
			},
		},
		{
			name: "complex property with empty nested properties",
			value: map[string]interface{}{
				"customAuth": map[string]interface{}{
					"type":        "array",
					"displayName": "Custom Parameters",
					"properties":  map[string]interface{}{},
				},
			},
			apiData: &mockAPIData{Name: "Empty"},
			contains: []string{
				`"customAuth"`,
				`"type": "json"`,
				"jsonencode(",
			},
			notContains: []string{
				`"displayName"`,
				`"Custom Parameters"`,
			},
		},
		{
			name: "complex property with invalid properties value",
			value: map[string]interface{}{
				"customAuth": map[string]interface{}{
					"type":        "array",
					"displayName": "Custom Parameters",
					"properties":  "invalid",
					"value":       "fallback-value",
				},
			},
			apiData: &mockAPIData{Name: "Invalid"},
			contains: []string{
				`"customAuth"`,
				`"type": "json"`,
				"jsonencode(",
				`fallback-value`,
			},
			notContains: []string{
				`"displayName"`,
				`"Custom Parameters"`,
			},
		},
		{
			name: "complex property with array value in nested property",
			value: map[string]interface{}{
				"customAuth": map[string]interface{}{
					"type": "array",
					"properties": map[string]interface{}{
						"customAttributes": map[string]interface{}{
							"type":  "array",
							"value": []interface{}{"attr1", "attr2"},
						},
					},
				},
			},
			apiData: &mockAPIData{Name: "ArrayTest"},
			contains: []string{
				`"customAuth"`,
				`"type": "json"`,
				`"customAttributes"`,
				`"attr1"`,
				`"attr2"`,
			},
		},
		{
			name: "complex property with no nested secrets",
			value: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"type": "array",
					"properties": map[string]interface{}{
						"providerName": map[string]interface{}{
							"type":  "string",
							"value": "Google",
						},
						"issuerUrl": map[string]interface{}{
							"type":  "string",
							"value": "https://accounts.google.com",
						},
					},
				},
			},
			apiData: &mockAPIData{Name: "Google"},
			contains: []string{
				`"type": "json"`,
				`"Google"`,
				`"https://accounts.google.com"`,
			},
			notContains: []string{
				`${var.`,
				`"secure"`,
			},
		},
		{
			name: "shape2 complex property with value as map",
			value: map[string]interface{}{
				"customAuth": map[string]interface{}{
					"type": "object",
					"value": map[string]interface{}{
						"properties": map[string]interface{}{
							"providerName": map[string]interface{}{
								"type":  "string",
								"value": "Login with Apple",
							},
							"clientSecret": map[string]interface{}{
								"type":   "string",
								"secure": true,
								"value":  "******",
							},
						},
					},
				},
			},
			apiData: &mockAPIData{Name: "Apple Login"},
			contains: []string{
				`"customAuth"`,
				`"type": "json"`,
				"jsonencode(",
				`"properties"`,
				`"providerName"`,
				`"Login with Apple"`,
				`${var.davinci_connection_Apple-0020-Login_customAuth_clientSecret}`,
				`"secure": true`,
			},
			notContains: []string{
				`"displayName"`,
			},
		},
		{
			name: "shape2 complex property with value as JSON string",
			value: map[string]interface{}{
				"customAuth": map[string]interface{}{
					"type":  "object",
					"value": `{"properties":{"providerName":{"type":"string","value":"Login with Apple"}}}`,
				},
			},
			apiData: &mockAPIData{Name: "Apple Login"},
			contains: []string{
				`"customAuth"`,
				`"type": "json"`,
				"jsonencode(",
				`"properties"`,
				`"providerName"`,
				`"Login with Apple"`,
			},
			notContains: []string{
				`"displayName"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleConnectorProperties(tt.value, tt.apiData, &schema.AttributeDefinition{}, &schema.ResourceDefinition{
				Metadata: schema.ResourceMetadata{
					ResourceType: "pingone_davinci_connector_instance",
				},
			})
			require.NoError(t, err)

			wrapped, ok := result.(core.TransformResultWithVariables)
			require.True(t, ok, "expected TransformResultWithVariables, got %T", result)

			rawHCL, ok := wrapped.Value.(core.RawHCLValue)
			require.True(t, ok, "expected RawHCLValue inside wrapper, got %T", wrapped.Value)

			for _, s := range tt.contains {
				assert.Contains(t, string(rawHCL), s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, string(rawHCL), s)
			}
		})
	}
}

func TestIsNestedSecretNonPatternNames(t *testing.T) {
	tests := []struct {
		name     string
		propName string
		want     bool
	}{
		{"credential is not matched", "credential", false},
		{"certificate is not matched", "certificate", false},
		{"scope is not matched", "scope", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNestedSecret(tt.propName, map[string]interface{}{"value": "some-value"})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHandleConnectorPropertiesComplexVariables(t *testing.T) {
	apiData := struct{ Name string }{Name: "Apple Login"}
	props := map[string]interface{}{
		"customAuth": map[string]interface{}{
			"type":        "array",
			"displayName": "Custom Parameters",
			"properties": map[string]interface{}{
				"providerName": map[string]interface{}{
					"type":                 "string",
					"displayName":          "Provider Name",
					"preferredControlType": "textField",
					"value":                "Login with Apple",
				},
				"clientSecret": map[string]interface{}{
					"type":                 "string",
					"displayName":          "Private Key",
					"secure":               true,
					"preferredControlType": "textArea",
					"value":                "******",
				},
			},
		},
	}

	result, err := handleConnectorProperties(props, apiData, &schema.AttributeDefinition{}, &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			ResourceType: "pingone_davinci_connector_instance",
		},
	})
	require.NoError(t, err)

	wrapped, ok := result.(core.TransformResultWithVariables)
	require.True(t, ok)

	// Two nested properties → two extracted variables.
	require.Len(t, wrapped.Variables, 2)

	varMap := make(map[string]core.ExtractedVariable)
	for _, v := range wrapped.Variables {
		varMap[v.VariableName] = v
	}

	// clientSecret — secret (secure: true + masked + name heuristic).
	cs, ok := varMap["davinci_connection_Apple-0020-Login_customAuth_clientSecret"]
	require.True(t, ok, "expected clientSecret variable")
	assert.True(t, cs.IsSecret)
	assert.True(t, cs.Sensitive)
	assert.Nil(t, cs.CurrentValue, "secrets should not expose values")
	assert.Equal(t, "properties.customAuth.clientSecret", cs.AttributePath)
	assert.Equal(t, "pingone_davinci_connector_instance", cs.ResourceType)

	// providerName — not a secret, should carry current value.
	pn, ok := varMap["davinci_connection_Apple-0020-Login_customAuth_providerName"]
	require.True(t, ok, "expected providerName variable")
	assert.False(t, pn.IsSecret)
	assert.False(t, pn.Sensitive)
	assert.Equal(t, "Login with Apple", pn.CurrentValue)
	assert.Equal(t, "properties.customAuth.providerName", pn.AttributePath)
}

