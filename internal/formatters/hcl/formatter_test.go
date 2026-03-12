package hcl_test

import (
	"strings"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	hclformatter "github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters/hcl"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalVariableDef returns a schema definition covering the expected variable attributes.
func minimalVariableDef() *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_variable",
			APIType:      "Variable",
			Name:         "DaVinci Variable",
			ShortName:    "variable",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage:  "github.com/pingidentity/pingone-go-client/davinci",
			SDKType:     "Variable",
			ListMethod:  "EnvironmentVariables.GetAll",
			IDField:     "ID",
			NameField:   "Name",
			LabelFields: []string{"name", "context"},
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", Computed: true},
			{
				Name: "EnvironmentID", TerraformName: "environment_id", Type: "string",
				Required: true, ReferencesType: "pingone_environment", ReferenceField: "id",
			},
			{Name: "Name", TerraformName: "name", Type: "string", Required: true},
			{Name: "Context", TerraformName: "context", Type: "string", Required: true},
			{Name: "DataType", TerraformName: "data_type", Type: "string", Required: true},
			{Name: "Mutable", TerraformName: "mutable", Type: "bool", Required: true},
			{Name: "DisplayName", TerraformName: "display_name", Type: "string"},
		},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: "{env_id}/{resource_id}",
		},
	}
}

func resourceData() *core.ResourceData {
	return &core.ResourceData{
		ResourceType: "pingone_davinci_variable",
		ID:           "var-abc123",
		Name:         "my_var__company",
		Attributes: map[string]interface{}{
			"id": "var-abc123",
			"environment_id": core.ResolvedReference{
				IsVariable:    true,
				VariableName:  "pingone_environment_id",
				OriginalValue: "env-xyz789",
			},
			"name":      "my_var",
			"context":   "company",
			"data_type": "string",
			"mutable":   true,
		},
	}
}

func TestFormatter_Format_WithDependencies(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	data := resourceData()

	output, err := f.Format(data, def, hclformatter.FormatOptions{SkipDependencies: false})
	require.NoError(t, err)

	assert.Contains(t, output, `resource "pingone_davinci_variable" "pingcli__my_var_company" {`)
	assert.Contains(t, output, "environment_id = var.pingone_environment_id")
	assert.Contains(t, output, `"my_var"`)
	assert.Contains(t, output, `"company"`)
	assert.Contains(t, output, `"string"`)
	assert.Contains(t, output, "true")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "}"), "should close with }")
}

func TestFormatter_Format_SkipDependencies(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	// SkipDependencies mode: attributes are raw UUIDs, not ResolvedReferences.
	data := &core.ResourceData{
		ResourceType: "pingone_davinci_variable",
		ID:           "var-abc123",
		Name:         "my_var__company",
		Attributes: map[string]interface{}{
			"id":             "var-abc123",
			"environment_id": "env-xyz789",
			"name":           "my_var",
			"context":        "company",
			"data_type":      "string",
			"mutable":        true,
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{
		SkipDependencies: true,
		EnvironmentID:    "env-xyz789",
	})
	require.NoError(t, err)

	assert.Contains(t, output, `resource "pingone_davinci_variable" "pingcli__my_var_company" {`)
	assert.Contains(t, output, `"env-xyz789"`)
	assert.NotContains(t, output, "var.pingone_environment_id")
}

func TestFormatter_Format_DisplayNameOptional(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()

	// Without display_name
	data := resourceData()
	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.NotContains(t, output, "display_name", "display_name should be absent when not set")

	// With display_name
	data.Attributes["display_name"] = "My Variable"
	output, err = f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "display_name")
	assert.Contains(t, output, `"My Variable"`)
}

func TestFormatter_FormatImportBlock(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	data := resourceData()

	output, err := f.FormatImportBlock(data, def, "env-xyz789")
	require.NoError(t, err)

	assert.Contains(t, output, "import {")
	assert.Contains(t, output, "to = pingone_davinci_variable.pingcli__my_var_company")
	assert.Contains(t, output, `"env-xyz789/var-abc123"`)
}

