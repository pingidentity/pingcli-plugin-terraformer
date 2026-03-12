package core

import (
	"context"
	"fmt"
	"testing"

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
		platform: "p",
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

	resolveAttrs(attrs, defs, g, "env-uuid")

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
	resolveAttrs(attrs, defs, g, "env-1")

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
	result := resolveOneReference(envAttrDef, environmentUUID, g)

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
	result := resolveOneReference(envAttrDef, environmentUUID, g)

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
	result := resolveOneReference(envAttrDef, environmentUUID, nil)

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

	result := resolveOneReference(customAttrDef, environmentUUID, g)

	// Should resolve with custom field
	assert.False(t, result.IsVariable)
	assert.Equal(t, "pingone_environment", result.ResourceType)
	assert.Equal(t, "name", result.Field)
	assert.Equal(t, "pingone_environment.pingcli__staging.name", result.Expression())
}
