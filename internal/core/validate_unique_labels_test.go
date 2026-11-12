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

func TestAssignLabels_DuplicateNames_DisambiguatesWithWarning(t *testing.T) {
	resources := []*ResourceData{
		{ID: "id1", Name: "My Flow", Attributes: map[string]interface{}{"name": "My Flow"}},
		{ID: "id2", Name: "My Flow", Attributes: map[string]interface{}{"name": "My Flow"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{NameField: "name"},
	}

	err := assignLabels(resources, def)
	require.Error(t, err) // returned as a warning by the orchestrator, not fatal
	assert.Contains(t, err.Error(), "duplicate resource labels disambiguated")
	assert.Contains(t, err.Error(), "id2") // the disambiguated (suffixed) resource
	// Both resources get unique labels: first keeps the base, second suffixed.
	assert.Equal(t, "pingcli__My-0020-Flow", resources[0].Label)
	assert.Equal(t, "pingcli__My-0020-Flow_2", resources[1].Label)
}

func TestAssignLabels_DuplicateSuffix_DeterministicByID(t *testing.T) {
	// Input order deliberately reversed relative to ID order: the _2 suffix
	// must land on the resource with the greater ID, not whichever came last
	// in the API list.
	resources := []*ResourceData{
		{ID: "bbb", Name: "Var", Attributes: map[string]interface{}{"name": "Var"}},
		{ID: "aaa", Name: "Var", Attributes: map[string]interface{}{"name": "Var"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{NameField: "name"},
	}

	err := assignLabels(resources, def)
	require.Error(t, err)
	assert.Equal(t, "pingcli__Var", resources[1].Label)   // ID "aaa" wins base
	assert.Equal(t, "pingcli__Var_2", resources[0].Label) // ID "bbb" suffixed
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
	assert.Contains(t, err.Error(), "duplicate resource labels disambiguated")
}

func TestAssignLabels_LabelFieldsDotPath_FlowContextVar(t *testing.T) {
	// Issue #166: flow-context variables are unique per (name, context, flow).
	// Two same-named flow-context variables on different flows must label apart
	// via the flow.id dot-path; company/flowInstance vars (no Flow) are unchanged.
	resources := []*ResourceData{
		{ID: "id1", Name: "flowContextVarA", Attributes: map[string]interface{}{
			"name": "flowContextVarA", "context": "flow",
			"flow": map[string]interface{}{"id": "flow-a-id"},
		}},
		{ID: "id2", Name: "flowContextVarA", Attributes: map[string]interface{}{
			"name": "flowContextVarA", "context": "flow",
			"flow": map[string]interface{}{"id": "flow-b-id"},
		}},
		// flowInstance variable with no flow attribute — label must stay two-part.
		{ID: "id3", Name: "flowInstanceVarA", Attributes: map[string]interface{}{
			"name": "flowInstanceVarA", "context": "flowInstance",
		}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:   "name",
			LabelFields: []string{"name", "context", "flow.id"},
		},
	}

	err := assignLabels(resources, def)
	require.NoError(t, err)
	assert.Equal(t, "pingcli__flowContextVarA_flow_flow-a-id", resources[0].Label)
	assert.Equal(t, "pingcli__flowContextVarA_flow_flow-b-id", resources[1].Label)
	assert.Equal(t, "pingcli__flowInstanceVarA_flowInstance", resources[2].Label)
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

func TestLabelFieldValue(t *testing.T) {
	attrs := map[string]interface{}{
		"name":    "someVar",
		"context": "flow",
		"flow":    map[string]interface{}{"id": "abc123"},
		"empty":   "",
		"nested":  map[string]interface{}{"deep": map[string]interface{}{"leaf": "x"}},
		"notmap":  "scalar",
	}

	assert.Equal(t, "someVar", LabelFieldValue(attrs, "name"))
	assert.Equal(t, "abc123", LabelFieldValue(attrs, "flow.id"))
	assert.Equal(t, "x", LabelFieldValue(attrs, "nested.deep.leaf"))
	assert.Equal(t, "", LabelFieldValue(attrs, "missing"))
	assert.Equal(t, "", LabelFieldValue(attrs, "empty"))
	assert.Equal(t, "", LabelFieldValue(attrs, "flow.missing"))
	assert.Equal(t, "", LabelFieldValue(attrs, "notmap.sub"))
	assert.Equal(t, "", LabelFieldValue(attrs, "name.extra")) // string mid-path
	assert.Equal(t, "", LabelFieldValue(attrs, ""))
	assert.Equal(t, "", LabelFieldValue(nil, "name"))
}

// TestVariableLabelFlowContextDotPath is the #166 end-to-end check: the
// processor must surface the nested flow.id (mirroring the SDK's
// ResourceRelationshipDaVinciReadOnly { Id string }) and terraformLabel must
// fold it into the label via the flow.id dot-path.
func TestVariableLabelFlowContextDotPath(t *testing.T) {
	type mockFlow struct {
		Id string
	}
	type mockVariable struct {
		Id      string
		Name    string
		Context string
		Flow    *mockFlow
	}

	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform: "pingone", ResourceType: "pingone_davinci_variable",
			APIType: "Variable", Name: "DaVinci Variable", ShortName: "variable", Version: "1.0",
		},
		API: schema.APIDefinition{
			IDField: "id", NameField: "name",
			LabelFields: []string{"name", "context", "flow.id"},
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "Id", Computed: true},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name", Required: true},
			{Name: "Context", TerraformName: "context", Type: "string", SourcePath: "Context", Required: true},
			{Name: "Flow", TerraformName: "flow", Type: "object", SourcePath: "Flow",
				NestedAttributes: []schema.AttributeDefinition{
					{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "Id"},
				}},
		},
	}

	proc := NewProcessor(newTestRegistry(t, def))

	flowA := "4b5d052aabccc96d00dc6017b473f8a6"
	flowB := "c95c6190d34283a6e9473ee909e7a42e"
	list := []interface{}{
		&mockVariable{Id: "var-1", Name: "flowContextVarA", Context: "flow", Flow: &mockFlow{Id: flowA}},
		&mockVariable{Id: "var-2", Name: "flowContextVarA", Context: "flow", Flow: &mockFlow{Id: flowB}},
		&mockVariable{Id: "var-3", Name: "flowInstanceVarA", Context: "flowInstance"},
	}

	processed, err := proc.ProcessResourceList("pingone_davinci_variable", list)
	require.NoError(t, err)
	require.Len(t, processed, 3)

	// flow object attribute must surface as a map keyed by terraform_name.
	assert.Equal(t, map[string]interface{}{"id": flowA}, processed[0].Attributes["flow"])

	require.NoError(t, assignLabels(processed, def))

	assert.Equal(t, "pingcli__flowContextVarA_flow_"+flowA, processed[0].Label)
	assert.Equal(t, "pingcli__flowContextVarA_flow_"+flowB, processed[1].Label)
	assert.Equal(t, "pingcli__flowInstanceVarA_flowInstance", processed[2].Label)
}
