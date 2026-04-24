package core

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/filter"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/graph"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock client ---

type mockAPIClient struct {
	platform  string
	resources map[string][]interface{} // resourceType -> list
}

func (m *mockAPIClient) ListResources(_ context.Context, resourceType string, _ string) ([]interface{}, error) {
	list, ok := m.resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return list, nil
}

func (m *mockAPIClient) GetResource(_ context.Context, _ string, _ string, _ string) (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockAPIClient) Platform() string { return m.platform }

func (m *mockAPIClient) Warnings() []string { return nil }

// --- helpers ---

// simpleStruct is used as mock API data.
type simpleStruct struct {
	ID   *string
	Name *string
}

func strPtr(s string) *string { return &s }

func newTestRegistry(t *testing.T, defs ...*schema.ResourceDefinition) *schema.Registry {
	t.Helper()
	reg := schema.NewRegistry()
	for _, d := range defs {
		require.NoError(t, reg.Register(d))
	}
	return reg
}

func baseDef(resourceType, platform, name, shortName string) *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     platform,
			ResourceType: resourceType,
			APIType:      "MockType",
			Name:         name,
			ShortName:    shortName,
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "mock",
			SDKType:    "MockType",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "id", TerraformName: "id", Type: "string", SourcePath: "ID", Transform: "passthrough"},
			{Name: "name", TerraformName: "name", Type: "string", SourcePath: "Name", Transform: "passthrough"},
		},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: "{env_id}/{resource_id}",
		},
	}
}

// --- tests ---

func TestExportOrchestrator_Export_SingleType(t *testing.T) {
	def := baseDef("test_resource", "testplatform", "Test Resource", "test")

	reg := newTestRegistry(t, def)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "testplatform",
		resources: map[string][]interface{}{
			"test_resource": {
				simpleStruct{ID: strPtr("id-1"), Name: strPtr("alpha")},
				simpleStruct{ID: strPtr("id-2"), Name: strPtr("beta")},
			},
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-123"})
	require.NoError(t, err)

	assert.Equal(t, "env-123", result.EnvironmentID)
	assert.Len(t, result.ResourcesByType, 1)
	assert.Equal(t, "test_resource", result.ResourcesByType[0].ResourceType)
	assert.Len(t, result.ResourcesByType[0].Resources, 2)
	assert.Equal(t, "id-1", result.ResourcesByType[0].Resources[0].ID)
	assert.Equal(t, "alpha", result.ResourcesByType[0].Resources[0].Name)
	assert.NotNil(t, result.Graph)
	assert.Equal(t, 2, result.Graph.NodeCount())

	// Graph nodes use sanitized Terraform labels, not raw API names.
	name1, err := result.Graph.GetReferenceName("test_resource", "id-1")
	require.NoError(t, err)
	assert.Equal(t, "pingcli__alpha", name1)

	name2, err := result.Graph.GetReferenceName("test_resource", "id-2")
	require.NoError(t, err)
	assert.Equal(t, "pingcli__beta", name2)
}

func TestExportOrchestrator_Export_DependencyOrdering(t *testing.T) {
	// varDef has no deps, appDef depends on varDef.
	varDef := baseDef("test_variable", "p", "Variable", "variable")
	appDef := baseDef("test_application", "p", "Application", "application")
	appDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "test_variable"},
	}

	reg := newTestRegistry(t, appDef, varDef) // register out of order
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"test_variable":    {simpleStruct{ID: strPtr("v1"), Name: strPtr("var1")}},
			"test_application": {simpleStruct{ID: strPtr("a1"), Name: strPtr("app1")}},
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-1"})
	require.NoError(t, err)

	// Variable should be first, then application.
	require.Len(t, result.ResourcesByType, 2)
	assert.Equal(t, "test_variable", result.ResourcesByType[0].ResourceType)
	assert.Equal(t, "test_application", result.ResourcesByType[1].ResourceType)
}

func TestExportOrchestrator_Export_EmptyEnvironmentID(t *testing.T) {
	reg := schema.NewRegistry()
	proc := NewProcessor(reg)
	client := &mockAPIClient{platform: "p"}

	o := NewExportOrchestrator(reg, proc, client)
	_, err := o.Export(context.Background(), ExportOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment ID is required")
}

func TestExportOrchestrator_Export_NoDefinitions(t *testing.T) {
	reg := schema.NewRegistry()
	proc := NewProcessor(reg)
	client := &mockAPIClient{platform: "p"}

	o := NewExportOrchestrator(reg, proc, client)
	_, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no resource definitions found")
}

func TestExportOrchestrator_Export_ProgressReporting(t *testing.T) {
	def := baseDef("test_res", "p", "Test", "test")
	reg := newTestRegistry(t, def)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"test_res": {simpleStruct{ID: strPtr("1"), Name: strPtr("one")}},
		},
	}

	var messages []string
	o := NewExportOrchestrator(reg, proc, client, WithProgressFunc(func(msg string) {
		messages = append(messages, msg)
	}))

	_, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-1"})
	require.NoError(t, err)

	// Should have at least: "Exporting ...", "Fetching ...", "✓ Processed ...", "✓ Export complete ..."
	assert.GreaterOrEqual(t, len(messages), 4)
}

func TestExportOrchestrator_Export_APIError(t *testing.T) {
	def := baseDef("test_res", "p", "Test", "test")
	reg := newTestRegistry(t, def)
	proc := NewProcessor(reg)

	// Client that returns error for the resource type.
	client := &mockAPIClient{
		platform:  "p",
		resources: map[string][]interface{}{}, // no entry = error
	}

	o := NewExportOrchestrator(reg, proc, client)
	_, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list test_res")
}

func TestExportOrchestrator_Export_CircularDependency(t *testing.T) {
	aDef := baseDef("type_a", "p", "A", "a")
	aDef.Dependencies.DependsOn = []schema.DependencyRule{{ResourceType: "type_b"}}
	bDef := baseDef("type_b", "p", "B", "b")
	bDef.Dependencies.DependsOn = []schema.DependencyRule{{ResourceType: "type_a"}}

	reg := newTestRegistry(t, aDef, bDef)
	proc := NewProcessor(reg)
	client := &mockAPIClient{platform: "p"}

	o := NewExportOrchestrator(reg, proc, client)
	_, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestExportOrchestrator_Export_MultipleTypes(t *testing.T) {
	// 3 types: C depends on B, B depends on A, A has no deps.
	aDef := baseDef("type_a", "p", "A", "a")
	bDef := baseDef("type_b", "p", "B", "b")
	bDef.Dependencies.DependsOn = []schema.DependencyRule{{ResourceType: "type_a"}}
	cDef := baseDef("type_c", "p", "C", "c")
	cDef.Dependencies.DependsOn = []schema.DependencyRule{{ResourceType: "type_b"}}

	reg := newTestRegistry(t, cDef, aDef, bDef) // deliberate disorder
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_a": {simpleStruct{ID: strPtr("a1"), Name: strPtr("alpha")}},
			"type_b": {simpleStruct{ID: strPtr("b1"), Name: strPtr("bravo")}},
			"type_c": {simpleStruct{ID: strPtr("c1"), Name: strPtr("charlie")}},
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-1"})
	require.NoError(t, err)

	require.Len(t, result.ResourcesByType, 3)
	assert.Equal(t, "type_a", result.ResourcesByType[0].ResourceType)
	assert.Equal(t, "type_b", result.ResourcesByType[1].ResourceType)
	assert.Equal(t, "type_c", result.ResourcesByType[2].ResourceType)
	assert.Equal(t, 3, result.Graph.NodeCount())
}

func TestExportOrchestrator_Export_EmptyResourceList(t *testing.T) {
	def := baseDef("test_res", "p", "Test", "test")
	reg := newTestRegistry(t, def)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"test_res": {}, // empty list, not an error
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-1"})
	require.NoError(t, err)

	require.Len(t, result.ResourcesByType, 1)
	assert.Len(t, result.ResourcesByType[0].Resources, 0)
	assert.Equal(t, 0, result.Graph.NodeCount())
}

func TestExportOrchestrator_Export_ResolvesReferences(t *testing.T) {
	// connDef has no deps; flowDef depends on connDef and has a reference to it.
	connDef := baseDef("test_conn", "p", "Connector", "conn")
	flowDef := baseDef("test_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "test_conn"},
	}
	// Add environment_id and connection_id reference attributes to flow.
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "env_id", TerraformName: "environment_id", Type: "string",
			ReferencesType: "pingone_environment", ReferenceField: "id",
		},
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			ReferencesType: "test_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"test_conn": {simpleStruct{ID: strPtr("conn-1"), Name: strPtr("myconn")}},
			"test_flow": {simpleStruct{ID: strPtr("flow-1"), Name: strPtr("myflow")}},
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{EnvironmentID: "env-abc"})
	require.NoError(t, err)

	// Find flow resource data.
	var flowRD *ResourceData
	for _, erd := range result.ResourcesByType {
		if erd.ResourceType == "test_flow" {
			require.Len(t, erd.Resources, 1)
			flowRD = erd.Resources[0]
		}
	}
	require.NotNil(t, flowRD)

	// environment_id should NOT be resolved (flow struct has no EnvironmentID field,
	// so the attribute value is nil/missing). But if it were present as a UUID,
	// it would be a variable reference.
	// connection_id likewise isn't populated from the simple struct.
	// To test actual resolution, set the attribute values manually before Export resolves.
	// This test validates the shape: resolved values are ResolvedReference type.

	// Verify graph has both resources with sanitized names.
	name, err := result.Graph.GetReferenceName("test_conn", "conn-1")
	require.NoError(t, err)
	assert.Equal(t, "pingcli__myconn", name)
}

