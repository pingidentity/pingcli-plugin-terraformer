package tfjson

import (
	"encoding/json"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unmarshalResource parses formatter output and returns the inner attribute map
// for the given resource type and label.
func unmarshalResource(t *testing.T, output, resourceType, label string) map[string]interface{} {
	t.Helper()
	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &doc))
	res, ok := doc["resource"].(map[string]interface{})
	require.True(t, ok, "expected top-level \"resource\" key")
	byType, ok := res[resourceType].(map[string]interface{})
	require.True(t, ok, "expected resource type %q", resourceType)
	byLabel, ok := byType[label].(map[string]interface{})
	require.True(t, ok, "expected label %q", label)
	return byLabel
}

func baseDef(attrs ...schema.AttributeDefinition) *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{ResourceType: "test_resource"},
		Attributes: attrs,
	}
}

func baseData(name, id string, attrs map[string]interface{}) *core.ResourceData {
	return &core.ResourceData{
		Name:       name,
		ID:         id,
		Attributes: attrs,
	}
}

func TestFileExtension(t *testing.T) {
	f := NewFormatter()
	assert.Equal(t, ".tf.json", f.FileExtension())
}

func TestFormat_NilData(t *testing.T) {
	f := NewFormatter()
	_, err := f.Format(nil, baseDef(), FormatOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource data is nil")
}

func TestFormat_NilDef(t *testing.T) {
	f := NewFormatter()
	_, err := f.Format(baseData("x", "id1", nil), nil, FormatOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource definition is nil")
}

func TestFormat_EmptyNameAndID(t *testing.T) {
	f := NewFormatter()
	_, err := f.Format(baseData("", "", nil), baseDef(), FormatOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource has neither Name nor ID set")
}

func TestFormat_ScalarAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrType string
		value    interface{}
		expected interface{}
	}{
		{"string", "string", "hello", "hello"},
		{"bool_true", "bool", true, true},
		{"bool_false", "bool", false, false},
		{"int64", "number", int64(42), float64(42)},
		{"float64", "number", float64(3.14), float64(3.14)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFormatter()
			def := baseDef(schema.AttributeDefinition{
				Name: "my_attr", TerraformName: "my_attr", Type: tc.attrType,
			})
			data := baseData("res1", "id1", map[string]interface{}{"my_attr": tc.value})
			out, err := f.Format(data, def, FormatOptions{})
			require.NoError(t, err)
			attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
			assert.Equal(t, tc.expected, attrs["my_attr"])
		})
	}
}

func TestFormat_NilAttributeSkipped(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "my_attr", TerraformName: "my_attr", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{"my_attr": nil})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	_, exists := attrs["my_attr"]
	assert.False(t, exists, "nil attribute should be omitted")
}

func TestFormat_MissingAttributeSkipped(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "my_attr", TerraformName: "my_attr", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	_, exists := attrs["my_attr"]
	assert.False(t, exists, "missing attribute should be omitted")
}

func TestFormat_ComputedOnlySkipped(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "computed_attr", TerraformName: "computed_attr", Type: "string",
		Computed: true, Required: false,
	})
	data := baseData("res1", "id1", map[string]interface{}{"computed_attr": "value"})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	_, exists := attrs["computed_attr"]
	assert.False(t, exists, "computed-only attribute should be omitted")
}

func TestFormat_ResolvedResourceReference(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "env_id", TerraformName: "env_id", Type: "string",
		ReferencesType: "other_resource",
	})
	ref := core.ResolvedReference{
		ResourceType: "other_resource",
		ResourceName: "other_label",
		Field:        "id",
	}
	data := baseData("res1", "id1", map[string]interface{}{"env_id": ref})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	assert.Equal(t, "${other_resource.other_label.id}", attrs["env_id"])
}

func TestFormat_ResolvedVariableReference(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "env_id", TerraformName: "env_id", Type: "string",
		ReferencesType: "some_resource",
	})
	ref := core.ResolvedReference{
		IsVariable:   true,
		VariableName: "my_var",
	}
	data := baseData("res1", "id1", map[string]interface{}{"env_id": ref})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	assert.Equal(t, "${var.my_var}", attrs["env_id"])
}

func TestFormat_RawStringReference(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "env_id", TerraformName: "env_id", Type: "string",
		ReferencesType: "some_resource",
	})
	data := baseData("res1", "id1", map[string]interface{}{"env_id": "raw-uuid-123"})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	assert.Equal(t, "raw-uuid-123", attrs["env_id"])
}