func TestFormatter_FormatList_Sorted(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()

	list := []*core.ResourceData{
		{ResourceType: "pingone_davinci_variable", ID: "id-z", Name: "zzz__company", Attributes: map[string]interface{}{"name": "zzz", "context": "company", "data_type": "string", "mutable": true}},
		{ResourceType: "pingone_davinci_variable", ID: "id-a", Name: "aaa__company", Attributes: map[string]interface{}{"name": "aaa", "context": "company", "data_type": "string", "mutable": false}},
	}

	output, err := f.FormatList(list, def, hclformatter.FormatOptions{SkipDependencies: true})
	require.NoError(t, err)

	idxA := strings.Index(output, "pingcli__aaa_company")
	idxZ := strings.Index(output, "pingcli__zzz_company")
	assert.Greater(t, idxZ, idxA, "resources should be sorted alphabetically by name")
}

func TestFormatter_Format_NilData(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()

	_, err := f.Format(nil, def, hclformatter.FormatOptions{})
	assert.Error(t, err)
}

func TestFormatter_Format_NilDef(t *testing.T) {
	f := hclformatter.NewFormatter()
	data := resourceData()

	_, err := f.Format(data, nil, hclformatter.FormatOptions{})
	assert.Error(t, err)
}

// ── Dynamic object block tests ──────────────────────────────────

func defWithValueAttr() *schema.ResourceDefinition {
	def := minimalVariableDef()
	def.Attributes = append(def.Attributes, schema.AttributeDefinition{
		Name:          "Value",
		TerraformName: "value",
		Type:          "object",
		Transform:     "custom",
		CustomTransform: "handleVariableValue",
	})
	return def
}

func TestFormatter_Format_DynamicBlockString(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	data.Attributes["value"] = map[string]interface{}{"string": "hello"}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "value = {")
	assert.Contains(t, output, `string = "hello"`)
}

func TestFormatter_Format_DynamicBlockBool(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	data.Attributes["value"] = map[string]interface{}{"bool": true}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "bool = true")
}

func TestFormatter_Format_DynamicBlockInt(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	data.Attributes["value"] = map[string]interface{}{"float32": int64(42)}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "float32 = 42")
}

func TestFormatter_Format_DynamicBlockRawValue(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	data.Attributes["value"] = map[string]interface{}{
		"json_object": core.RawHCLValue(`{"key":"val"}`),
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, `json_object = { "key" : "val" }`)
}

func TestFormatter_Format_DynamicBlockNilSkipped(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	// value attribute not set in data → should be omitted

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.NotContains(t, output, "value =")
}

func TestFormatter_Format_DynamicBlockEmptyMapSkipped(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	data.Attributes["value"] = map[string]interface{}{}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.NotContains(t, output, "value =")
}

// ── Object block tests (nested_attributes) ─────────────────────

func defWithObjectAttr() *schema.ResourceDefinition {
	def := minimalVariableDef()
	def.Attributes = append(def.Attributes, schema.AttributeDefinition{
		Name:          "Config",
		TerraformName: "config",
		Type:          "object",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "Host", TerraformName: "host", Type: "string"},
			{Name: "Port", TerraformName: "port", Type: "string"},
			{
				Name: "EnvRef", TerraformName: "env_ref", Type: "string",
				ReferencesType: "pingone_environment", ReferenceField: "id",
			},
		},
	})
	return def
}

func TestFormatter_Format_ObjectBlock(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithObjectAttr()
	data := resourceData()
	data.Attributes["config"] = map[string]interface{}{
		"host": "example.com",
		"port": "443",
		"env_ref": core.ResolvedReference{
			IsVariable:    true,
			VariableName:  "pingone_environment_id",
			OriginalValue: "env-abc",
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "config")
	assert.Contains(t, output, `"example.com"`)
	assert.Contains(t, output, `"443"`)
	assert.Contains(t, output, "env_ref")
	// Nested environment references resolve to var.pingone_environment_id.
	assert.Contains(t, output, "var.pingone_environment_id")
}

func TestFormatter_Format_ObjectBlockNil(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithObjectAttr()
	data := resourceData()
	// config not set → should be omitted
	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.NotContains(t, output, "config")
}

func TestFormatter_Format_ObjectBlockNotMap(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithObjectAttr()
	data := resourceData()
	data.Attributes["config"] = "not-a-map"
	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.NotContains(t, output, "config")
}

// ── Set with nested attributes tests ───────────────────────────

func defWithSetAttr() *schema.ResourceDefinition {
	def := minimalVariableDef()
	def.Attributes = append(def.Attributes, schema.AttributeDefinition{
		Name:          "Tags",
		TerraformName: "tags",
		Type:          "set",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "Key", TerraformName: "key", Type: "string"},
			{Name: "Value", TerraformName: "value", Type: "string"},
			{Name: "Enabled", TerraformName: "enabled", Type: "bool"},
		},
	})
	return def
}