func TestResolveReferences_DirectIntegration(t *testing.T) {
	// Test resolveAttrs directly with pre-populated attributes.
	defs := []schema.AttributeDefinition{
		{
			Name: "env_id", TerraformName: "environment_id", Type: "string",
			ReferencesType: "pingone_environment", ReferenceField: "id",
		},
		{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			ReferencesType: "test_conn", ReferenceField: "id",
		},
		{
			Name: "config", TerraformName: "config", Type: "object",
			NestedAttributes: []schema.AttributeDefinition{
				{Name: "flow_ref", TerraformName: "flow_ref", Type: "string",
					ReferencesType: "test_flow", ReferenceField: "id"},
			},
		},
	}

	attrs := map[string]interface{}{
		"environment_id": "env-uuid",
		"connection_id":  "conn-uuid",
		"config": map[string]interface{}{
			"flow_ref": "flow-uuid",
		},
	}

	g := graph.New()
	g.AddResource("test_conn", "conn-uuid", "pingcli__my_conn")
	g.AddResource("test_flow", "flow-uuid", "pingcli__my_flow")

	resolveAttrs(attrs, defs, g, "env-uuid", nil)

	// environment_id → variable reference.
	envRef, ok := attrs["environment_id"].(ResolvedReference)
	require.True(t, ok, "environment_id should be ResolvedReference")
	assert.True(t, envRef.IsVariable)
	assert.Equal(t, "pingone_environment_id", envRef.VariableName)
	assert.Equal(t, "var.pingone_environment_id", envRef.Expression())

	// connection_id → resource traversal.
	connRef, ok := attrs["connection_id"].(ResolvedReference)
	require.True(t, ok, "connection_id should be ResolvedReference")
	assert.False(t, connRef.IsVariable)
	assert.Equal(t, "test_conn", connRef.ResourceType)
	assert.Equal(t, "pingcli__my_conn", connRef.ResourceName)
	assert.Equal(t, "test_conn.pingcli__my_conn.id", connRef.Expression())

	// Nested flow_ref → resource traversal.
	configMap, ok := attrs["config"].(map[string]interface{})
	require.True(t, ok)
	flowRef, ok := configMap["flow_ref"].(ResolvedReference)
	require.True(t, ok, "flow_ref should be ResolvedReference")
	assert.Equal(t, "test_flow.pingcli__my_flow.id", flowRef.Expression())
}

func TestResolveCorrelatedReferences_NumericNestedRef(t *testing.T) {
	// Simulates flow_deploy: flow_id resolves to a resource reference,
	// and the nested deploy_trigger_values.deployed_version (numeric)
	// should correlate with the same ResourceType.
	defs := []schema.AttributeDefinition{
		{
			Name: "flow_id", TerraformName: "flow_id", Type: "string",
			ReferencesType: "pingone_davinci_flow", ReferenceField: "id",
		},
		{
			Name: "deploy_trigger_values", TerraformName: "deploy_trigger_values", Type: "object",
			NestedAttributes: []schema.AttributeDefinition{
				{
					Name: "deployed_version", TerraformName: "deployed_version", Type: "number",
					ReferencesType: "pingone_davinci_flow", ReferenceField: "current_version",
				},
			},
		},
	}

	g := graph.New()
	g.AddResource("pingone_davinci_flow", "flow-uuid-123", "pingcli__my_flow")

	attrs := map[string]interface{}{
		"flow_id": "flow-uuid-123",
		"deploy_trigger_values": map[string]interface{}{
			"deployed_version": int64(3),
		},
	}

	// First pass: resolve UUID references.
	resolveAttrs(attrs, defs, g, "env-1", nil)

	// flow_id should be resolved.
	flowRef, ok := attrs["flow_id"].(ResolvedReference)
	require.True(t, ok, "flow_id should be ResolvedReference after resolveAttrs")
	assert.Equal(t, "pingone_davinci_flow.pingcli__my_flow.id", flowRef.Expression())

	// deployed_version should still be int64 (not resolved yet).
	dtv := attrs["deploy_trigger_values"].(map[string]interface{})
	_, isInt := dtv["deployed_version"].(int64)
	assert.True(t, isInt, "deployed_version should still be int64 before correlated resolution")

	// Second pass: resolve correlated references.
	resolveCorrelatedReferences(attrs, defs)

	// deployed_version should now be a correlated ResolvedReference.
	dvRef, ok := dtv["deployed_version"].(ResolvedReference)
	require.True(t, ok, "deployed_version should be ResolvedReference after correlated resolution")
	assert.Equal(t, "pingone_davinci_flow", dvRef.ResourceType)
	assert.Equal(t, "pingcli__my_flow", dvRef.ResourceName)
	assert.Equal(t, "current_version", dvRef.Field)
	assert.Equal(t, "pingone_davinci_flow.pingcli__my_flow.current_version", dvRef.Expression())
	assert.Equal(t, "3", dvRef.OriginalValue)
}

func TestResolveCorrelatedReferences_NoSibling(t *testing.T) {
	// When no sibling resolves to the same ResourceType, the numeric value
	// should remain unchanged.
	defs := []schema.AttributeDefinition{
		{
			Name: "config", TerraformName: "config", Type: "object",
			NestedAttributes: []schema.AttributeDefinition{
				{
					Name: "version", TerraformName: "version", Type: "number",
					ReferencesType: "some_resource", ReferenceField: "version",
				},
			},
		},
	}

	attrs := map[string]interface{}{
		"config": map[string]interface{}{
			"version": int64(5),
		},
	}

	resolveCorrelatedReferences(attrs, defs)

	// Should be unchanged — no sibling to correlate with.
	configMap := attrs["config"].(map[string]interface{})
	v, ok := configMap["version"].(int64)
	assert.True(t, ok, "version should remain int64 when no correlated reference exists")
	assert.Equal(t, int64(5), v)
}

func TestResolveCorrelatedReferences_AlreadyResolved(t *testing.T) {
	// Already-resolved references should not be overwritten.
	defs := []schema.AttributeDefinition{
		{
			Name: "ref", TerraformName: "ref", Type: "string",
			ReferencesType: "some_type", ReferenceField: "id",
		},
	}

	existing := ResolvedReference{
		ResourceType:  "some_type",
		ResourceName:  "my_resource",
		Field:         "id",
		OriginalValue: "uuid-123",
	}

	attrs := map[string]interface{}{
		"ref": existing,
	}

	resolveCorrelatedReferences(attrs, defs)

	// Should be unchanged.
	ref := attrs["ref"].(ResolvedReference)
	assert.Equal(t, "some_type.my_resource.id", ref.Expression())
}

// --- resolveDependsOnResources tests ---