func TestFormat_EmptyReferenceSkipped(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "env_id", TerraformName: "env_id", Type: "string",
		ReferencesType: "some_resource",
	})
	data := baseData("res1", "id1", map[string]interface{}{"env_id": ""})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	_, exists := attrs["env_id"]
	assert.False(t, exists, "empty reference string should be omitted")
}

func TestFormat_RawHCLValue_Scalar(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "expr", TerraformName: "expr", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"expr": core.RawHCLValue("jsonencode({\"key\": \"value\"})"),
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	assert.Equal(t, "${jsonencode({\"key\": \"value\"})}", attrs["expr"])
}

func TestFormat_RawHCLValue_DynamicObject(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "dynamic", TerraformName: "dynamic", Type: "object",
		// No NestedAttributes → dynamic object path
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"dynamic": core.RawHCLValue("some_expression()"),
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	assert.Equal(t, "${some_expression()}", attrs["dynamic"])
}

func TestFormat_NestedObject(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "config", TerraformName: "config", Type: "object",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "enabled", TerraformName: "enabled", Type: "bool"},
			{Name: "timeout", TerraformName: "timeout", Type: "number"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"config": map[string]interface{}{
			"enabled": true,
			"timeout": int64(30),
		},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	cfg, ok := attrs["config"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, cfg["enabled"])
	assert.Equal(t, float64(30), cfg["timeout"])
}

func TestFormat_EmptyNestedObjectSkipped(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "config", TerraformName: "config", Type: "object",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "enabled", TerraformName: "enabled", Type: "bool"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"config": map[string]interface{}{},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	_, exists := attrs["config"]
	assert.False(t, exists, "empty nested object should be omitted")
}

func TestFormat_MapWithNestedSchema(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "settings", TerraformName: "settings", Type: "map",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "value", TerraformName: "value", Type: "string"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"settings": map[string]interface{}{
			"bravo": map[string]interface{}{"value": "b_val"},
			"alpha": map[string]interface{}{"value": "a_val"},
		},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	settings, ok := attrs["settings"].(map[string]interface{})
	require.True(t, ok)
	alpha, ok := settings["alpha"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "a_val", alpha["value"])
	bravo, ok := settings["bravo"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "b_val", bravo["value"])
}

func TestFormat_ListOfObjects(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "items", TerraformName: "items", Type: "list",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "name", TerraformName: "name", Type: "string"},
			{Name: "count", TerraformName: "count", Type: "number"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "first", "count": int64(1)},
			map[string]interface{}{"name": "second", "count": int64(2)},
		},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	items, ok := attrs["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 2)
	first, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "first", first["name"])
	assert.Equal(t, float64(1), first["count"])
}

func TestFormat_SetWithNestedAttributes(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "tags", TerraformName: "tags", Type: "set",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "key", TerraformName: "key", Type: "string"},
			{Name: "value", TerraformName: "value", Type: "string"},
			{Name: "enabled", TerraformName: "enabled", Type: "bool"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"tags": []interface{}{
			map[string]interface{}{"key": "env", "value": "prod", "enabled": true},
			map[string]interface{}{"key": "team", "value": "alpha", "enabled": false},
		},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	tags, ok := attrs["tags"].([]interface{})
	require.True(t, ok)
	require.Len(t, tags, 2)
	tag0, ok := tags[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "env", tag0["key"])
	assert.Equal(t, "prod", tag0["value"])
	assert.Equal(t, true, tag0["enabled"])
	tag1, ok := tags[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "team", tag1["key"])
	assert.Equal(t, "alpha", tag1["value"])
	assert.Equal(t, false, tag1["enabled"])
}

func TestFormat_SetWithNestedAttributesEmpty(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "tags", TerraformName: "tags", Type: "set",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "key", TerraformName: "key", Type: "string"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"tags": []interface{}{},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	// Empty set present in attributes should be rendered as empty array
	tags, ok := attrs["tags"]
	assert.True(t, ok, "empty set should be rendered")
	assert.Equal(t, []interface{}{}, tags, "empty set should be an empty array")
}

func TestFormat_SetWithNestedAttributesNil(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "tags", TerraformName: "tags", Type: "set",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "key", TerraformName: "key", Type: "string"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	// Nil set should not be present
	_, ok := attrs["tags"]
	assert.False(t, ok, "nil set should be omitted")
}