func TestFormatter_Format_SetWithNestedAttributes(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithSetAttr()
	data := resourceData()
	data.Attributes["tags"] = []interface{}{
		map[string]interface{}{
			"key":     "environment",
			"value":   "production",
			"enabled": true,
		},
		map[string]interface{}{
			"key":     "owner",
			"value":   "team-a",
			"enabled": false,
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)

	// Should contain set blocks with nested attributes
	assert.Contains(t, output, "tags")
	assert.Contains(t, output, `"environment"`)
	assert.Contains(t, output, `"production"`)
	assert.Contains(t, output, `"owner"`)
	assert.Contains(t, output, `"team-a"`)
	assert.Contains(t, output, "true")
	assert.Contains(t, output, "false")
}

func TestFormatter_Format_SetWithNestedAttributesEmpty(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithSetAttr()
	data := resourceData()
	data.Attributes["tags"] = []interface{}{}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	// Empty set should be omitted
	assert.NotContains(t, output, "tags")
}

func TestFormatter_Format_SetWithNestedAttributesNil(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithSetAttr()
	data := resourceData()
	// tags not set → omitted

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.NotContains(t, output, "tags")
}

func TestFormatter_Format_SetWithNestedAttributesReferences(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithSetAttr()
	// Add a reference type attribute to the set definition
	def.Attributes[len(def.Attributes)-1].NestedAttributes = append(
		def.Attributes[len(def.Attributes)-1].NestedAttributes,
		schema.AttributeDefinition{
			Name:           "ConnRef",
			TerraformName:  "conn_ref",
			Type:           "string",
			ReferencesType: "pingone_davinci_connector_instance",
			ReferenceField: "id",
		},
	)

	data := resourceData()
	data.Attributes["tags"] = []interface{}{
		map[string]interface{}{
			"key": "connector-1",
			"value": core.ResolvedReference{
				ResourceType: "pingone_davinci_connector_instance",
				ResourceName: "pingcli__my_connector",
				Field:        "id",
			},
			"enabled": true,
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)

	// Nested references should render as resource traversals
	assert.Contains(t, output, "pingone_davinci_connector_instance.pingcli__my_connector.id")
}

// ── Scalar value type tests ─────────────────────────────────────

func defWithScalars() *schema.ResourceDefinition {
	def := minimalVariableDef()
	def.Attributes = append(def.Attributes,
		schema.AttributeDefinition{Name: "Count", TerraformName: "count_val", Type: "number"},
		schema.AttributeDefinition{Name: "Rate", TerraformName: "rate", Type: "number"},
		schema.AttributeDefinition{Name: "Tags", TerraformName: "tags", Type: "list"},
		schema.AttributeDefinition{Name: "EmptyTags", TerraformName: "empty_tags", Type: "list"},
		schema.AttributeDefinition{Name: "Misc", TerraformName: "misc", Type: "string"},
	)
	return def
}

func TestFormatter_Format_ScalarTypes(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithScalars()
	data := resourceData()
	data.Attributes["count_val"] = int64(99)
	data.Attributes["rate"] = 3.14
	data.Attributes["tags"] = []interface{}{"a", "b"}
	data.Attributes["empty_tags"] = []interface{}{}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "count_val")
	assert.Contains(t, output, "99")
	assert.Contains(t, output, "rate")
	assert.Contains(t, output, "3.14")
	assert.Contains(t, output, `"a"`)
	assert.Contains(t, output, `"b"`)
	assert.Contains(t, output, "empty_tags")
}

func TestFormatter_Format_FallbackScalar(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithScalars()
	data := resourceData()
	data.Attributes["misc"] = struct{ X int }{42}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "misc")
}