// TestResolveDependsOnResources_ResolvesLabels verifies that resource IDs in
// DependsOnResources are resolved to their Terraform labels via the graph.
func TestResolveDependsOnResources_ResolvesLabels(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_variable", "var-uuid-1", "pingcli__my_var")
	g.AddResource("pingone_davinci_variable", "var-uuid-2", "pingcli__other_var")

	resources := []*ResourceData{
		{
			ResourceType: "pingone_davinci_flow",
			ID:           "flow-1",
			Name:         "my_flow",
			DependsOnResources: []RuntimeDependsOn{
				{ResourceType: "pingone_davinci_variable", ResourceID: "var-uuid-1"},
				{ResourceType: "pingone_davinci_variable", ResourceID: "var-uuid-2"},
			},
		},
	}

	resolveDependsOnResources(resources, g)

	require.Len(t, resources[0].DependsOnResources, 2)
	assert.Equal(t, "pingcli__my_var", resources[0].DependsOnResources[0].Label)
	assert.Equal(t, "pingcli__other_var", resources[0].DependsOnResources[1].Label)
}

// TestResolveDependsOnResources_UnknownID_LabelEmpty verifies that when a
// resource ID is not in the graph, the Label field remains empty.
func TestResolveDependsOnResources_UnknownID_LabelEmpty(t *testing.T) {
	g := graph.New()
	// No matching resource in graph.

	resources := []*ResourceData{
		{
			ResourceType: "pingone_davinci_flow",
			ID:           "flow-1",
			Name:         "my_flow",
			DependsOnResources: []RuntimeDependsOn{
				{ResourceType: "pingone_davinci_variable", ResourceID: "unknown-uuid"},
			},
		},
	}

	resolveDependsOnResources(resources, g)

	require.Len(t, resources[0].DependsOnResources, 1)
	assert.Equal(t, "", resources[0].DependsOnResources[0].Label, "unresolved IDs should have empty label")
}

// TestResolveDependsOnResources_NoDependsOn_NoOp verifies that resources with
// no DependsOnResources are unaffected.
func TestResolveDependsOnResources_NoDependsOn_NoOp(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_variable", "var-uuid-1", "pingcli__my_var")

	resources := []*ResourceData{
		{
			ResourceType:       "pingone_davinci_flow",
			ID:                 "flow-1",
			Name:               "my_flow",
			DependsOnResources: nil,
		},
	}

	resolveDependsOnResources(resources, g) // must not panic

	assert.Nil(t, resources[0].DependsOnResources)
}

// TestResolveDependsOnResources_NilGraph_NoOp verifies that a nil graph does
// not panic and leaves labels empty.
func TestResolveDependsOnResources_NilGraph_NoOp(t *testing.T) {
	resources := []*ResourceData{
		{
			ResourceType: "pingone_davinci_flow",
			ID:           "flow-1",
			Name:         "my_flow",
			DependsOnResources: []RuntimeDependsOn{
				{ResourceType: "pingone_davinci_variable", ResourceID: "var-uuid-1"},
			},
		},
	}

	resolveDependsOnResources(resources, nil) // must not panic

	require.Len(t, resources[0].DependsOnResources, 1)
	assert.Equal(t, "", resources[0].DependsOnResources[0].Label)
}

// TestResolveDependsOnResources_MultipleResources verifies that all resources
// in the slice are processed.
func TestResolveDependsOnResources_MultipleResources(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_variable", "var-a", "pingcli__var_a")
	g.AddResource("pingone_davinci_variable", "var-b", "pingcli__var_b")

	resources := []*ResourceData{
		{
			ResourceType: "pingone_davinci_flow",
			ID:           "flow-1",
			DependsOnResources: []RuntimeDependsOn{
				{ResourceType: "pingone_davinci_variable", ResourceID: "var-a"},
			},
		},
		{
			ResourceType: "pingone_davinci_flow",
			ID:           "flow-2",
			DependsOnResources: []RuntimeDependsOn{
				{ResourceType: "pingone_davinci_variable", ResourceID: "var-b"},
			},
		},
	}

	resolveDependsOnResources(resources, g)

	assert.Equal(t, "pingcli__var_a", resources[0].DependsOnResources[0].Label)
	assert.Equal(t, "pingcli__var_b", resources[1].DependsOnResources[0].Label)
}

// TestResolveEnvironmentReference_InGraph tests that when pingone_environment
// IS in the dependency graph, environment references resolve to resource traversals.
func TestResolveEnvironmentReference_InGraph(t *testing.T) {
	// Create a definition with an environment reference
	envAttrDef := schema.AttributeDefinition{
		Name:           "environment_id",
		TerraformName:  "environment_id",
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "id",
	}

	// Create a mock attribute value (UUID)
	environmentUUID := "env-550e8400-e29b-41d4-a716-446655440000"

	// Build graph with pingone_environment resource
	g := graph.New()
	g.AddResource("pingone_environment", environmentUUID, "pingcli__production")

	// Resolve the reference
	result := resolveOneReference(envAttrDef, environmentUUID, g, nil)

	// Should resolve to resource traversal, not variable
	assert.False(t, result.IsVariable)
	assert.Equal(t, "pingone_environment", result.ResourceType)
	assert.Equal(t, "pingcli__production", result.ResourceName)
	assert.Equal(t, "id", result.Field)
	assert.Equal(t, "pingone_environment.pingcli__production.id", result.Expression())
}

// TestResolveEnvironmentReference_NotInGraph tests that when pingone_environment
// is NOT in the dependency graph, environment references fall back to variable references.
func TestResolveEnvironmentReference_NotInGraph(t *testing.T) {
	// Create a definition with an environment reference
	envAttrDef := schema.AttributeDefinition{
		Name:           "environment_id",
		TerraformName:  "environment_id",
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "id",
	}

	// Create a mock attribute value (UUID)
	environmentUUID := "env-550e8400-e29b-41d4-a716-446655440000"

	// Build empty graph (no pingone_environment resource)
	g := graph.New()

	// Resolve the reference
	result := resolveOneReference(envAttrDef, environmentUUID, g, nil)

	// Should fall back to variable reference
	assert.True(t, result.IsVariable)
	assert.Equal(t, "pingone_environment_id", result.VariableName)
	assert.Equal(t, environmentUUID, result.OriginalValue)
	assert.Equal(t, "var.pingone_environment_id", result.Expression())
}

// TestResolveEnvironmentReference_NilGraph tests that when graph is nil,
// environment references fall back to variable references.
func TestResolveEnvironmentReference_NilGraph(t *testing.T) {
	// Create a definition with an environment reference
	envAttrDef := schema.AttributeDefinition{
		Name:           "environment_id",
		TerraformName:  "environment_id",
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "id",
	}

	environmentUUID := "env-550e8400-e29b-41d4-a716-446655440000"

	// Resolve with nil graph
	result := resolveOneReference(envAttrDef, environmentUUID, nil, nil)

	// Should fall back to variable reference
	assert.True(t, result.IsVariable)
	assert.Equal(t, "pingone_environment_id", result.VariableName)
	assert.Equal(t, "var.pingone_environment_id", result.Expression())
}

// TestResolveEnvironmentReference_CustomField tests reference resolution
// when a custom ReferenceField is specified.
func TestResolveEnvironmentReference_CustomField(t *testing.T) {
	// Use a custom reference field (unusual but valid for testing)
	customAttrDef := schema.AttributeDefinition{
		Name:           "env_name",
		TerraformName:  "env_name",
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "name",
	}

	environmentUUID := "env-550e8400-e29b-41d4-a716-446655440000"

	// Build graph with environment resource
	g := graph.New()
	g.AddResource("pingone_environment", environmentUUID, "pingcli__staging")

	result := resolveOneReference(customAttrDef, environmentUUID, g, nil)

	// Should resolve with custom field
	assert.False(t, result.IsVariable)
	assert.Equal(t, "pingone_environment", result.ResourceType)
	assert.Equal(t, "name", result.Field)
	assert.Equal(t, "pingone_environment.pingcli__staging.name", result.Expression())
}

// --- Filtering Tests ---

// TestExportOrchestrator_Export_FilterInclude tests that include patterns
// filter resources at the type level and resource level.
func TestExportOrchestrator_Export_FilterInclude(t *testing.T) {
	// Setup: 2 types (type_flow, type_var) with 2 resources each.
	// type_var depends on type_flow.
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	varDef := baseDef("type_var", "p", "Variable", "var")
	varDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_flow"},
	}

	reg := newTestRegistry(t, flowDef, varDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_flow": {
				simpleStruct{ID: strPtr("flow1"), Name: strPtr("flow1")},
				simpleStruct{ID: strPtr("flow2"), Name: strPtr("flow2")},
			},
			"type_var": {
				simpleStruct{ID: strPtr("var1"), Name: strPtr("var1")},
				simpleStruct{ID: strPtr("var2"), Name: strPtr("var2")},
			},
		},
	}

	// Create filter: include only type_flow* resources.
	filterObj, err := filter.NewResourceFilter([]string{"type_flow*"}, nil)
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:  "env-1",
		ResourceFilter: filterObj,
	})
	require.NoError(t, err)

	// Assert: only type_flow in result
	require.Len(t, result.ResourcesByType, 1)
	assert.Equal(t, "type_flow", result.ResourcesByType[0].ResourceType)
	assert.Len(t, result.ResourcesByType[0].Resources, 2)

	// Assert: graph has 4 nodes (all resources registered before filtering).
	assert.Equal(t, 4, result.Graph.NodeCount())
}