func TestFormat_DynamicObject_Map(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "extra", TerraformName: "extra", Type: "object",
		// No NestedAttributes → dynamic path
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"extra": map[string]interface{}{
			"key1": "val1",
			"key2": int64(99),
		},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	extra, ok := attrs["extra"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "val1", extra["key1"])
	// int64 rendered via renderScalar, then JSON round-trips to float64
	assert.Equal(t, float64(99), extra["key2"])
}

func TestFormat_DynamicObject_RawHCLValue(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "extra", TerraformName: "extra", Type: "object",
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"extra": map[string]interface{}{
			"raw_field": core.RawHCLValue("some_func()"),
			"normal":    "plain",
		},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	extra, ok := attrs["extra"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "${some_func()}", extra["raw_field"])
	assert.Equal(t, "plain", extra["normal"])
}

func TestFormat_ListScalars(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "tags", TerraformName: "tags", Type: "list",
		// No NestedAttributes → falls through to scalar path
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"tags": []interface{}{"alpha", "bravo"},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	tags, ok := attrs["tags"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"alpha", "bravo"}, tags)
}

func TestFormatList_SortedByName(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	dataList := []*core.ResourceData{
		baseData("zulu", "id3", map[string]interface{}{"val": "z"}),
		baseData("alpha", "id1", map[string]interface{}{"val": "a"}),
		baseData("mike", "id2", map[string]interface{}{"val": "m"}),
	}
	out, err := f.FormatList(dataList, def, FormatOptions{})
	require.NoError(t, err)

	// FormatList produces a single JSON document with all labels merged
	// under "resource"."test_resource".
	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &doc))

	res, ok := doc["resource"].(map[string]interface{})
	require.True(t, ok)
	byType, ok := res["test_resource"].(map[string]interface{})
	require.True(t, ok)
	require.Len(t, byType, 3)

	// All three labels present.
	assert.Contains(t, byType, "pingcli__alpha")
	assert.Contains(t, byType, "pingcli__mike")
	assert.Contains(t, byType, "pingcli__zulu")

	// Spot-check values.
	alphaAttrs := byType["pingcli__alpha"].(map[string]interface{})
	assert.Equal(t, "a", alphaAttrs["val"])
}

func TestFormatImportBlock_Basic(t *testing.T) {
	f := NewFormatter()
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{ResourceType: "test_resource"},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: "{env_id}/{resource_id}",
		},
	}
	data := baseData("my_res", "abc-123", nil)
	out, err := f.FormatImportBlock(data, def, "env-456")
	require.NoError(t, err)

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &doc))
	importArr, ok := doc["import"].([]interface{})
	require.True(t, ok)
	require.Len(t, importArr, 1)
	entry, ok := importArr[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test_resource.pingcli__my_res", entry["to"])
	assert.Equal(t, "env-456/abc-123", entry["id"])
}

func TestFormatImportBlock_AttributePlaceholder(t *testing.T) {
	f := NewFormatter()
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{ResourceType: "test_resource"},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: "{env_id}/{resource_id}/{app_name}",
		},
	}
	data := baseData("my_res", "rid-1", map[string]interface{}{"app_name": "myapp"})
	out, err := f.FormatImportBlock(data, def, "env-789")
	require.NoError(t, err)

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &doc))
	importArr := doc["import"].([]interface{})
	entry := importArr[0].(map[string]interface{})
	assert.Equal(t, "env-789/rid-1/myapp", entry["id"])
}

func TestFormatImportBlock_ResolvedReferenceOriginalValue(t *testing.T) {
	f := NewFormatter()
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{ResourceType: "test_resource"},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: "{env_id}/{resource_id}/{ref_attr}",
		},
	}
	data := baseData("my_res", "rid-1", map[string]interface{}{
		"ref_attr": core.ResolvedReference{OriginalValue: "orig-uuid"},
	})
	out, err := f.FormatImportBlock(data, def, "env-1")
	require.NoError(t, err)

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &doc))
	importArr := doc["import"].([]interface{})
	entry := importArr[0].(map[string]interface{})
	assert.Equal(t, "env-1/rid-1/orig-uuid", entry["id"])
}

func TestFormatImportBlock_NilData(t *testing.T) {
	f := NewFormatter()
	_, err := f.FormatImportBlock(nil, baseDef(), "env-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource data is nil")
}