// ── Label and name edge cases ────────────────────────────────────

func TestFormatter_Format_IDFallbackLabel(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	def.API.LabelFields = nil
	data := &core.ResourceData{
		ResourceType: "pingone_davinci_variable",
		ID:           "raw-id-123",
		Attributes:   map[string]interface{}{"data_type": "string", "mutable": true},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{SkipDependencies: true})
	require.NoError(t, err)
	assert.Contains(t, output, "raw-id-123")
}

func TestFormatter_Format_TerraformNameFallback(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	// Add attribute with no TerraformName
	def.Attributes = append(def.Attributes, schema.AttributeDefinition{
		Name: "CustomField",
		Type: "string",
	})
	data := resourceData()
	data.Attributes["customfield"] = "val"

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "customfield")
	assert.Contains(t, output, `"val"`)
}

func TestFormatter_Format_ReferenceVarNameNoField(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	def.Attributes = append(def.Attributes, schema.AttributeDefinition{
		Name:           "SomeRef",
		TerraformName:  "some_ref",
		Type:           "string",
		ReferencesType: "some_resource",
	})
	data := resourceData()
	// Pre-resolved variable reference (no ReferenceField → var name is just the type).
	data.Attributes["some_ref"] = core.ResolvedReference{
		IsVariable:    true,
		VariableName:  "some_resource",
		OriginalValue: "uuid-123",
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "var.some_resource")
}

// ── FormatImportBlock nil/empty tests ────────────────────────────

func TestFormatter_FormatImportBlock_NilData(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	_, err := f.FormatImportBlock(nil, def, "env")
	assert.Error(t, err)
}

func TestFormatter_FormatImportBlock_NilDef(t *testing.T) {
	f := hclformatter.NewFormatter()
	data := resourceData()
	_, err := f.FormatImportBlock(data, nil, "env")
	assert.Error(t, err)
}

// ── FormatList error test ────────────────────────────────────────

func TestFormatter_FormatList_Error(t *testing.T) {
	f := hclformatter.NewFormatter()
	// nil def should cause Format to fail for each item
	list := []*core.ResourceData{resourceData()}
	_, err := f.FormatList(list, nil, hclformatter.FormatOptions{})
	assert.Error(t, err)
}

// ── DynamicBlock with float value ────────────────────────────────

func TestFormatter_Format_DynamicBlockFloat(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	data.Attributes["value"] = map[string]interface{}{"float32": float64(3.14)}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "3.14")
}

func TestFormatter_Format_DynamicBlockIntegerFloat(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithValueAttr()
	data := resourceData()
	data.Attributes["value"] = map[string]interface{}{"float32": float64(7)}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, output, "float32 = 7")
}

// ── ResolvedReference rendering tests ─────────────────────

func defWithConnRef() *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_flow",
			APIType:      "Flow",
			Name:         "DaVinci Flow",
			ShortName:    "flow",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/pingidentity/pingone-go-client/davinci",
			SDKType:    "Flow",
			ListMethod: "Environment.Flows.GetAll",
			IDField:    "ID",
			NameField:  "Name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", Computed: true},
			{
				Name: "EnvironmentID", TerraformName: "environment_id", Type: "string",
				Required: true, ReferencesType: "pingone_environment", ReferenceField: "id",
			},
			{Name: "Name", TerraformName: "name", Type: "string", Required: true},
			{
				Name: "ConnectionID", TerraformName: "connection_id", Type: "string",
				ReferencesType: "pingone_davinci_connector_instance", ReferenceField: "id",
			},
		},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: "{env_id}/{resource_id}",
		},
	}
}

func TestFormatter_Format_ResolvedReferenceResourceTraversal(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithConnRef()
	data := &core.ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "flow-123",
		Name:         "My Flow",
		Attributes: map[string]interface{}{
			"id": "flow-123",
			"environment_id": core.ResolvedReference{
				IsVariable:    true,
				VariableName:  "pingone_environment_id",
				OriginalValue: "env-abc",
			},
			"name": "My Flow",
			"connection_id": core.ResolvedReference{
				ResourceType:  "pingone_davinci_connector_instance",
				ResourceName:  "pingcli__http_connector",
				Field:         "id",
				OriginalValue: "conn-456",
			},
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)

	// connection_id should render as resource traversal.
	assert.Contains(t, output, "pingone_davinci_connector_instance.pingcli__http_connector.id")
	// environment_id should render as variable reference.
	assert.Contains(t, output, "var.pingone_environment_id")
}