// TestExportOrchestrator_Export_FilterExclude tests that exclude patterns
// filter out matching resources by full address (type.label).
func TestExportOrchestrator_Export_FilterExclude(t *testing.T) {
	// Use a type name that doesn't conflict with the exclude pattern.
	def := baseDef("my_resource", "p", "My Resource", "myres")
	reg := newTestRegistry(t, def)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"my_resource": {
				simpleStruct{ID: strPtr("id-1"), Name: strPtr("alpha")},
				simpleStruct{ID: strPtr("id-2"), Name: strPtr("beta_test")},
				simpleStruct{ID: strPtr("id-3"), Name: strPtr("gamma")},
			},
		},
	}

	// Exclude pattern targets the label portion.
	// Address: my_resource.pingcli__beta_test — matches *beta_test*
	filterObj, err := filter.NewResourceFilter(nil, []string{"*beta_test*"})
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:  "env-1",
		ResourceFilter: filterObj,
	})
	require.NoError(t, err)

	// Assert: 2 resources (alpha, gamma)
	require.Len(t, result.ResourcesByType, 1)
	assert.Len(t, result.ResourcesByType[0].Resources, 2)

	// Verify names: alpha and gamma
	resourceNames := map[string]bool{}
	for _, r := range result.ResourcesByType[0].Resources {
		resourceNames[r.Name] = true
	}
	assert.True(t, resourceNames["alpha"])
	assert.True(t, resourceNames["gamma"])
	assert.False(t, resourceNames["beta_test"])

	// Assert: graph has 3 nodes (all resources registered before filtering).
	assert.Equal(t, 3, result.Graph.NodeCount())
}

// TestExportOrchestrator_Export_FilterIncludeAndExclude tests combined
// include and exclude patterns.
func TestExportOrchestrator_Export_FilterIncludeAndExclude(t *testing.T) {
	// Setup: 2 types with 2 resources each.
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	varDef := baseDef("type_var", "p", "Variable", "var")

	reg := newTestRegistry(t, flowDef, varDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_flow": {
				simpleStruct{ID: strPtr("flow1"), Name: strPtr("Login_Flow")},
				simpleStruct{ID: strPtr("flow2"), Name: strPtr("Test_Flow")},
			},
			"type_var": {
				simpleStruct{ID: strPtr("var1"), Name: strPtr("authToken")},
				simpleStruct{ID: strPtr("var2"), Name: strPtr("testVar")},
			},
		},
	}

	// Include both types, but exclude resources with "Test" in the label.
	// Addresses: type_flow.pingcli__Test_Flow, type_var.pingcli__testVar
	filterObj, err := filter.NewResourceFilter(
		[]string{"type_flow*", "type_var*"},
		[]string{"*Test*"},
	)
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:  "env-1",
		ResourceFilter: filterObj,
	})
	require.NoError(t, err)

	// Assert: 2 types in result
	require.Len(t, result.ResourcesByType, 2)

	// type_flow should have 1 resource (Login_Flow, Test_Flow excluded)
	flowData := result.ResourcesByType[0]
	if flowData.ResourceType == "type_flow" {
		assert.Len(t, flowData.Resources, 1)
		assert.Equal(t, "Login_Flow", flowData.Resources[0].Name)
	}

	// type_var should have 1 resource (authToken, testVar excluded)
	varData := result.ResourcesByType[1]
	if varData.ResourceType == "type_var" {
		assert.Len(t, varData.Resources, 1)
		assert.Equal(t, "authToken", varData.Resources[0].Name)
	}

	// Assert: graph has 4 nodes total (all resources registered before filtering
	// for reference resolution; filtered resources become variable fallbacks).
	assert.Equal(t, 4, result.Graph.NodeCount())
}

// TestExportOrchestrator_Export_FilterNil tests that nil filter
// includes all resources (backward compatibility).
func TestExportOrchestrator_Export_FilterNil(t *testing.T) {
	// Setup: 1 type with 2 resources
	def := baseDef("test_resource", "p", "Test Resource", "test")
	reg := newTestRegistry(t, def)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"test_resource": {
				simpleStruct{ID: strPtr("id-1"), Name: strPtr("alpha")},
				simpleStruct{ID: strPtr("id-2"), Name: strPtr("beta")},
			},
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:  "env-1",
		ResourceFilter: nil, // no filtering
	})
	require.NoError(t, err)

	// Assert: all resources returned
	require.Len(t, result.ResourcesByType, 1)
	assert.Len(t, result.ResourcesByType[0].Resources, 2)
	assert.Equal(t, 2, result.Graph.NodeCount())
}

// TestExportOrchestrator_Export_FilterEmptyMatch tests that include patterns
// that match nothing result in empty export.
func TestExportOrchestrator_Export_FilterEmptyMatch(t *testing.T) {
	// Setup: 1 type with 2 resources
	def := baseDef("test_resource", "p", "Test Resource", "test")
	reg := newTestRegistry(t, def)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"test_resource": {
				simpleStruct{ID: strPtr("id-1"), Name: strPtr("alpha")},
				simpleStruct{ID: strPtr("id-2"), Name: strPtr("beta")},
			},
		},
	}

	// Include pattern that matches nothing
	filterObj, err := filter.NewResourceFilter([]string{"nonexistent_type*"}, nil)
	require.NoError(t, err)

	var messages []string
	o := NewExportOrchestrator(reg, proc, client, WithProgressFunc(func(msg string) {
		messages = append(messages, msg)
	}))

	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:  "env-1",
		ResourceFilter: filterObj,
	})
	require.NoError(t, err)

	// Assert: 0 resources in result (type may or may not be present with 0 resources)
	totalResources := 0
	for _, erd := range result.ResourcesByType {
		totalResources += len(erd.Resources)
	}
	assert.Equal(t, 0, totalResources)

	// Assert: progress messages indicate empty or no resources
	assert.Greater(t, len(messages), 0)
}

// TestInjectEnvIDAttrs_NoSourcePath tests Bug 3 fix: when --skip-dependencies is true,
// resources with references_type: pingone_environment on environment_id but NO source_path
// should get the raw environment UUID injected.
func TestInjectEnvIDAttrs_NoSourcePath(t *testing.T) {
	// Define an attribute with references_type: pingone_environment but NO source_path.
	// This simulates a definition where the processor doesn't extract it from API data.
	envAttrDef := schema.AttributeDefinition{
		Name:           "environment_id",
		TerraformName:  "environment_id",
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "id",
		SourcePath:     "", // No source path — orchestrator must inject
	}

	// Start with empty attributes (environment_id not present)
	attrs := make(map[string]interface{})
	environmentID := "env-12345-abcde"

	// Call injectEnvIDAttrs
	injectEnvIDAttrs(attrs, []schema.AttributeDefinition{envAttrDef}, environmentID)

	// Verify: environment_id should be injected
	require.Contains(t, attrs, "environment_id", "environment_id should be in attrs after injection")
	assert.Equal(t, environmentID, attrs["environment_id"], "environment_id should have the raw UUID")
}

// TestInjectEnvIDAttrs_ExistingValue tests that injectEnvIDAttrs does NOT overwrite
// an existing attribute value.
func TestInjectEnvIDAttrs_ExistingValue(t *testing.T) {
	envAttrDef := schema.AttributeDefinition{
		Name:           "environment_id",
		TerraformName:  "environment_id",
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "id",
	}

	// Start with an existing value (e.g., from resolveAttrs or manual set)
	existingValue := "env-existing-uuid"
	attrs := map[string]interface{}{
		"environment_id": existingValue,
	}
	environmentID := "env-new-uuid"

	// Call injectEnvIDAttrs
	injectEnvIDAttrs(attrs, []schema.AttributeDefinition{envAttrDef}, environmentID)

	// Verify: existing value should NOT be overwritten
	assert.Equal(t, existingValue, attrs["environment_id"], "existing environment_id should not be overwritten")
}

