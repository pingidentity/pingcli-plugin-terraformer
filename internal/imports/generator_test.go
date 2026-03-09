package imports

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDef(resourceType, importFormat string, labelFields []string) *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			ResourceType: resourceType,
		},
		API: schema.APIDefinition{
			LabelFields: labelFields,
		},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: importFormat,
		},
	}
}

func testData(id, name string, attrs map[string]interface{}) *core.ResourceData {
	return &core.ResourceData{
		ID:         id,
		Name:       name,
		Attributes: attrs,
	}
}

func TestGenerateImportBlock_Simple(t *testing.T) {
	g := NewGenerator()
	def := testDef("pingone_davinci_variable", "{env_id}/{resource_id}", nil)
	data := testData("var-123", "my_var", nil)

	result, err := g.GenerateImportBlock(data, def, "env-456")
	require.NoError(t, err)
	assert.Contains(t, result, "import {")
	assert.Contains(t, result, "to = pingone_davinci_variable.pingcli__my_var")
	assert.Contains(t, result, `"env-456/var-123"`)
}

func TestGenerateImportBlock_LabelFields(t *testing.T) {
	g := NewGenerator()
	def := testDef("pingone_davinci_variable", "{env_id}/{resource_id}", []string{"name", "context"})
	data := testData("var-123", "my_var__company", map[string]interface{}{
		"name":    "my_var",
		"context": "company",
	})

	result, err := g.GenerateImportBlock(data, def, "env-456")
	require.NoError(t, err)
	assert.Contains(t, result, "to = pingone_davinci_variable.pingcli__my_var_company")
	assert.Contains(t, result, `"env-456/var-123"`)
}

func TestGenerateImportBlock_ThreePartID(t *testing.T) {
	g := NewGenerator()
	def := testDef("pingone_davinci_application_flow_policy", "{env_id}/{application_id}/{resource_id}", nil)
	data := testData("policy-abc", "signin_policy", map[string]interface{}{
		"application_id": "app-789",
	})

	result, err := g.GenerateImportBlock(data, def, "env-456")
	require.NoError(t, err)
	assert.Contains(t, result, `"env-456/app-789/policy-abc"`)
}

func TestGenerateImportBlock_MissingPlaceholder(t *testing.T) {
	g := NewGenerator()
	def := testDef("pingone_davinci_application_flow_policy", "{env_id}/{application_id}/{resource_id}", nil)
	// application_id not in attributes
	data := testData("policy-abc", "signin_policy", nil)

	_, err := g.GenerateImportBlock(data, def, "env-456")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "application_id")
}

func TestGenerateImportBlock_NilData(t *testing.T) {
	g := NewGenerator()
	def := testDef("x", "{env_id}/{resource_id}", nil)
	_, err := g.GenerateImportBlock(nil, def, "env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource data is nil")
}

func TestGenerateImportBlock_NilDef(t *testing.T) {
	g := NewGenerator()
	data := testData("id", "name", nil)
	_, err := g.GenerateImportBlock(data, nil, "env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource definition is nil")
}

func TestGenerateImportBlock_EmptyFormat(t *testing.T) {
	g := NewGenerator()
	def := testDef("x", "", nil)
	data := testData("id", "name", nil)
	_, err := g.GenerateImportBlock(data, def, "env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no import_id_format")
}

func TestGenerateImportBlock_MissingEnvID(t *testing.T) {
	g := NewGenerator()
	def := testDef("x", "{env_id}/{resource_id}", nil)
	data := testData("id", "name", nil)
	_, err := g.GenerateImportBlock(data, def, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "env_id")
}

func TestGenerateImportBlock_MissingResourceID(t *testing.T) {
	g := NewGenerator()
	def := testDef("x", "{env_id}/{resource_id}", nil)
	data := testData("", "name", nil)
	_, err := g.GenerateImportBlock(data, def, "env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource_id")
}

func TestGenerateImportBlocks_Sorted(t *testing.T) {
	g := NewGenerator()
	def := testDef("pingone_davinci_variable", "{env_id}/{resource_id}", nil)
	list := []*core.ResourceData{
		testData("id-z", "zzz", nil),
		testData("id-a", "aaa", nil),
	}

	result, err := g.GenerateImportBlocks(list, def, "env-456")
	require.NoError(t, err)

	// aaa should appear before zzz
	idxA := len(result) // safe default
	idxZ := len(result)
	for i := 0; i < len(result)-2; i++ {
		if result[i:i+3] == "aaa" {
			idxA = i
		}
		if result[i:i+3] == "zzz" {
			idxZ = i
		}
	}
	assert.Less(t, idxA, idxZ, "import blocks should be sorted by name")
}

func TestExpandImportID_AllPlaceholders(t *testing.T) {
	data := testData("res-1", "test", map[string]interface{}{
		"parent_id": "parent-2",
	})
	result, err := expandImportID("{env_id}/{parent_id}/{resource_id}", data, "env-3")
	require.NoError(t, err)
	assert.Equal(t, "env-3/parent-2/res-1", result)
}