func TestFormatter_Format_ResolvedReferenceFallbackVar(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := defWithConnRef()
	data := &core.ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "flow-123",
		Name:         "My Flow",
		Attributes: map[string]interface{}{
			"id": "flow-123",
			"environment_id": core.ResolvedReference{
				IsVariable:    true,
				VariableName:  "pingone_environment_id",
				OriginalValue: "env-abc",
			},
			"name": "My Flow",
			// Unresolved reference falls back to variable.
			"connection_id": core.ResolvedReference{
				IsVariable:    true,
				VariableName:  "pingone_davinci_connector_instance_id",
				OriginalValue: "conn-unknown",
			},
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)

	// Fallback to variable reference.
	assert.Contains(t, output, "var.pingone_davinci_connector_instance_id")
}

func TestFormatter_Format_NestedResolvedReference(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := minimalVariableDef()
	def.Attributes = append(def.Attributes, schema.AttributeDefinition{
		Name:          "Config",
		TerraformName: "config",
		Type:          "object",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "Host", TerraformName: "host", Type: "string"},
			{
				Name: "FlowRef", TerraformName: "flow_ref", Type: "string",
				ReferencesType: "pingone_davinci_flow", ReferenceField: "id",
			},
		},
	})
	data := resourceData()
	data.Attributes["config"] = map[string]interface{}{
		"host": "example.com",
		"flow_ref": core.ResolvedReference{
			ResourceType:  "pingone_davinci_flow",
			ResourceName:  "pingcli__my_flow",
			Field:         "id",
			OriginalValue: "flow-888",
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)

	// Nested flow reference should render as resource traversal.
	assert.Contains(t, output, "pingone_davinci_flow.pingcli__my_flow.id")
}

// ── Regression tests: variable-eligible attributes rendering ────

// Regression test: variable-eligible attributes must render as var.X references
func TestFormatter_Format_ScalarVariableEligibleAttribute(t *testing.T) {
	f := hclformatter.NewFormatter()
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			ResourceType: "test_resource",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "Name", TerraformName: "name", Type: "string"},
		},
	}
	data := &core.ResourceData{
		Name: "test",
		Attributes: map[string]interface{}{
			"name": core.ResolvedReference{
				IsVariable:   true,
				VariableName: "davinci_variable_test_name",
			},
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)

	// BUG: This should render as unquoted traversal expression (var.davinci_variable_test_name)
	// but currently renders as quoted string or is missing.
	assert.Contains(t, output, "name = var.davinci_variable_test_name",
		"variable-eligible scalar attributes must render as var.X references without quotes")
}

// Regression test: variable-eligible attributes inside type_discriminated_block must render as var.X references
func TestFormatter_Format_TypeDiscriminatedBlockVariableReference(t *testing.T) {
	f := hclformatter.NewFormatter()

	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			ResourceType: "pingone_davinci_variable",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", Computed: true},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{
				Name:          "Value",
				TerraformName: "value",
				Type:          "type_discriminated_block",
				TypeDiscriminatedBlock: &schema.TypeDiscriminatedBlockConfig{
					TypeKeyMap: map[string]string{
						"string": "string",
						"bool":   "bool",
						"float32": "float32",
					},
				},
			},
		},
	}

	data := &core.ResourceData{
		Name: "my_var",
		ID:   "var-123",
		Attributes: map[string]interface{}{
			"id":   "var-123",
			"name": "my_var",
			"value": map[string]interface{}{
				"string": core.ResolvedReference{
					IsVariable:   true,
					VariableName: "davinci_variable_test_value",
				},
			},
		},
	}

	output, err := f.Format(data, def, hclformatter.FormatOptions{})
	require.NoError(t, err)

	// BUG: The var.X reference inside the block should render without quotes
	assert.Contains(t, output, "value = {",
		"type_discriminated_block should contain a block")
	assert.Contains(t, output, "var.davinci_variable_test_value",
		"variable references inside type_discriminated_block must render as var.X")
}
