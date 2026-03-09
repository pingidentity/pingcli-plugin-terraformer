package core

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignLabels_NoDuplicates(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "Flow A", Attributes: map[string]interface{}{"name": "Flow A"}},
		{ID: "id2", Name: "Flow B", Attributes: map[string]interface{}{"name": "Flow B"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{NameField: "name"},
	}

	err := assignLabels(resources, def)
	assert.NoError(t, err)
	assert.Equal(t, "pingcli__Flow-0020-A", resources[0].Label)
	assert.Equal(t, "pingcli__Flow-0020-B", resources[1].Label)
}

func TestAssignLabels_DuplicateNames(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "My Flow", Attributes: map[string]interface{}{"name": "My Flow"}},
		{ID: "id2", Name: "My Flow", Attributes: map[string]interface{}{"name": "My Flow"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{NameField: "name"},
	}

	err := assignLabels(resources, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate resource labels")
	assert.Contains(t, err.Error(), "id1")
	assert.Contains(t, err.Error(), "id2")
}

func TestAssignLabels_LabelFieldsComposite_NoFalsePositive(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "origin", Attributes: map[string]interface{}{"name": "origin", "context": "company"}},
		{ID: "id2", Name: "origin", Attributes: map[string]interface{}{"name": "origin", "context": "flowInstance"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:   "name",
			LabelFields: []string{"name", "context"},
		},
	}

	err := assignLabels(resources, def)
	assert.NoError(t, err)
	assert.Equal(t, "pingcli__origin_company", resources[0].Label)
	assert.Equal(t, "pingcli__origin_flowInstance", resources[1].Label)
}

func TestAssignLabels_LabelFieldsComposite_Duplicate(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "origin", Attributes: map[string]interface{}{"name": "origin", "context": "company"}},
		{ID: "id2", Name: "origin", Attributes: map[string]interface{}{"name": "origin", "context": "company"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:   "name",
			LabelFields: []string{"name", "context"},
		},
	}

	err := assignLabels(resources, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate resource labels")
}

func TestAssignLabels_EmptyNameIDFallback_NoCollision(t *testing.T) {
	resources := []*ResourceData{
		{ID: "uuid-1", Name: "", Attributes: map[string]interface{}{}},
		{ID: "uuid-2", Name: "", Attributes: map[string]interface{}{}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{NameField: "name"},
	}

	err := assignLabels(resources, def)
	assert.NoError(t, err)
	assert.Equal(t, "uuid-1", resources[0].Label)
	assert.Equal(t, "uuid-2", resources[1].Label)
}

func TestAssignLabels_AllowDuplicateLabels(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
		{ID: "id2", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:            "name",
			AllowDuplicateLabels: true,
		},
	}

	err := assignLabels(resources, def)
	assert.NoError(t, err)
	assert.Equal(t, "pingcli__My-0020-Policy", resources[0].Label)
	assert.Equal(t, "pingcli__My-0020-Policy_2", resources[1].Label)
}

func TestAssignLabels_AllowDuplicateLabels_False(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
		{ID: "id2", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:            "name",
			AllowDuplicateLabels: false,
		},
	}

	err := assignLabels(resources, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate resource labels")
}

func TestAssignLabels_AllowDuplicateLabels_Disambiguates(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
		{ID: "id2", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:            "name",
			AllowDuplicateLabels: true,
		},
	}

	err := assignLabels(resources, def)
	require.NoError(t, err)
	assert.Equal(t, "pingcli__My-0020-Policy", resources[0].Label)
	assert.Equal(t, "pingcli__My-0020-Policy_2", resources[1].Label)
}

func TestAssignLabels_ThreeDuplicates_AllowDuplicateLabels(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
		{ID: "id2", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
		{ID: "id3", Name: "My Policy", Attributes: map[string]interface{}{"name": "My Policy"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:            "name",
			AllowDuplicateLabels: true,
		},
	}

	err := assignLabels(resources, def)
	require.NoError(t, err)
	assert.Equal(t, "pingcli__My-0020-Policy", resources[0].Label)
	assert.Equal(t, "pingcli__My-0020-Policy_2", resources[1].Label)
	assert.Equal(t, "pingcli__My-0020-Policy_3", resources[2].Label)
}
