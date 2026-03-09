package variables

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRegistry(def *schema.ResourceDefinition) *schema.Registry {
	r := schema.NewRegistry()
	_ = r.Register(def)
	return r
}

func TestExtractBasic(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_variable",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "name", TerraformName: "name", Type: "string", VariableEligible: false},
			{Name: "value", TerraformName: "value", Type: "string", VariableEligible: true},
		},
		Variables: schema.VariableDefinition{
			EligibleAttributes: []schema.VariableRule{
				{
					AttributePath:  "value",
					VariablePrefix: "davinci_variable_",
					Description:    "Value for {name}",
				},
			},
		},
	}

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	data := map[string]interface{}{
		"name":  "company_name",
		"value": "Acme Corp",
	}

	vars, err := ext.Extract("pingone_davinci_variable", data, "company_name")
	require.NoError(t, err)
	require.Len(t, vars, 1)

	v := vars[0]
	assert.Equal(t, "pingone_davinci_variable", v.ResourceType)
	assert.Equal(t, "company_name", v.ResourceName)
	assert.Equal(t, "value", v.AttributePath)
	assert.Equal(t, "Acme Corp", v.CurrentValue)
	assert.Equal(t, "davinci_variable_company_name_value", v.VariableName)
	assert.Equal(t, "string", v.VariableType)
	assert.Equal(t, "Value for company_name", v.Description)
	assert.False(t, v.IsSecret)
}

func TestExtractSkipsNonEligible(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_variable",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "name", TerraformName: "name", Type: "string", VariableEligible: false},
			{Name: "description", TerraformName: "description", Type: "string", VariableEligible: false},
		},
	}

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	data := map[string]interface{}{"name": "x", "description": "y"}
	vars, err := ext.Extract("pingone_davinci_variable", data, "x")
	require.NoError(t, err)
	assert.Empty(t, vars)
}

func TestExtractSkipsNilValue(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_variable",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "value", TerraformName: "value", Type: "string", VariableEligible: true},
		},
	}

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	data := map[string]interface{}{"value": nil}
	vars, err := ext.Extract("pingone_davinci_variable", data, "x")
	require.NoError(t, err)
	assert.Empty(t, vars)
}

func TestExtractSensitiveAttribute(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_connection",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "api_key", TerraformName: "api_key", Type: "string", VariableEligible: true, Sensitive: true},
		},
		Variables: schema.VariableDefinition{
			EligibleAttributes: []schema.VariableRule{
				{AttributePath: "api_key", IsSecret: true},
			},
		},
	}

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	data := map[string]interface{}{"api_key": "secret123"}
	vars, err := ext.Extract("pingone_davinci_connection", data, "http_conn")
	require.NoError(t, err)
	require.Len(t, vars, 1)

	assert.True(t, vars[0].Sensitive)
	assert.True(t, vars[0].IsSecret)
}

func TestExtractUnknownResourceType(t *testing.T) {
	reg := schema.NewRegistry()
	ext := NewVariableExtractor(reg)

	_, err := ext.Extract("nonexistent_type", nil, "x")
	require.Error(t, err)
}

func TestInferTerraformType(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"string", "string"},
		{"number", "number"},
		{"bool", "bool"},
		{"object", "any"},
		{"map", "any"},
		{"list", "list(any)"},
		{"unknown", "string"},
		{"", "string"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.out, inferTerraformType(tt.in), "input: %s", tt.in)
	}
}

func TestSanitizeVariableName(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"simple", "simple"},
		{"with-dashes", "with_dashes"},
		{"CamelCase", "camelcase"},
		{"dots.and.slashes/here", "dots_and_slashes_here"},
		{"a b c", "a_b_c"},
		{"already_underscored", "already_underscored"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.out, sanitizeVariableName(tt.in), "input: %s", tt.in)
	}
}

func TestExpandDescription(t *testing.T) {
	assert.Equal(t, "Value for my_flow", expandDescription("Value for {name}", "my_flow"))
	assert.Equal(t, "no placeholders", expandDescription("no placeholders", "x"))
}

func TestExtractNoRuleFallback(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_variable",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "value", TerraformName: "value", Type: "string", VariableEligible: true},
		},
	}

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	data := map[string]interface{}{"value": "test"}
	vars, err := ext.Extract("pingone_davinci_variable", data, "my_var")
	require.NoError(t, err)
	require.Len(t, vars, 1)

	assert.Equal(t, "pingone_davinci_variable_my_var_value", vars[0].VariableName)
}

func TestExtractMultipleEligible(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_connection",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "base_url", TerraformName: "base_url", Type: "string", VariableEligible: true},
			{Name: "api_key", TerraformName: "api_key", Type: "string", VariableEligible: true, Sensitive: true},
			{Name: "name", TerraformName: "name", Type: "string", VariableEligible: false},
		},
	}

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	data := map[string]interface{}{
		"base_url": "https://example.com",
		"api_key":  "secret",
		"name":     "My Conn",
	}

	vars, err := ext.Extract("pingone_davinci_connection", data, "http")
	require.NoError(t, err)
	assert.Len(t, vars, 2)
}

func TestExtractWithTerraformName(t *testing.T) {
	// Simulates real YAML definitions where Name is PascalCase and
	// TerraformName is the lowercase data-map key used by the processor.
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_variable",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "Name", TerraformName: "name", Type: "string", VariableEligible: false},
			{Name: "Value", TerraformName: "value", Type: "string", VariableEligible: true},
		},
		Variables: schema.VariableDefinition{
			EligibleAttributes: []schema.VariableRule{
				{
					AttributePath:  "value",
					VariablePrefix: "davinci_variable_",
					Description:    "Value for {name}",
				},
			},
		},
	}

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	// Data map keyed by terraform_name (as the processor produces).
	data := map[string]interface{}{
		"name":  "company_name",
		"value": "Acme Corp",
	}

	vars, err := ext.Extract("pingone_davinci_variable", data, "company_name")
	require.NoError(t, err)
	require.Len(t, vars, 1)

	v := vars[0]
	assert.Equal(t, "value", v.AttributePath)
	assert.Equal(t, "Acme Corp", v.CurrentValue)
	assert.Equal(t, "davinci_variable_company_name_value", v.VariableName)
	assert.Equal(t, "Value for company_name", v.Description)
}