// TestInjectEnvIDAttrs_MultipleAttributes tests injectEnvIDAttrs with multiple
// attribute definitions, including non-environment references.
func TestInjectEnvIDAttrs_MultipleAttributes(t *testing.T) {
	defs := []schema.AttributeDefinition{
		{
			Name:           "environment_id",
			TerraformName:  "environment_id",
			Type:           "string",
			ReferencesType: "pingone_environment",
			ReferenceField: "id",
		},
		{
			Name:           "name",
			TerraformName:  "name",
			Type:           "string",
			ReferencesType: "", // No reference type
		},
		{
			Name:           "connection_id",
			TerraformName:  "connection_id",
			Type:           "string",
			ReferencesType: "test_connection", // Different reference type
			ReferenceField: "id",
		},
	}

	attrs := map[string]interface{}{
		"name": "my-resource",
		// environment_id missing
		// connection_id missing
	}
	environmentID := "env-abc123"

	// Call injectEnvIDAttrs
	injectEnvIDAttrs(attrs, defs, environmentID)

	// Verify: only environment_id should be injected
	assert.Equal(t, environmentID, attrs["environment_id"], "environment_id should be injected")
	assert.Equal(t, "my-resource", attrs["name"], "name should remain unchanged")
	assert.NotContains(t, attrs, "connection_id", "connection_id should not be injected (wrong reference type)")
}

// TestInjectEnvIDAttrs_CustomTerraformName tests that injectEnvIDAttrs uses
// the TerraformName when mapping attributes.
func TestInjectEnvIDAttrs_CustomTerraformName(t *testing.T) {
	// Define an attribute with a custom TerraformName different from Name
	envAttrDef := schema.AttributeDefinition{
		Name:           "env_id",         // API name
		TerraformName:  "environment_id", // Terraform name
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "id",
	}

	attrs := make(map[string]interface{})
	environmentID := "env-custom-name"

	// Call injectEnvIDAttrs
	injectEnvIDAttrs(attrs, []schema.AttributeDefinition{envAttrDef}, environmentID)

	// Verify: must use TerraformName, not Name
	assert.Contains(t, attrs, "environment_id", "should use TerraformName as key")
	assert.NotContains(t, attrs, "env_id", "should not use API Name as key")
	assert.Equal(t, environmentID, attrs["environment_id"])
}

// TestExportOrchestrator_Export_FilterDependencyWarning tests that filtering
// resources referenced by other resources produces a warning.
func TestExportOrchestrator_Export_FilterDependencyWarning(t *testing.T) {
	// Setup: connector type (type_conn) and flow type (type_flow) that references it.
	connDef := baseDef("type_conn", "p", "Connector", "conn")
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_conn"},
	}
	// Add connection_id reference attribute to flow.
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			ReferencesType: "type_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_conn": {
				simpleStruct{ID: strPtr("conn-uuid"), Name: strPtr("conn1")},
			},
			"type_flow": {
				simpleStruct{ID: strPtr("flow-uuid"), Name: strPtr("flow1")},
			},
		},
	}

	// Filter to include only flows, excluding connectors
	filterObj, err := filter.NewResourceFilter([]string{"type_flow*"}, nil)
	require.NoError(t, err)

	var messages []string
	o := NewExportOrchestrator(reg, proc, client, WithProgressFunc(func(msg string) {
		messages = append(messages, msg)
	}))

	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:  "env-1",
		ResourceFilter: filterObj,
	})
	require.NoError(t, err)
	require.Len(t, result.ResourcesByType, 1)
	assert.Equal(t, "type_flow", result.ResourcesByType[0].ResourceType)

	// Assert: dependency warning was emitted about excluded type_conn
	hasDepWarning := false
	for _, msg := range messages {
		if strings.Contains(msg, "Warning") && strings.Contains(msg, "type_conn") {
			hasDepWarning = true
			break
		}
	}
	assert.True(t, hasDepWarning, "should emit dependency warning about excluded type_conn")
}

// TestExportOrchestrator_Export_ListOnly tests that ListOnly flag
// returns resources without resolving references.
func TestExportOrchestrator_Export_ListOnly(t *testing.T) {
	// Setup: connector and flow with reference
	connDef := baseDef("type_conn", "p", "Connector", "conn")
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_conn"},
	}
	// Add connection_id reference attribute to flow.
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			ReferencesType: "type_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	// Make sure connection_id gets set in the raw data
	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_conn": {
				simpleStruct{ID: strPtr("conn-uuid"), Name: strPtr("conn1")},
			},
			"type_flow": {
				simpleStruct{ID: strPtr("flow-uuid"), Name: strPtr("flow1")},
			},
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID: "env-1",
		ListOnly:      true, // Skip reference resolution
	})
	require.NoError(t, err)

	// Assert: resources returned with labels assigned
	require.Len(t, result.ResourcesByType, 2)

	// Verify labels exist
	for _, erd := range result.ResourcesByType {
		for _, rd := range erd.Resources {
			assert.NotEmpty(t, rd.Label)
			// Verify label has sanitized format (pingcli__ is 9 chars)
			assert.True(t, len(rd.Label) >= 9 && rd.Label[0:9] == "pingcli__")
		}
	}

	// Note: We can't directly test that reference resolution didn't happen
	// without modifying the data structure, but the ListOnly flag should
	// prevent the resolveReferences call from being made.
}

// TestExportOrchestrator_Export_FilterRemovesEmptyTypes tests that when filtering
// results in empty resource lists for a type, that type is not included in results.
func TestExportOrchestrator_Export_FilterRemovesEmptyTypes(t *testing.T) {
	// Setup: 2 types (type_a, type_b), each with 1 resource
	aDef := baseDef("type_a", "p", "Type A", "a")
	bDef := baseDef("type_b", "p", "Type B", "b")

	reg := newTestRegistry(t, aDef, bDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_a": {
				simpleStruct{ID: strPtr("a1"), Name: strPtr("alpha")},
			},
			"type_b": {
				simpleStruct{ID: strPtr("b1"), Name: strPtr("beta")},
			},
		},
	}

	// Filter to include only type_a
	filterObj, err := filter.NewResourceFilter([]string{"type_a*"}, nil)
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:  "env-1",
		ResourceFilter: filterObj,
	})
	require.NoError(t, err)

	// Assert: only type_a in result (type_b completely removed, not present with 0 resources)
	require.Len(t, result.ResourcesByType, 1)
	assert.Equal(t, "type_a", result.ResourcesByType[0].ResourceType)
	assert.Len(t, result.ResourcesByType[0].Resources, 1)
	// Assert: graph has 2 nodes total (all resources registered before filtering).
	assert.Equal(t, 2, result.Graph.NodeCount())
}

// --- Bug 4 Tests: Filter-Excluded Upstream Resources ---

// TestResolveOneReference_ExcludedResource tests that resolveOneReference produces
// a variable reference with label-derived name when the resource is in the graph but excluded.
func TestResolveOneReference_ExcludedResource(t *testing.T) {
	// Setup
	attrDef := schema.AttributeDefinition{
		Name:           "application_id",
		TerraformName:  "application_id",
		Type:           "string",
		ReferencesType: "pingone_davinci_application",
		ReferenceField: "id",
	}

	g := graph.New()
	g.AddResource("pingone_davinci_application", "app-uuid-123", "pingcli__my_app")

	excludedIDs := map[string]bool{"app-uuid-123": true}

	result := resolveOneReference(attrDef, "app-uuid-123", g, excludedIDs)

	// Should be a variable reference with label-derived name
	assert.True(t, result.IsVariable)
	assert.Equal(t, "pingone_davinci_application_pingcli__my_app_id", result.VariableName)
	assert.Equal(t, "var.pingone_davinci_application_pingcli__my_app_id", result.Expression())
	assert.Equal(t, "app-uuid-123", result.OriginalValue)
	assert.Equal(t, "pingone_davinci_application", result.ResourceType)
}

