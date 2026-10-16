package variables

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
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

// secretVariableRuleDef mirrors the live variables.eligible_attributes rule
// from definitions/pingone/davinci/variable.yaml: the value attribute is
// eligible, and secretness/sensitivity are conditional on data_type == secret.
func secretVariableRuleDef() *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_variable",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "value", TerraformName: "value", Type: "type_discriminated_block", VariableEligible: true},
		},
		Variables: schema.VariableDefinition{
			EligibleAttributes: []schema.VariableRule{
				{
					AttributePath:  "value",
					VariablePrefix: "davinci_variable_",
					IsSecret:       false,
					SensitiveWhenAll: []schema.DefaultCondition{
						{AttributeEquals: &schema.AttributeValueCondition{Name: "data_type", Value: "secret"}},
					},
					IsSecretWhenAll: []schema.DefaultCondition{
						{AttributeEquals: &schema.AttributeValueCondition{Name: "data_type", Value: "secret"}},
					},
					Description: "Value for DaVinci variable {name}",
				},
			},
		},
	}
}

func TestExtract_ConditionalSecretRule(t *testing.T) {
	reg := testRegistry(secretVariableRuleDef())
	ext := NewVariableExtractor(reg)

	// Secret variable: processor-emitted placeholder inside the tdb map.
	data := map[string]interface{}{
		"data_type": "secret",
		"value":     map[string]interface{}{"secret_string": "REPLACED_BY_VARIABLE_EXTRACTOR"},
	}
	vars, err := ext.Extract("pingone_davinci_variable", data, "secret-a")
	require.NoError(t, err)
	require.Len(t, vars, 1)

	v := vars[0]
	assert.True(t, v.IsSecret, "data_type=secret must mark the variable secret")
	assert.True(t, v.Sensitive, "secret variables must be sensitive")
	assert.Equal(t, "davinci_variable_secret_a_value", v.VariableName)
	assert.Equal(t, "Value for DaVinci variable secret-a", v.Description)
	assert.Equal(t, "REPLACED_BY_VARIABLE_EXTRACTOR", v.CurrentValue)
}

func TestExtract_ConditionalSecretRule_NotSecretForStringType(t *testing.T) {
	reg := testRegistry(secretVariableRuleDef())
	ext := NewVariableExtractor(reg)

	// Plain string variable: same rule, but data_type != secret → the
	// conditional flags must not fire and the exported value keeps its
	// tfvars value.
	data := map[string]interface{}{
		"data_type": "string",
		"value":     map[string]interface{}{"string": "abcd"},
	}
	vars, err := ext.Extract("pingone_davinci_variable", data, "string-a")
	require.NoError(t, err)
	require.Len(t, vars, 1)

	v := vars[0]
	assert.False(t, v.IsSecret, "non-secret data_type must not be marked secret")
	assert.False(t, v.Sensitive, "non-secret data_type must not be sensitive")
}

func TestExtract_ConditionalRule_NoConditionsNeverFires(t *testing.T) {
	// A rule with no conditional flags must stay non-secret/non-sensitive
	// even when other rules in the same definition carry them — guards the
	// vacuous-true pitfall of evaluating an empty condition list.
	def := secretVariableRuleDef()
	def.Attributes = append(def.Attributes,
		schema.AttributeDefinition{Name: "name", TerraformName: "name", Type: "string", VariableEligible: true},
	)
	def.Variables.EligibleAttributes = append(def.Variables.EligibleAttributes,
		schema.VariableRule{AttributePath: "name", VariablePrefix: "davinci_variable_"},
	)

	reg := testRegistry(def)
	ext := NewVariableExtractor(reg)

	data := map[string]interface{}{
		"data_type": "secret",
		"value":     map[string]interface{}{"secret_string": "REPLACED_BY_VARIABLE_EXTRACTOR"},
		"name":      "secret-a",
	}
	vars, err := ext.Extract("pingone_davinci_variable", data, "secret-a")
	require.NoError(t, err)
	require.Len(t, vars, 2)

	byPath := make(map[string]*core.ExtractedVariable)
	for _, v := range vars {
		byPath[v.AttributePath] = v
	}
	assert.True(t, byPath["value"].IsSecret, "value rule with matching conditions fires")
	assert.False(t, byPath["name"].IsSecret, "rule without conditional flags must never fire")
	assert.False(t, byPath["name"].Sensitive)
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

func TestExpandDescription(t *testing.T) {
	assert.Equal(t, "Value for my_flow", expandDescription("Value for {name}", "my_flow"))
	assert.Equal(t, "no placeholders", expandDescription("no placeholders", "x"))
}

func TestExtractNoRuleFallback(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
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
