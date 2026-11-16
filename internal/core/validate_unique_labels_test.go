package core

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/graph"
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

	err := assignLabels(resources, def, nil)
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

	err := assignLabels(resources, def, nil)
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

	err := assignLabels(resources, def, nil)
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

	err := assignLabels(resources, def, nil)
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

	err := assignLabels(resources, def, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate resource labels disambiguated")
}

func TestAssignLabels_LabelFieldsDotPath_FlowContextVar(t *testing.T) {
	// Plain dot-path label field (no "@type"): value used verbatim. This is
	// the nil-graph fallback form; the graph-resolved "@type" form has its
	// own test (TestVariableLabelFlowRefGraphResolved).
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

	err := assignLabels(resources, def, nil)
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

	err := assignLabels(resources, def, nil)
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

	err := assignLabels(resources, def, nil)
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

	assert.Equal(t, "someVar", LabelFieldValue(attrs, "name", nil))
	assert.Equal(t, "abc123", LabelFieldValue(attrs, "flow.id", nil))
	assert.Equal(t, "x", LabelFieldValue(attrs, "nested.deep.leaf", nil))
	assert.Equal(t, "", LabelFieldValue(attrs, "missing", nil))
	assert.Equal(t, "", LabelFieldValue(attrs, "empty", nil))
	assert.Equal(t, "", LabelFieldValue(attrs, "flow.missing", nil))
	assert.Equal(t, "", LabelFieldValue(attrs, "notmap.sub", nil))
	assert.Equal(t, "", LabelFieldValue(attrs, "name.extra", nil)) // string mid-path
	assert.Equal(t, "", LabelFieldValue(attrs, "", nil))
	assert.Equal(t, "", LabelFieldValue(nil, "name", nil))
}

// TestVariableLabelFlowContextDotPath is the #166 end-to-end check for the
// processor: it must surface the nested flow.id (mirroring the SDK's
// ResourceRelationshipDaVinciReadOnly { Id string }) and terraformLabel must
// fold it into the label. Uses a nil graph, so the "@type" field degrades to
// the raw dot-path value — the graph-resolved form is covered by
// TestVariableLabelFlowRefGraphResolved.
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

	require.NoError(t, assignLabels(processed, def, nil))

	assert.Equal(t, "pingcli__flowContextVarA_flow_"+flowA, processed[0].Label)
	assert.Equal(t, "pingcli__flowContextVarA_flow_"+flowB, processed[1].Label)
	assert.Equal(t, "pingcli__flowInstanceVarA_flowInstance", processed[2].Label)
}

// TestAssignLabels_LegacyFlowInstanceDuplicates covers the #166 degradation
// path: legacy environments can contain two flowInstance-context variables
// with the same name — the current API rejects creating them
// (UNIQUENESS_VIOLATION), so they cannot exist as real fixtures, but the
// exporter must not abort. Both get unique labels (deterministic, ID-ordered)
// and assignLabels returns the warning error the orchestrator reports.
func TestAssignLabels_LegacyFlowInstanceDuplicates(t *testing.T) {
	resources := []*ResourceData{
		// Input order reversed vs ID order: suffix must follow ID order.
		{ID: "legacy-id-2", Name: "cancelUrl", Attributes: map[string]interface{}{
			"name": "cancelUrl", "context": "flowInstance"}},
		{ID: "legacy-id-1", Name: "cancelUrl", Attributes: map[string]interface{}{
			"name": "cancelUrl", "context": "flowInstance"}},
	}
	def := &schema.ResourceDefinition{
		API: schema.APIDefinition{
			NameField:   "name",
			LabelFields: []string{"name", "context", "flow.id"},
		},
	}

	err := assignLabels(resources, def, nil)
	require.Error(t, err) // warning, not fatal — orchestrator reports it
	assert.Contains(t, err.Error(), "duplicate resource labels disambiguated")
	assert.Contains(t, err.Error(), "legacy-id-2")

	// legacy-id-1 sorts first and keeps the base label; legacy-id-2 is suffixed.
	assert.Equal(t, "pingcli__cancelUrl_flowInstance", resources[1].Label)
	assert.Equal(t, "pingcli__cancelUrl_flowInstance_2", resources[0].Label)
}