func TestFormatImportBlock_NilDef(t *testing.T) {
	f := NewFormatter()
	_, err := f.FormatImportBlock(baseData("x", "id1", nil), nil, "env-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource definition is nil")
}

func TestFormat_LabelUsesNameSanitized(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	data := baseData("My Resource (v2)", "id1", map[string]interface{}{"val": "x"})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	// SanitizeResourceName hex-encodes space and parens
	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &doc))
	res := doc["resource"].(map[string]interface{})
	byType := res["test_resource"].(map[string]interface{})
	// The label should start with "pingcli__"
	for label := range byType {
		assert.True(t, len(label) > 0)
		assert.Contains(t, label, "pingcli__")
	}
}

func TestFormat_LabelFallsBackToID(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	data := baseData("", "fallback-id", map[string]interface{}{"val": "x"})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	_ = unmarshalResource(t, out, "test_resource", "fallback-id")
}

func TestFormat_LabelFields(t *testing.T) {
	f := NewFormatter()
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{ResourceType: "test_resource"},
		API: schema.APIDefinition{
			LabelFields: []string{"name", "context"},
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "val", TerraformName: "val", Type: "string"},
		},
	}
	data := baseData("ignored", "id1", map[string]interface{}{
		"name":    "myvar",
		"context": "company",
		"val":     "x",
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	// SanitizeMultiKeyResourceName joins with underscore and prefixes pingcli__
	_ = unmarshalResource(t, out, "test_resource", "pingcli__myvar_company")
}

func TestFormat_TerraformNameFallback(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "MyAttr", TerraformName: "", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{"myattr": "hello"})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	assert.Equal(t, "hello", attrs["myattr"])
}

func TestFormat_NestedObject_Reference(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "config", TerraformName: "config", Type: "object",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "ref_field", TerraformName: "ref_field", Type: "string"},
		},
	})
	ref := core.ResolvedReference{
		ResourceType: "other_type",
		ResourceName: "other_label",
		Field:        "id",
	}
	data := baseData("res1", "id1", map[string]interface{}{
		"config": map[string]interface{}{"ref_field": ref},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	cfg := attrs["config"].(map[string]interface{})
	assert.Equal(t, "${other_type.other_label.id}", cfg["ref_field"])
}

func TestFormat_TypeDiscriminatedBlock(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "value", TerraformName: "value", Type: "type_discriminated_block",
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"value": map[string]interface{}{"string": "hello"},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	valBlock, ok := attrs["value"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "hello", valBlock["string"])
}

func TestFormat_OutputEndsWithNewline(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{"val": "x"})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	assert.True(t, len(out) > 0)
	assert.Equal(t, "\n", string(out[len(out)-1]))
}

// ── nil_value: keep_empty on collection types formatter tests ────

// TestFormat_SetWithNestedAttributesKeepEmpty verifies that an empty
// []interface{}{} value (produced by the processor when nil_value: keep_empty is
// configured) for a set-with-nested-attributes renders as "js_links": [] in JSON
// output rather than being omitted.
//
// This test FAILS before the formatter is updated: the tfjson formatter currently
// skips when len(slice) == 0, omitting the attribute entirely.
func TestFormat_SetWithNestedAttributesKeepEmpty(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name:          "js_links",
		TerraformName: "js_links",
		Type:          "set",
		NilValue:      "keep_empty",
		NestedAttributes: []schema.AttributeDefinition{
			{Name: "value", TerraformName: "value", Type: "string"},
			{Name: "label", TerraformName: "label", Type: "string"},
		},
	})
	data := baseData("res1", "id1", map[string]interface{}{
		"js_links": []interface{}{}, // explicitly set empty slice
	})

	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)

	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")

	// FAILING ASSERTION: js_links should be present as [] not absent.
	jsLinksVal, exists := attrs["js_links"]
	assert.True(t, exists, "js_links should be present in JSON output when nil_value: keep_empty produces an empty slice")
	if exists {
		jsLinksSlice, ok := jsLinksVal.([]interface{})
		assert.True(t, ok, "js_links should be a JSON array, got %T", jsLinksVal)
		if ok {
			assert.Empty(t, jsLinksSlice, "js_links should be an empty JSON array []")
		}
	}
}