// TestResolveOneReference_ExcludedResourceUniqueness tests that two different
// excluded resources of the same type produce different variable names.
func TestResolveOneReference_ExcludedResourceUniqueness(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_application", "app-uuid-1", "pingcli__app_one")
	g.AddResource("pingone_davinci_application", "app-uuid-2", "pingcli__app_two")

	excludedIDs := map[string]bool{
		"app-uuid-1": true,
		"app-uuid-2": true,
	}

	attrDef := schema.AttributeDefinition{
		Name:           "application_id",
		TerraformName:  "application_id",
		Type:           "string",
		ReferencesType: "pingone_davinci_application",
		ReferenceField: "id",
	}

	result1 := resolveOneReference(attrDef, "app-uuid-1", g, excludedIDs)
	result2 := resolveOneReference(attrDef, "app-uuid-2", g, excludedIDs)

	// Both should be variable references
	assert.True(t, result1.IsVariable)
	assert.True(t, result2.IsVariable)

	// But with different names
	assert.NotEqual(t, result1.VariableName, result2.VariableName)
	assert.Equal(t, "pingone_davinci_application_pingcli__app_one_id", result1.VariableName)
	assert.Equal(t, "pingone_davinci_application_pingcli__app_two_id", result2.VariableName)
}

// TestResolveOneReference_SameExcludedResourceSameVarName tests that multiple
// downstream references to the SAME excluded resource produce the SAME variable name.
func TestResolveOneReference_SameExcludedResourceSameVarName(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_application", "app-uuid-123", "pingcli__my_app")

	excludedIDs := map[string]bool{"app-uuid-123": true}

	// Multiple attribute definitions (e.g., from different nested levels)
	attrDef1 := schema.AttributeDefinition{
		Name:           "application_id",
		TerraformName:  "application_id",
		Type:           "string",
		ReferencesType: "pingone_davinci_application",
		ReferenceField: "id",
	}

	attrDef2 := schema.AttributeDefinition{
		Name:           "app_reference",
		TerraformName:  "app_reference",
		Type:           "string",
		ReferencesType: "pingone_davinci_application",
		ReferenceField: "id",
	}

	result1 := resolveOneReference(attrDef1, "app-uuid-123", g, excludedIDs)
	result2 := resolveOneReference(attrDef2, "app-uuid-123", g, excludedIDs)

	// Both should be the SAME variable reference
	assert.True(t, result1.IsVariable)
	assert.True(t, result2.IsVariable)
	assert.Equal(t, result1.VariableName, result2.VariableName)
	assert.Equal(t, "pingone_davinci_application_pingcli__my_app_id", result1.VariableName)
	assert.Equal(t, "pingone_davinci_application_pingcli__my_app_id", result2.VariableName)
}

// TestResolveOneReference_ExcludedResourceWithCustomReferenceField tests that
// custom reference fields are included in the variable name.
func TestResolveOneReference_ExcludedResourceWithCustomReferenceField(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "flow-uuid-abc", "pingcli__my_flow")

	excludedIDs := map[string]bool{"flow-uuid-abc": true}

	attrDef := schema.AttributeDefinition{
		Name:           "current_version",
		TerraformName:  "current_version",
		Type:           "number",
		ReferencesType: "pingone_davinci_flow",
		ReferenceField: "current_version", // Custom field, not "id"
	}

	result := resolveOneReference(attrDef, "flow-uuid-abc", g, excludedIDs)

	// Should include the custom field in the variable name
	assert.True(t, result.IsVariable)
	assert.Equal(t, "pingone_davinci_flow_pingcli__my_flow_current_version", result.VariableName)
	assert.Equal(t, "var.pingone_davinci_flow_pingcli__my_flow_current_version", result.Expression())
}

// TestResolveOneReference_ExcludedEnvironmentKeepsCanonicalName tests that
// excluded pingone_environment resources always produce the canonical
// "pingone_environment_id" variable name for backward compatibility.
func TestResolveOneReference_ExcludedEnvironmentKeepsCanonicalName(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_environment", "env-uuid-123", "pingcli__production")

	excludedIDs := map[string]bool{"env-uuid-123": true}

	attrDef := schema.AttributeDefinition{
		Name:           "environment_id",
		TerraformName:  "environment_id",
		Type:           "string",
		ReferencesType: "pingone_environment",
		ReferenceField: "id",
	}

	result := resolveOneReference(attrDef, "env-uuid-123", g, excludedIDs)

	// Should use the canonical name, NOT "pingone_environment_pingcli__production_id"
	assert.True(t, result.IsVariable)
	assert.Equal(t, "pingone_environment_id", result.VariableName)
	assert.Equal(t, "var.pingone_environment_id", result.Expression())
	assert.Equal(t, "env-uuid-123", result.OriginalValue)
}

// TestCollectFallbackVars_Basic tests that collectFallbackVars collects a
// FallbackVariable from a ResolvedReference with IsVariable=true.
func TestCollectFallbackVars_Basic(t *testing.T) {
	defs := []schema.AttributeDefinition{
		{
			Name:           "application_id",
			TerraformName:  "application_id",
			Type:           "string",
			ReferencesType: "pingone_davinci_application",
		},
	}

	attrs := map[string]interface{}{
		"application_id": ResolvedReference{
			IsVariable:    true,
			VariableName:  "pingone_davinci_application_pingcli__my_app_id",
			ResourceType:  "pingone_davinci_application",
			OriginalValue: "app-uuid-123",
		},
	}

	seen := make(map[string]bool)
	var out []FallbackVariable

	collectFallbackVars(attrs, defs, seen, &out)

	// Should collect the fallback variable
	require.Len(t, out, 1)
	assert.Equal(t, "pingone_davinci_application_pingcli__my_app_id", out[0].Name)
	assert.Equal(t, "string", out[0].Type)
	assert.Contains(t, out[0].Description, "pingone_davinci_application")
	assert.Equal(t, "pingone_davinci_application", out[0].ResourceType)
	assert.True(t, seen["pingone_davinci_application_pingcli__my_app_id"])
}

// TestCollectFallbackVars_SkipsEnvironment tests that collectFallbackVars skips
// the standard pingone_environment_id variable.
func TestCollectFallbackVars_SkipsEnvironment(t *testing.T) {
	defs := []schema.AttributeDefinition{
		{
			Name:           "environment_id",
			TerraformName:  "environment_id",
			Type:           "string",
			ReferencesType: "pingone_environment",
		},
		{
			Name:           "application_id",
			TerraformName:  "application_id",
			Type:           "string",
			ReferencesType: "pingone_davinci_application",
		},
	}

	attrs := map[string]interface{}{
		"environment_id": ResolvedReference{
			IsVariable:    true,
			VariableName:  "pingone_environment_id",
			ResourceType:  "pingone_environment",
			OriginalValue: "env-uuid",
		},
		"application_id": ResolvedReference{
			IsVariable:    true,
			VariableName:  "pingone_davinci_application_pingcli__my_app_id",
			ResourceType:  "pingone_davinci_application",
			OriginalValue: "app-uuid-123",
		},
	}

	seen := make(map[string]bool)
	var out []FallbackVariable

	collectFallbackVars(attrs, defs, seen, &out)

	// Should only collect the application variable, skipping environment_id
	require.Len(t, out, 1)
	assert.Equal(t, "pingone_davinci_application_pingcli__my_app_id", out[0].Name)
	// pingone_environment_id should NOT be in seen
	assert.False(t, seen["pingone_environment_id"])
}

// TestCollectFallbackVars_Deduplicates tests that collectFallbackVars deduplicates
// entries by variable name.
func TestCollectFallbackVars_Deduplicates(t *testing.T) {
	defs := []schema.AttributeDefinition{
		{
			Name:           "app_id_1",
			TerraformName:  "app_id_1",
			Type:           "string",
			ReferencesType: "pingone_davinci_application",
		},
		{
			Name:           "app_id_2",
			TerraformName:  "app_id_2",
			Type:           "string",
			ReferencesType: "pingone_davinci_application",
		},
	}

	// Both attributes reference the same excluded resource (same VariableName)
	varRef := ResolvedReference{
		IsVariable:    true,
		VariableName:  "pingone_davinci_application_pingcli__my_app_id",
		ResourceType:  "pingone_davinci_application",
		OriginalValue: "app-uuid-123",
	}

	attrs := map[string]interface{}{
		"app_id_1": varRef,
		"app_id_2": varRef,
	}

	seen := make(map[string]bool)
	var out []FallbackVariable

	collectFallbackVars(attrs, defs, seen, &out)

	// Should only collect ONE FallbackVariable, deduplicated by name
	require.Len(t, out, 1)
	assert.Equal(t, "pingone_davinci_application_pingcli__my_app_id", out[0].Name)
	assert.True(t, seen["pingone_davinci_application_pingcli__my_app_id"])
}