// TestVariableLabelFlowRefGraphResolved verifies the "@type" label-field form
// end to end: flows are added to the graph first (topo order), then flow-
// context variables resolve their label segment to the flow's label with the
// pingcli__ prefix stripped — pingcli__<var>_flow_<flowLabel>, no double
// prefix, no raw UUID.
func TestVariableLabelFlowRefGraphResolved(t *testing.T) {
	type mockFlow struct {
		Id   string
		Name string
	}
	type mockVariable struct {
		Id      string
		Name    string
		Context string
		Flow    *mockFlow
	}

	flowDef := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform: "pingone", ResourceType: "pingone_davinci_flow",
			APIType: "Flow", Name: "DaVinci Flow", ShortName: "flow", Version: "1.0",
		},
		API: schema.APIDefinition{IDField: "id", NameField: "name"},
	}
	varDef := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform: "pingone", ResourceType: "pingone_davinci_variable",
			APIType: "Variable", Name: "DaVinci Variable", ShortName: "variable", Version: "1.0",
		},
		API: schema.APIDefinition{
			IDField: "id", NameField: "name",
			LabelFields: []string{"name", "context", "flow.id@pingone_davinci_flow"},
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
	reg := newTestRegistry(t, flowDef, varDef)

	flowA, flowB := "flow-aaaa", "flow-bbbb"
	// Label the flows first (as the orchestrator does per topo order), then
	// register them in the graph — the exact sequence production uses.
	flows := []*ResourceData{
		{ResourceType: "pingone_davinci_flow", ID: flowA, Name: "My Flow"},
		{ResourceType: "pingone_davinci_flow", ID: flowB, Name: "Other Flow"},
	}
	require.NoError(t, assignLabels(flows, flowDef, nil))
	g := graph.New()
	for _, f := range flows {
		g.AddResource(f.ResourceType, f.ID, f.Label)
	}
	assert.Equal(t, "pingcli__My-0020-Flow", flows[0].Label)
	assert.Equal(t, "pingcli__Other-0020-Flow", flows[1].Label)

	proc := NewProcessor(reg)
	variables := []interface{}{
		&mockVariable{Id: "var-1", Name: "e2eVarA", Context: "flow", Flow: &mockFlow{Id: flowA}},
		&mockVariable{Id: "var-2", Name: "e2eVarA", Context: "flow", Flow: &mockFlow{Id: flowB}},
		&mockVariable{Id: "var-3", Name: "flowInstanceVarA", Context: "flowInstance"},
	}
	processed, err := proc.ProcessResourceList("pingone_davinci_variable", variables)
	require.NoError(t, err)
	require.NoError(t, assignLabels(processed, varDef, g))

	// Resolved: flow LABEL (prefix-stripped), not the UUID; no pingcli__ twice.
	assert.Equal(t, "pingcli__e2eVarA_flow_My-0020-Flow", processed[0].Label)
	assert.Equal(t, "pingcli__e2eVarA_flow_Other-0020-Flow", processed[1].Label)
	assert.Equal(t, "pingcli__flowInstanceVarA_flowInstance", processed[2].Label)

	// Unresolvable ID (filtered flow, missing from graph): segment skipped,
	// label falls back to the remaining fields — no UUID leak.
	orphan := &ResourceData{ResourceType: "pingone_davinci_variable", ID: "var-4",
		Attributes: map[string]interface{}{
			"name": "e2eVarA", "context": "flow",
			"flow": map[string]interface{}{"id": "not-in-graph"},
		}}
	require.NoError(t, assignLabels([]*ResourceData{orphan}, varDef, g))
	assert.Equal(t, "pingcli__e2eVarA_flow", orphan.Label)
}

// TestLabelFieldValueGraphForms pins the LabelFieldValue "@type" resolution
// matrix against a small graph.
func TestLabelFieldValueGraphForms(t *testing.T) {
	attrs := map[string]interface{}{
		"name": "someVar",
		"flow": map[string]interface{}{"id": "flow-1"},
	}
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "flow-1", "pingcli__My-0020-Flow")

	// Resolved, prefix stripped.
	assert.Equal(t, "My-0020-Flow", LabelFieldValue(attrs, "flow.id@pingone_davinci_flow", g))
	// Missing target in graph → "".
	assert.Equal(t, "", LabelFieldValue(attrs, "missing.id@pingone_davinci_flow", g))
	// Nil graph → raw value fallback (dot-path behavior preserved).
	assert.Equal(t, "flow-1", LabelFieldValue(attrs, "flow.id@pingone_davinci_flow", nil))
	// Malformed @-syntax → "".
	assert.Equal(t, "", LabelFieldValue(attrs, "@", g))
	assert.Equal(t, "", LabelFieldValue(attrs, "flow.id@", g))
	assert.Equal(t, "", LabelFieldValue(attrs, "@pingone_davinci_flow", g))
	// Plain fields unaffected by graph presence.
	assert.Equal(t, "someVar", LabelFieldValue(attrs, "name", g))
}