func TestFormat_ProducesValidJSON(t *testing.T) {
	f := NewFormatter()
	def := baseDef(
		schema.AttributeDefinition{Name: "str", TerraformName: "str", Type: "string"},
		schema.AttributeDefinition{Name: "num", TerraformName: "num", Type: "number"},
		schema.AttributeDefinition{Name: "flag", TerraformName: "flag", Type: "bool"},
	)
	data := baseData("res1", "id1", map[string]interface{}{
		"str":  "hello",
		"num":  int64(42),
		"flag": true,
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)
	assert.True(t, json.Valid([]byte(out)))
}

// ── depends_on rendering tests ─────────────────────────────────

// TestFormat_DependsOnRendered verifies that when DependsOnResources is non-empty
// with resolved labels, the JSON output contains a depends_on array.
func TestFormat_DependsOnRendered(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{"val": "x"})
	data.DependsOnResources = []core.RuntimeDependsOn{
		{ResourceType: "pingone_davinci_variable", ResourceID: "var-1", Label: "pingcli__my_var"},
		{ResourceType: "pingone_davinci_variable", ResourceID: "var-2", Label: "pingcli__other_var"},
	}

	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)

	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	dependsOn, ok := attrs["depends_on"].([]interface{})
	require.True(t, ok, "depends_on should be a JSON array")
	require.Len(t, dependsOn, 2)
	assert.Equal(t, "pingone_davinci_variable.pingcli__my_var", dependsOn[0])
	assert.Equal(t, "pingone_davinci_variable.pingcli__other_var", dependsOn[1])
}

// TestFormat_DependsOnEmpty_NotRendered verifies that no depends_on key is
// emitted when DependsOnResources is nil.
func TestFormat_DependsOnEmpty_NotRendered(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{"val": "x"})
	data.DependsOnResources = nil

	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)

	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	_, exists := attrs["depends_on"]
	assert.False(t, exists, "depends_on should be absent when DependsOnResources is nil")
}

// TestFormat_DependsOnUnresolvedLabelsSkipped verifies that entries with an
// empty Label are omitted from the depends_on array.
func TestFormat_DependsOnUnresolvedLabelsSkipped(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{"val": "x"})
	data.DependsOnResources = []core.RuntimeDependsOn{
		{ResourceType: "pingone_davinci_variable", ResourceID: "var-1", Label: "pingcli__my_var"},
		{ResourceType: "pingone_davinci_variable", ResourceID: "var-unresolved", Label: ""},
	}

	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)

	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	dependsOn, ok := attrs["depends_on"].([]interface{})
	require.True(t, ok, "depends_on should be a JSON array")
	require.Len(t, dependsOn, 1, "unresolved entry (empty label) should be skipped")
	assert.Equal(t, "pingone_davinci_variable.pingcli__my_var", dependsOn[0])
}

// TestFormat_DependsOnAllUnresolved_NotRendered verifies that depends_on is
// omitted entirely when all entries have empty labels.
func TestFormat_DependsOnAllUnresolved_NotRendered(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
		Name: "val", TerraformName: "val", Type: "string",
	})
	data := baseData("res1", "id1", map[string]interface{}{"val": "x"})
	data.DependsOnResources = []core.RuntimeDependsOn{
		{ResourceType: "pingone_davinci_variable", ResourceID: "var-1", Label: ""},
	}

	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)

	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	_, exists := attrs["depends_on"]
	assert.False(t, exists, "depends_on should be absent when all labels are unresolved")
}

// Regression test: variable-eligible attributes inside type_discriminated_block must render as var.X references
func TestFormat_TypeDiscriminatedBlock_ResolvedVariableReference(t *testing.T) {
	f := NewFormatter()
	def := baseDef(schema.AttributeDefinition{
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
	})
	ref := core.ResolvedReference{
		IsVariable:   true,
		VariableName: "davinci_variable_test_value",
	}
	data := baseData("res1", "id1", map[string]interface{}{
		"value": map[string]interface{}{
			"string": ref,
		},
	})
	out, err := f.Format(data, def, FormatOptions{})
	require.NoError(t, err)

	// BUG: ResolvedReference inside type_discriminated_block should render as ${var.X}
	attrs := unmarshalResource(t, out, "test_resource", "pingcli__res1")
	value, ok := attrs["value"].(map[string]interface{})
	require.True(t, ok, "value should be a map object")
	// The "string" key inside the block should contain the variable reference
	assert.Equal(t, "${var.davinci_variable_test_value}", value["string"],
		"variable references inside type_discriminated_block must render as ${var.X}")
}