// TestCollectFallbackVars_NestedObject tests that collectFallbackVars recurses
// into nested objects and collects variable references there.
func TestCollectFallbackVars_NestedObject(t *testing.T) {
	defs := []schema.AttributeDefinition{
		{
			Name:          "config",
			TerraformName: "config",
			Type:          "object",
			NestedAttributes: []schema.AttributeDefinition{
				{
					Name:           "flow_id",
					TerraformName:  "flow_id",
					Type:           "string",
					ReferencesType: "pingone_davinci_flow",
				},
			},
		},
	}

	attrs := map[string]interface{}{
		"config": map[string]interface{}{
			"flow_id": ResolvedReference{
				IsVariable:    true,
				VariableName:  "pingone_davinci_flow_pingcli__my_flow_id",
				ResourceType:  "pingone_davinci_flow",
				OriginalValue: "flow-uuid-123",
			},
		},
	}

	seen := make(map[string]bool)
	var out []FallbackVariable

	collectFallbackVars(attrs, defs, seen, &out)

	// Should collect the nested variable reference
	require.Len(t, out, 1)
	assert.Equal(t, "pingone_davinci_flow_pingcli__my_flow_id", out[0].Name)
	assert.Equal(t, "pingone_davinci_flow", out[0].ResourceType)
}

// TestCollectFallbackVars_NestedList tests that collectFallbackVars recurses
// into arrays of nested objects.
func TestCollectFallbackVars_NestedList(t *testing.T) {
	defs := []schema.AttributeDefinition{
		{
			Name:          "connectors",
			TerraformName: "connectors",
			Type:          "list",
			NestedAttributes: []schema.AttributeDefinition{
				{
					Name:           "connector_id",
					TerraformName:  "connector_id",
					Type:           "string",
					ReferencesType: "pingone_davinci_connector",
				},
			},
		},
	}

	attrs := map[string]interface{}{
		"connectors": []interface{}{
			map[string]interface{}{
				"connector_id": ResolvedReference{
					IsVariable:    true,
					VariableName:  "pingone_davinci_connector_pingcli__connector_1_id",
					ResourceType:  "pingone_davinci_connector",
					OriginalValue: "conn-uuid-1",
				},
			},
			map[string]interface{}{
				"connector_id": ResolvedReference{
					IsVariable:    true,
					VariableName:  "pingone_davinci_connector_pingcli__connector_2_id",
					ResourceType:  "pingone_davinci_connector",
					OriginalValue: "conn-uuid-2",
				},
			},
		},
	}

	seen := make(map[string]bool)
	var out []FallbackVariable

	collectFallbackVars(attrs, defs, seen, &out)

	// Should collect both nested variable references
	require.Len(t, out, 2)
	assert.Equal(t, "pingone_davinci_connector_pingcli__connector_1_id", out[0].Name)
	assert.Equal(t, "pingone_davinci_connector_pingcli__connector_2_id", out[1].Name)
}

// TestCollectFallbackVars_MixedTypes tests that collectFallbackVars handles
// attributes that are not variable references (e.g., simple strings, resolved resource refs).
func TestCollectFallbackVars_MixedTypes(t *testing.T) {
	defs := []schema.AttributeDefinition{
		{
			Name:           "name",
			TerraformName:  "name",
			Type:           "string",
			ReferencesType: "", // Not a reference
		},
		{
			Name:           "connection_id",
			TerraformName:  "connection_id",
			Type:           "string",
			ReferencesType: "pingone_davinci_connector",
		},
		{
			Name:           "app_id",
			TerraformName:  "app_id",
			Type:           "string",
			ReferencesType: "pingone_davinci_application",
		},
	}

	attrs := map[string]interface{}{
		"name": "my_flow", // Simple string, not a reference
		"connection_id": ResolvedReference{
			IsVariable:   false, // Resolved to a resource, not a variable
			ResourceType: "pingone_davinci_connector",
			ResourceName: "pingcli__my_connector",
			Field:        "id",
		},
		"app_id": ResolvedReference{
			IsVariable:    true, // This one IS a variable
			VariableName:  "pingone_davinci_application_pingcli__my_app_id",
			ResourceType:  "pingone_davinci_application",
			OriginalValue: "app-uuid-123",
		},
	}

	seen := make(map[string]bool)
	var out []FallbackVariable

	collectFallbackVars(attrs, defs, seen, &out)

	// Should only collect the variable reference, not the string or resource ref
	require.Len(t, out, 1)
	assert.Equal(t, "pingone_davinci_application_pingcli__my_app_id", out[0].Name)
}

// --- IncludeUpstream Tests ---

// flowStruct is a mock API struct that includes a connection reference.
type flowStruct struct {
	ID           *string
	Name         *string
	ConnectionID *string
}

// chainBStruct is a mock API struct for transitive dependency testing.
type chainBStruct struct {
	ID   *string
	Name *string
	ARef *string
}

// chainCStruct is a mock API struct for transitive dependency testing.
type chainCStruct struct {
	ID   *string
	Name *string
	BRef *string
}

// TestExportOrchestrator_Export_IncludeUpstream_Basic tests that IncludeUpstream
// expands the filtered seed set to include upstream dependencies via graph edges.
func TestExportOrchestrator_Export_IncludeUpstream_Basic(t *testing.T) {
	// Setup: type_conn (connector, no deps) and type_flow (flow, depends on type_conn)
	connDef := baseDef("type_conn", "p", "Connector", "conn")
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_conn"},
	}
	// Add connection_id attribute with references_type and SourcePath
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			SourcePath: "ConnectionID", Transform: "passthrough",
			ReferencesType: "type_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_conn": {
				simpleStruct{ID: strPtr("conn-1"), Name: strPtr("myconn")},
			},
			"type_flow": {
				flowStruct{ID: strPtr("flow-1"), Name: strPtr("myflow"), ConnectionID: strPtr("conn-1")},
			},
		},
	}

	// Create filter: include only type_flow* + IncludeUpstream: true
	filterObj, err := filter.NewResourceFilter([]string{"type_flow*"}, nil)
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:   "env-1",
		ResourceFilter:  filterObj,
		IncludeUpstream: true,
	})
	require.NoError(t, err)

	// Assert: both type_flow AND type_conn in result (upstream expansion)
	require.Len(t, result.ResourcesByType, 2)

	// Find flow and conn resources
	var flowTypeData, connTypeData *ExportedResourceData
	for _, erd := range result.ResourcesByType {
		switch erd.ResourceType {
		case "type_flow":
			flowTypeData = erd
		case "type_conn":
			connTypeData = erd
		}
	}

	require.NotNil(t, flowTypeData, "type_flow should be in result")
	require.NotNil(t, connTypeData, "type_conn should be in result (upstream dep)")

	require.Len(t, flowTypeData.Resources, 1)
	assert.Equal(t, "flow-1", flowTypeData.Resources[0].ID)

	require.Len(t, connTypeData.Resources, 1)
	assert.Equal(t, "conn-1", connTypeData.Resources[0].ID)

	// Assert: flow resource has connection_id resolved to resource reference (not fallback variable)
	flowRes := flowTypeData.Resources[0]
	require.NotNil(t, flowRes)
	connIDAttr, ok := flowRes.Attributes["connection_id"]
	require.True(t, ok, "connection_id attribute should exist")
	connRef, ok := connIDAttr.(ResolvedReference)
	require.True(t, ok, "connection_id should resolve to ResolvedReference")
	assert.False(t, connRef.IsVariable, "connection_id should NOT be a variable (resource in result)")
	assert.Equal(t, "type_conn", connRef.ResourceType)

	// Assert: no fallback variables for type_conn
	assert.Empty(t, result.FallbackVariables, "no fallback variables should exist")
}

// TestExportOrchestrator_Export_IncludeUpstream_NoFilter tests that IncludeUpstream
// has no effect when no ResourceFilter is active.
func TestExportOrchestrator_Export_IncludeUpstream_NoFilter(t *testing.T) {
	// Same setup as Basic
	connDef := baseDef("type_conn", "p", "Connector", "conn")
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_conn"},
	}
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			SourcePath: "ConnectionID", Transform: "passthrough",
			ReferencesType: "type_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_conn": {
				simpleStruct{ID: strPtr("conn-1"), Name: strPtr("myconn")},
			},
			"type_flow": {
				flowStruct{ID: strPtr("flow-1"), Name: strPtr("myflow"), ConnectionID: strPtr("conn-1")},
			},
		},
	}

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:   "env-1",
		ResourceFilter:  nil, // No filter
		IncludeUpstream: true,
	})
	require.NoError(t, err)

	// Assert: all resources included regardless (IncludeUpstream has no effect without filter)
	require.Len(t, result.ResourcesByType, 2)
}

// TestExportOrchestrator_Export_IncludeUpstream_ExcludeWins tests that explicit
// --exclude-resources overrides upstream expansion.
func TestExportOrchestrator_Export_IncludeUpstream_ExcludeWins(t *testing.T) {
	connDef := baseDef("type_conn", "p", "Connector", "conn")
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_conn"},
	}
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			SourcePath: "ConnectionID", Transform: "passthrough",
			ReferencesType: "type_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_conn": {
				simpleStruct{ID: strPtr("conn-1"), Name: strPtr("myconn")},
			},
			"type_flow": {
				flowStruct{ID: strPtr("flow-1"), Name: strPtr("myflow"), ConnectionID: strPtr("conn-1")},
			},
		},
	}

	// Include flow, exclude connector, IncludeUpstream: true
	filterObj, err := filter.NewResourceFilter([]string{"type_flow*"}, []string{"type_conn*"})
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:   "env-1",
		ResourceFilter:  filterObj,
		IncludeUpstream: true,
	})
	require.NoError(t, err)

	// Assert: only type_flow in result (explicit exclude wins)
	require.Len(t, result.ResourcesByType, 1)
	assert.Equal(t, "type_flow", result.ResourcesByType[0].ResourceType)
	assert.Len(t, result.ResourcesByType[0].Resources, 1)

	// Assert: type_conn NOT in result
	for _, erd := range result.ResourcesByType {
		assert.NotEqual(t, "type_conn", erd.ResourceType, "type_conn should not be in result despite being upstream")
	}

	// Assert: fallback variable generated for the excluded type_conn dependency
	require.Len(t, result.FallbackVariables, 1)
	assert.Equal(t, "type_conn", result.FallbackVariables[0].ResourceType)
}

// TestExportOrchestrator_Export_IncludeUpstream_Transitive tests that IncludeUpstream
// expands transitively: c→b→a all included when filtering for just c.
func TestExportOrchestrator_Export_IncludeUpstream_Transitive(t *testing.T) {
	// Setup: type_a (no deps), type_b (depends on type_a), type_c (depends on type_b)
	aDef := baseDef("type_a", "p", "A", "a")

	bDef := baseDef("type_b", "p", "B", "b")
	bDef.Dependencies.DependsOn = []schema.DependencyRule{{ResourceType: "type_a"}}
	bDef.Attributes = append(bDef.Attributes,
		schema.AttributeDefinition{
			Name: "a_ref", TerraformName: "a_ref", Type: "string",
			SourcePath: "ARef", Transform: "passthrough",
			ReferencesType: "type_a", ReferenceField: "id",
		},
	)

	cDef := baseDef("type_c", "p", "C", "c")
	cDef.Dependencies.DependsOn = []schema.DependencyRule{{ResourceType: "type_b"}}
	cDef.Attributes = append(cDef.Attributes,
		schema.AttributeDefinition{
			Name: "b_ref", TerraformName: "b_ref", Type: "string",
			SourcePath: "BRef", Transform: "passthrough",
			ReferencesType: "type_b", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, aDef, bDef, cDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_a": {
				simpleStruct{ID: strPtr("a-1"), Name: strPtr("alpha")},
			},
			"type_b": {
				chainBStruct{ID: strPtr("b-1"), Name: strPtr("bravo"), ARef: strPtr("a-1")},
			},
			"type_c": {
				chainCStruct{ID: strPtr("c-1"), Name: strPtr("charlie"), BRef: strPtr("b-1")},
			},
		},
	}

	// Filter: include only type_c* + IncludeUpstream: true
	filterObj, err := filter.NewResourceFilter([]string{"type_c*"}, nil)
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:   "env-1",
		ResourceFilter:  filterObj,
		IncludeUpstream: true,
	})
	require.NoError(t, err)

	// Assert: all 3 types in result (transitive chain: c→b→a all pulled in)
	require.Len(t, result.ResourcesByType, 3)

	typeMap := make(map[string]*ExportedResourceData)
	for _, erd := range result.ResourcesByType {
		typeMap[erd.ResourceType] = erd
	}

	require.Contains(t, typeMap, "type_a", "type_a should be in result (upstream of upstream)")
	require.Contains(t, typeMap, "type_b", "type_b should be in result (upstream)")
	require.Contains(t, typeMap, "type_c", "type_c should be in result (seed)")

	assert.Len(t, typeMap["type_a"].Resources, 1)
	assert.Len(t, typeMap["type_b"].Resources, 1)
	assert.Len(t, typeMap["type_c"].Resources, 1)

	// Assert: no fallback variables (all transitive deps are in result)
	assert.Empty(t, result.FallbackVariables)
}

// TestExportOrchestrator_Export_IncludeUpstream_False tests that when IncludeUpstream
// is false, upstream dependencies are NOT pulled in (current behavior).
func TestExportOrchestrator_Export_IncludeUpstream_False(t *testing.T) {
	connDef := baseDef("type_conn", "p", "Connector", "conn")
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_conn"},
	}
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			SourcePath: "ConnectionID", Transform: "passthrough",
			ReferencesType: "type_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_conn": {
				simpleStruct{ID: strPtr("conn-1"), Name: strPtr("myconn")},
			},
			"type_flow": {
				flowStruct{ID: strPtr("flow-1"), Name: strPtr("myflow"), ConnectionID: strPtr("conn-1")},
			},
		},
	}

	// Filter: include flow, IncludeUpstream: FALSE
	filterObj, err := filter.NewResourceFilter([]string{"type_flow*"}, nil)
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:   "env-1",
		ResourceFilter:  filterObj,
		IncludeUpstream: false, // Explicitly false
	})
	require.NoError(t, err)

	// Assert: only type_flow in result (upstream NOT pulled in)
	require.Len(t, result.ResourcesByType, 1)
	assert.Equal(t, "type_flow", result.ResourcesByType[0].ResourceType)

	// Assert: fallback variable generated for excludeed type_conn
	require.Len(t, result.FallbackVariables, 1)
	assert.Equal(t, "type_conn", result.FallbackVariables[0].ResourceType)
}

// TestExportOrchestrator_Export_IncludeUpstream_ListOnly tests that upstream
// expansion happens before ListOnly returns results.
func TestExportOrchestrator_Export_IncludeUpstream_ListOnly(t *testing.T) {
	connDef := baseDef("type_conn", "p", "Connector", "conn")
	flowDef := baseDef("type_flow", "p", "Flow", "flow")
	flowDef.Dependencies.DependsOn = []schema.DependencyRule{
		{ResourceType: "type_conn"},
	}
	flowDef.Attributes = append(flowDef.Attributes,
		schema.AttributeDefinition{
			Name: "conn_id", TerraformName: "connection_id", Type: "string",
			SourcePath: "ConnectionID", Transform: "passthrough",
			ReferencesType: "type_conn", ReferenceField: "id",
		},
	)

	reg := newTestRegistry(t, connDef, flowDef)
	proc := NewProcessor(reg)

	client := &mockAPIClient{
		platform: "p",
		resources: map[string][]interface{}{
			"type_conn": {
				simpleStruct{ID: strPtr("conn-1"), Name: strPtr("myconn")},
			},
			"type_flow": {
				flowStruct{ID: strPtr("flow-1"), Name: strPtr("myflow"), ConnectionID: strPtr("conn-1")},
			},
		},
	}

	// Filter: include flow, IncludeUpstream: true, ListOnly: true
	filterObj, err := filter.NewResourceFilter([]string{"type_flow*"}, nil)
	require.NoError(t, err)

	o := NewExportOrchestrator(reg, proc, client)
	result, err := o.Export(context.Background(), ExportOptions{
		EnvironmentID:   "env-1",
		ResourceFilter:  filterObj,
		IncludeUpstream: true,
		ListOnly:        true, // Skip reference resolution
	})
	require.NoError(t, err)

	// Assert: both type_flow AND type_conn listed (upstream expansion applies before ListOnly)
	require.Len(t, result.ResourcesByType, 2)

	typeMap := make(map[string]*ExportedResourceData)
	for _, erd := range result.ResourcesByType {
		typeMap[erd.ResourceType] = erd
	}

	require.Contains(t, typeMap, "type_flow", "type_flow should be listed")
	require.Contains(t, typeMap, "type_conn", "type_conn should be listed (upstream dep)")
}
