package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/graph"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// Helper to create minimal ResourceDefinition for testing
func testResourceDef(resourceType string) *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			ResourceType: resourceType,
		},
	}
}

// TestEmbeddedReferenceRegistry_RegisterAndRetrieve tests basic registry operations
func TestEmbeddedReferenceRegistry_RegisterAndRetrieve(t *testing.T) {
	reg := NewEmbeddedReferenceRegistry()

	rule1 := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	rule2 := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.connectors.*.data.properties",
		TargetResourceType: "pingone_davinci_connector",
		JSONKeyPath:        "connectorId.value",
		ReferenceField:     "id",
	}

	// Initially empty
	if len(reg.Rules()) != 0 {
		t.Errorf("expected empty registry, got %d rules", len(reg.Rules()))
	}

	// Register first rule
	reg.Register(rule1)
	if len(reg.Rules()) != 1 {
		t.Errorf("expected 1 rule, got %d", len(reg.Rules()))
	}

	// Register second rule
	reg.Register(rule2)
	if len(reg.Rules()) != 2 {
		t.Errorf("expected 2 rules, got %d", len(reg.Rules()))
	}

	// Verify rules are retrievable
	rules := reg.Rules()
	if rules[0].ResourceType != "pingone_davinci_flow" {
		t.Errorf("expected first rule ResourceType pingone_davinci_flow, got %s", rules[0].ResourceType)
	}
	if rules[1].TargetResourceType != "pingone_davinci_connector" {
		t.Errorf("expected second rule TargetResourceType pingone_davinci_connector, got %s", rules[1].TargetResourceType)
	}
}

// TestResolveEmbeddedReferences_SingleSubFlow tests replacement of a single
// embedded subFlowId UUID with a Terraform reference
func TestResolveEmbeddedReferences_SingleSubFlow(t *testing.T) {
	// Create graph with target flow and source flow
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "aaaaaaaa-0000-4000-8000-000000000001", "pingcli__My-0020-Flow")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue("jsonencode({\n  \"nodeTitle\": {\n    \"value\": \"Sign On Flow\"\n  },\n  \"subFlowId\": {\n    \"value\": {\n      \"label\": \"My Flow\",\n      \"value\": \"aaaaaaaa-0000-4000-8000-000000000001\"\n    }\n  }\n})"),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	if !strings.Contains(string(resolvedValue), "${pingone_davinci_flow.pingcli__My-0020-Flow.id}") {
		t.Errorf("expected UUID to be replaced with reference in RawHCLValue, got: %s", resolvedValue)
	}

	// UUID string itself should no longer appear
	if strings.Contains(string(resolvedValue), "\"aaaaaaaa-0000-4000-8000-000000000001\"") {
		t.Errorf("expected raw UUID to be removed, still found in: %s", resolvedValue)
	}

	// Verify graph edge created
	deps := g.GetDependencies("pingone_davinci_flow", "parent-flow-id")
	if len(deps) == 0 {
		t.Error("expected graph edge to be created")
	}

	// nodeTitle should be unchanged
	if !strings.Contains(string(resolvedValue), "Sign On Flow") {
		t.Error("expected nodeTitle to be preserved")
	}
}

// TestResolveEmbeddedReferences_MultipleNodes tests processing of multiple
// nodes within a single flow, some with subFlowId and some without
func TestResolveEmbeddedReferences_MultipleNodes(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "aaaaaaaa-0000-4000-8000-000000000002", "pingcli__Sub-0020-Flow-1")
	g.AddResource("pingone_davinci_flow", "flow-sub2", "pingcli__Sub-0020-Flow-2")
	g.AddResource("pingone_davinci_flow", "parent-flow", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	// node1 has subFlowId, node2 has no subFlowId
	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue("jsonencode({\n  \"subFlowId\": {\n    \"value\": {\n      \"label\": \"Sub Flow 1\",\n      \"value\": \"aaaaaaaa-0000-4000-8000-000000000002\"\n    }\n  }\n})"),
						},
					},
					"node2": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue("jsonencode({\n  \"nodeTitle\": {\n    \"value\": \"My Node\"\n  }\n})"),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	node2Before := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node2"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	nodes := resourceData.Attributes["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})

	node1Props := nodes["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
	node2Props := nodes["node2"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// node1 should have subFlowId replaced
	if !strings.Contains(string(node1Props), "${pingone_davinci_flow.pingcli__Sub-0020-Flow-1.id}") {
		t.Errorf("node1 should have subFlowId replaced, got: %s", node1Props)
	}

	// node2 should be unchanged (no subFlowId)
	if node2Props != node2Before {
		t.Errorf("node2 should be unchanged, got: %s", node2Props)
	}
}

// TestResolveEmbeddedReferences_UUIDNotInGraph tests that when an embedded
// UUID doesn't exist in the graph, it's left unchanged and no edge is added
func TestResolveEmbeddedReferences_UUIDNotInGraph(t *testing.T) {
	g := graph.New()
	// Add source but not target — "flow-unknown" is not in the graph
	g.AddResource("pingone_davinci_flow", "parent-flow", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-unknown"}}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
	if resolvedValue != originalValue {
		t.Errorf("expected UUID to remain unchanged when not in graph, got: %s", resolvedValue)
	}

	deps := g.GetDependencies("pingone_davinci_flow", "parent-flow")
	if len(deps) != 0 {
		t.Errorf("expected no edges to be added, got %d", len(deps))
	}
}

// TestResolveEmbeddedReferences_NoMatchingJSONPath tests when properties
// don't contain the expected JSON key path (subFlowId missing)
func TestResolveEmbeddedReferences_NoMatchingJSONPath(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "flow-sub1", "pingcli__Sub-Flow-1")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	// Properties without subFlowId
	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"nodeTitle": {"value": "Sign On"}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// Verify: RawHCLValue should remain unchanged if key path doesn't exist
	if resolvedValue != originalValue {
		t.Error("expected RawHCLValue unchanged when JSONKeyPath not found")
	}
}

// TestResolveEmbeddedReferences_WildcardTraversal verifies that the wildcard
// (*) in AttributePath correctly iterates over all map keys at that level
func TestResolveEmbeddedReferences_WildcardTraversal(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "aaaaaaaa-0000-4000-8000-00000000000a", "pingcli__Flow-A")
	g.AddResource("pingone_davinci_flow", "aaaaaaaa-0000-4000-8000-00000000000b", "pingcli__Flow-B")
	g.AddResource("pingone_davinci_flow", "aaaaaaaa-0000-4000-8000-00000000000c", "pingcli__Flow-C")
	g.AddResource("pingone_davinci_flow", "parent-flow", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	// Three nodes with different IDs
	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"nodeA": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "aaaaaaaa-0000-4000-8000-00000000000a"}}})`),
						},
					},
					"nodeB": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "aaaaaaaa-0000-4000-8000-00000000000b"}}})`),
						},
					},
					"nodeC": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "aaaaaaaa-0000-4000-8000-00000000000c"}}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	nodes := resourceData.Attributes["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})

	// Verify all three nodes were processed
	expected := map[string]string{
		"nodeA": "${pingone_davinci_flow.pingcli__Flow-A.id}",
		"nodeB": "${pingone_davinci_flow.pingcli__Flow-B.id}",
		"nodeC": "${pingone_davinci_flow.pingcli__Flow-C.id}",
	}
	for nodeKey, ref := range expected {
		props := nodes[nodeKey].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
		if !strings.Contains(string(props), ref) {
			t.Errorf("node %s should contain %s, got: %s", nodeKey, ref, props)
		}
	}

	// Verify 3 graph edges created
	deps := g.GetDependencies("pingone_davinci_flow", "parent-flow")
	if len(deps) != 3 {
		t.Errorf("expected 3 graph edges, got %d", len(deps))
	}
}

// TestResolveEmbeddedReferences_NoRulesNoOp verifies that when rules is empty,
// no processing occurs and data remains unchanged
func TestResolveEmbeddedReferences_NoRulesNoOp(t *testing.T) {
	g := graph.New()

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-unknown"}}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// Call with empty rules
	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	if resolvedValue != originalValue {
		t.Error("expected no changes with empty rules")
	}
}

// TestResolveEmbeddedReferences_DifferentResourceTypeSkipped verifies that
// rules are only applied to matching resource types
func TestResolveEmbeddedReferences_DifferentResourceTypeSkipped(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "flow-sub1", "pingcli__Sub-Flow")
	g.AddResource("pingone_davinci_connector", "connector-1", "pingcli__My-Connector")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-sub1"}}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_connector", // Different type!
		ID:           "connector-1",
		Label:        "pingcli__My-Connector",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_connector",
		Definition:   testResourceDef("pingone_davinci_connector"),
		Resources:    []*ResourceData{resourceData},
	}

	originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// Verify: rule not applied because resource type doesn't match
	if resolvedValue != originalValue {
		t.Error("expected rule not to apply to different resource type")
	}
}

// TestResolveEmbeddedReferences_ExtensibleNewRule verifies that custom rules
// with different JSONKeyPath values work correctly, demonstrating extensibility
func TestResolveEmbeddedReferences_ExtensibleNewRule(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_connector", "aaaaaaaa-0000-4000-8000-000000000d01", "pingcli__Custom-Connector")
	g.AddResource("pingone_davinci_flow", "flow-with-connector", "pingcli__Flow-With-Connector")

	// Custom rule with different JSONKeyPath
	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_connector",
		JSONKeyPath:        "customConnectorId.value", // Different key path!
		ReferenceField:     "id",
	}

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"customConnectorId": {"value": "aaaaaaaa-0000-4000-8000-000000000d01"}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "flow-with-connector",
		Label:        "pingcli__Flow-With-Connector",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// Verify: custom key path was replaced
	if !strings.Contains(string(resolvedValue), "${pingone_davinci_connector.pingcli__Custom-Connector.id}") {
		t.Errorf("expected custom key path to be replaced, got: %s", resolvedValue)
	}
}

// TestResolveEmbeddedReferences_PreservesOtherAttributes verifies that
// attributes outside the targeted paths are not modified
func TestResolveEmbeddedReferences_PreservesOtherAttributes(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "aaaaaaaa-0000-4000-8000-000000000e01", "pingcli__Sub-Flow")
	g.AddResource("pingone_davinci_flow", "parent-flow", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	}

	attrs := map[string]interface{}{
		"name":        "My Flow",
		"description": "A test flow",
		"other_data": map[string]interface{}{
			"unrelated": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-sub1"}}})`),
		},
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "aaaaaaaa-0000-4000-8000-000000000e01"}}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	originalOtherData := attrs["other_data"].(map[string]interface{})["unrelated"].(RawHCLValue)

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	// Verify: name and description unchanged
	if attrs["name"] != "My Flow" {
		t.Error("expected name to remain unchanged")
	}
	if attrs["description"] != "A test flow" {
		t.Error("expected description to remain unchanged")
	}

	// Verify: other_data.unrelated unchanged (outside target path)
	if attrs["other_data"].(map[string]interface{})["unrelated"].(RawHCLValue) != originalOtherData {
		t.Error("expected unrelated data to remain unchanged")
	}

	// Verify: graph_data.elements.nodes.*.data.properties WAS modified
	targetProps := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
	if !strings.Contains(string(targetProps), "${pingone_davinci_flow.pingcli__Sub-Flow.id}") {
		t.Error("expected targeted properties to be modified")
	}
}

// TestResolveEmbeddedReferences_StrategyReferenceWithFallback_UUIDNotInGraph tests
// that when Strategy is "reference_with_fallback" and the UUID is not in the graph,
// the UUID is replaced with a Terraform var reference and a FallbackVariable is returned.
func TestResolveEmbeddedReferences_StrategyReferenceWithFallback_UUIDNotInGraph(t *testing.T) {
	g := graph.New()
	// No pingone_davinci_form resources in graph — only the parent flow
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_form",
		JSONKeyPath:        "form.value",
		ReferenceField:     "id",
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "davinci_form",
		VariableNamingPath: "nodeTitle.value",
	}

	// Properties include both the UUID field and the naming field
	propertiesJSON := `jsonencode({"form": {"value": "bbbbbbbb-0000-4000-8000-000000000001"}, "nodeTitle": {"value": "Example - Sign On"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// UUID should be replaced with var reference; "Example - Sign On" → toLower → sanitize = "example___sign_on"
	const expectedVarRef = "${var.davinci_form_example___sign_on}"
	if !strings.Contains(string(resolvedValue), expectedVarRef) {
		t.Errorf("expected UUID replaced with %q, got: %s", expectedVarRef, resolvedValue)
	}

	// Raw UUID string should no longer appear
	if strings.Contains(string(resolvedValue), `"bbbbbbbb-0000-4000-8000-000000000001"`) {
		t.Errorf("expected raw UUID to be removed, still found in: %s", resolvedValue)
	}

	// nodeTitle should be preserved in the JSON
	if !strings.Contains(string(resolvedValue), "Example - Sign On") {
		t.Error("expected nodeTitle value to be preserved in JSON")
	}

	// A single FallbackVariable should be returned
	if len(fallbackVars) != 1 {
		t.Fatalf("expected 1 FallbackVariable, got %d", len(fallbackVars))
	}
	if fallbackVars[0].Name != "davinci_form_example___sign_on" {
		t.Errorf("expected FallbackVariable.Name %q, got %q", "davinci_form_example___sign_on", fallbackVars[0].Name)
	}
	if fallbackVars[0].Type != "string" {
		t.Errorf("expected FallbackVariable.Type %q, got %q", "string", fallbackVars[0].Type)
	}
}

// TestResolveEmbeddedReferences_StrategyReferenceWithFallback_UUIDInGraph tests
// that when Strategy is "reference_with_fallback" and the UUID IS in the graph,
// the UUID is replaced with a resource reference (not a variable) and no
// FallbackVariable is returned.
func TestResolveEmbeddedReferences_StrategyReferenceWithFallback_UUIDInGraph(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_form", "bbbbbbbb-0000-4000-8000-000000000001", "pingcli__Example_Sign_On")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_form",
		JSONKeyPath:        "form.value",
		ReferenceField:     "id",
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "davinci_form",
		VariableNamingPath: "nodeTitle.value",
	}

	propertiesJSON := `jsonencode({"form": {"value": "bbbbbbbb-0000-4000-8000-000000000001"}, "nodeTitle": {"value": "Example - Sign On"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// UUID should be replaced with a resource reference, not a variable
	const expectedRef = "${pingone_davinci_form.pingcli__Example_Sign_On.id}"
	if !strings.Contains(string(resolvedValue), expectedRef) {
		t.Errorf("expected UUID replaced with resource reference %q, got: %s", expectedRef, resolvedValue)
	}

	// Should NOT contain a var reference
	if strings.Contains(string(resolvedValue), "${var.") {
		t.Errorf("expected no var reference when UUID is in graph, got: %s", resolvedValue)
	}

	// No FallbackVariable should be returned
	if len(fallbackVars) != 0 {
		t.Errorf("expected 0 FallbackVariables when UUID resolved to reference, got %d", len(fallbackVars))
	}
}

// TestResolveEmbeddedReferences_StrategyVariable_AlwaysEmitsVariable tests that
// when Strategy is "variable", a FallbackVariable is always emitted even when
// the target UUID can be resolved in the graph.
func TestResolveEmbeddedReferences_StrategyVariable_AlwaysEmitsVariable(t *testing.T) {
	g := graph.New()
	// Target IS in the graph — but strategy "variable" should still emit a var
	g.AddResource("pingone_davinci_form", "bbbbbbbb-0000-4000-8000-000000000002", "pingcli__Some_Form")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_form",
		JSONKeyPath:        "form.value",
		ReferenceField:     "id",
		Strategy:           "variable",
		VariablePrefix:     "external_id",
		VariableNamingPath: "nodeTitle.value",
	}

	propertiesJSON := `jsonencode({"form": {"value": "bbbbbbbb-0000-4000-8000-000000000002"}, "nodeTitle": {"value": "Sign On Node"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// "Sign On Node" → toLower → "sign on node" → sanitize → "sign_on_node"
	const expectedVarRef = "${var.external_id_sign_on_node}"
	if !strings.Contains(string(resolvedValue), expectedVarRef) {
		t.Errorf("expected var reference %q for strategy=variable, got: %s", expectedVarRef, resolvedValue)
	}

	// Should NOT contain a direct resource reference
	if strings.Contains(string(resolvedValue), "${pingone_davinci_form.") {
		t.Errorf("expected no resource reference for strategy=variable, got: %s", resolvedValue)
	}

	// A FallbackVariable must always be returned for strategy=variable
	if len(fallbackVars) != 1 {
		t.Fatalf("expected 1 FallbackVariable for strategy=variable, got %d", len(fallbackVars))
	}
	if fallbackVars[0].Name != "external_id_sign_on_node" {
		t.Errorf("expected FallbackVariable.Name %q, got %q", "external_id_sign_on_node", fallbackVars[0].Name)
	}
	if fallbackVars[0].Type != "string" {
		t.Errorf("expected FallbackVariable.Type %q, got %q", "string", fallbackVars[0].Type)
	}
}

// TestResolveEmbeddedReferences_StrategyDefault_BackwardCompatible validates that
// rules with an empty Strategy field preserve the original behaviour:
//   - UUID in graph  → replaced with resource reference, no variable emitted
//   - UUID not in graph → left unchanged, no variable emitted
func TestResolveEmbeddedReferences_StrategyDefault_BackwardCompatible(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_davinci_flow", "bbbbbbbb-0000-4000-8000-000000000003", "pingcli__In-Graph-Flow")
	g.AddResource("pingone_davinci_flow", "parent-flow", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
		// Strategy intentionally empty — default/backward-compatible behaviour
	}

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node-known": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "bbbbbbbb-0000-4000-8000-000000000003"}}})`),
						},
					},
					"node-unknown": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-not-in-graph"}}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	unknownBefore := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node-unknown"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	nodes := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})
	knownProps := nodes["node-known"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
	unknownProps := nodes["node-unknown"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// UUID in graph → resource reference
	if !strings.Contains(string(knownProps), "${pingone_davinci_flow.pingcli__In-Graph-Flow.id}") {
		t.Errorf("expected in-graph UUID to be resolved to reference, got: %s", knownProps)
	}

	// UUID not in graph → left unchanged (no variable emitted)
	if unknownProps != unknownBefore {
		t.Errorf("expected out-of-graph UUID to remain unchanged, got: %s", unknownProps)
	}

	// No FallbackVariables for default strategy
	if len(fallbackVars) != 0 {
		t.Errorf("expected 0 FallbackVariables for default strategy, got %d", len(fallbackVars))
	}
}

// TestResolveEmbeddedReferences_VariableNamingPath_Missing tests that when
// VariableNamingPath is specified but the key is absent from the properties JSON,
// the implementation falls back to UUID-prefix naming using the first 8 chars of
// the UUID (sanitized).
func TestResolveEmbeddedReferences_VariableNamingPath_Missing(t *testing.T) {
	g := graph.New()
	// Target UUID not in graph
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_form",
		JSONKeyPath:        "form.value",
		ReferenceField:     "id",
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "davinci_form",
		VariableNamingPath: "nodeTitle.value", // key absent from JSON below
	}

	// Properties do NOT contain nodeTitle — only the UUID field
	propertiesJSON := `jsonencode({"form": {"value": "abcde123-0000-4000-8000-000000000000"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// UUID "abcde123-0000-4000-8000-000000000000": first 8 chars = "abcde123", SanitizeVariableName = "abcde123"
	// Variable name = "davinci_form_abcde123"
	const expectedVarRef = "${var.davinci_form_abcde123}"
	if !strings.Contains(string(resolvedValue), expectedVarRef) {
		t.Errorf("expected UUID-prefix fallback var reference %q, got: %s", expectedVarRef, resolvedValue)
	}

	if len(fallbackVars) != 1 {
		t.Fatalf("expected 1 FallbackVariable, got %d", len(fallbackVars))
	}
	if fallbackVars[0].Name != "davinci_form_abcde123" {
		t.Errorf("expected FallbackVariable.Name %q (UUID-prefix fallback), got %q", "davinci_form_abcde123", fallbackVars[0].Name)
	}
	if fallbackVars[0].Type != "string" {
		t.Errorf("expected FallbackVariable.Type %q, got %q", "string", fallbackVars[0].Type)
	}
}

// TestResolveEmbeddedReferences_MultipleNodesDistinctVariables tests that when
// multiple nodes each reference different UUIDs with different titles, each gets
// its own uniquely-named FallbackVariable.
func TestResolveEmbeddedReferences_MultipleNodesDistinctVariables(t *testing.T) {
	g := graph.New()
	// No pingone_davinci_form resources in graph
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_form",
		JSONKeyPath:        "form.value",
		ReferenceField:     "id",
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "davinci_form",
		VariableNamingPath: "nodeTitle.value",
	}

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"form": {"value": "bbbbbbbb-0000-4000-8000-000000000011"}, "nodeTitle": {"value": "Login Form"}})`),
						},
					},
					"node2": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"form": {"value": "bbbbbbbb-0000-4000-8000-000000000022"}, "nodeTitle": {"value": "Signup Form"}})`),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	nodes := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})
	node1Props := nodes["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
	node2Props := nodes["node2"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// "Login Form" → "login_form" → "davinci_form_login_form"
	if !strings.Contains(string(node1Props), "${var.davinci_form_login_form}") {
		t.Errorf("expected node1 to contain ${var.davinci_form_login_form}, got: %s", node1Props)
	}

	// "Signup Form" → "signup_form" → "davinci_form_signup_form"
	if !strings.Contains(string(node2Props), "${var.davinci_form_signup_form}") {
		t.Errorf("expected node2 to contain ${var.davinci_form_signup_form}, got: %s", node2Props)
	}

	// Two distinct FallbackVariables must be returned
	if len(fallbackVars) != 2 {
		t.Fatalf("expected 2 FallbackVariables for 2 distinct UUIDs, got %d", len(fallbackVars))
	}

	names := make(map[string]bool)
	for _, fv := range fallbackVars {
		names[fv.Name] = true
	}
	if !names["davinci_form_login_form"] {
		t.Error("expected FallbackVariable davinci_form_login_form not found")
	}
	if !names["davinci_form_signup_form"] {
		t.Error("expected FallbackVariable davinci_form_signup_form not found")
	}
}

// TestResolveEmbeddedReferences_DuplicateUUIDs_Deduplicated tests that when
// multiple nodes reference the same UUID, only one FallbackVariable is returned.
func TestResolveEmbeddedReferences_DuplicateUUIDs_Deduplicated(t *testing.T) {
	g := graph.New()
	// Target UUID not in graph
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_form",
		JSONKeyPath:        "form.value",
		ReferenceField:     "id",
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "davinci_form",
		VariableNamingPath: "nodeTitle.value",
	}

	// Two nodes share the same UUID
	sharedProps := `jsonencode({"form": {"value": "bbbbbbbb-0000-4000-8000-000000000033"}, "nodeTitle": {"value": "Shared Form"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(sharedProps),
						},
					},
					"node2": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(sharedProps),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	// Both nodes should have the var reference substituted
	nodes := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})
	for _, nodeKey := range []string{"node1", "node2"} {
		props := nodes[nodeKey].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)
		if !strings.Contains(string(props), "${var.davinci_form_shared_form}") {
			t.Errorf("expected %s to contain ${var.davinci_form_shared_form}, got: %s", nodeKey, props)
		}
	}

	// Only ONE FallbackVariable despite two nodes referencing the same UUID
	if len(fallbackVars) != 1 {
		t.Errorf("expected 1 deduplicated FallbackVariable, got %d", len(fallbackVars))
	}
	if len(fallbackVars) > 0 && fallbackVars[0].Name != "davinci_form_shared_form" {
		t.Errorf("expected FallbackVariable.Name %q, got %q", "davinci_form_shared_form", fallbackVars[0].Name)
	}
}

// TestLooksLikeUUID verifies the UUID-format guard helper against a mix of
// valid UUID-shaped strings and the non-UUID placeholder/sentinel strings
// used throughout this file's fixtures.
func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid hyphenated UUID", "860b5cd5-45cc-466d-abbd-64298bb90bed", true},
		{"valid UUID uppercase", "860B5CD5-45CC-466D-ABBD-64298BB90BED", true},
		{"valid UUID no hyphens", "860b5cd545cc466dabbd64298bb90bed", true},
		{"empty string", "", false},
		{"sentinel useThemeId", "useThemeId", false},
		{"sentinel activeTheme", "activeTheme", false},
		{"legacy placeholder flow-abc123", "flow-abc123", false},
		{"legacy placeholder form-uuid-xyz", "form-uuid-xyz", false},
		{"legacy placeholder conn-custom1", "conn-custom1", false},
		{"truncated uuid-like string", "abcde123-4567-890", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeUUID(tt.input); got != tt.want {
				t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestResolveEmbeddedReferences_Precondition_Match verifies that a rule with
// PreconditionKeyPath set fires when the sibling path resolves to exactly
// PreconditionValue.
func TestResolveEmbeddedReferences_Precondition_Match(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_branding_theme", "cccccccc-0000-4000-8000-000000000001", "pingcli__My_Theme")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:        "pingone_davinci_flow",
		AttributePath:       "graph_data.elements.nodes.*.data.properties",
		TargetResourceType:  "pingone_branding_theme",
		JSONKeyPath:         "themeId.value",
		ReferenceField:      "id",
		Strategy:            "reference_with_fallback",
		VariablePrefix:      "davinci_theme",
		VariableNamingPath:  "nodeTitle.value",
		PreconditionKeyPath: "theme.value",
		PreconditionValue:   "useThemeId",
	}

	propertiesJSON := `jsonencode({"theme": {"value": "useThemeId"}, "themeId": {"value": "cccccccc-0000-4000-8000-000000000001"}, "nodeTitle": {"value": "Theme Node"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	const expectedRef = "${pingone_branding_theme.pingcli__My_Theme.id}"
	if !strings.Contains(string(resolvedValue), expectedRef) {
		t.Errorf("expected precondition-satisfied rule to resolve to %q, got: %s", expectedRef, resolvedValue)
	}

	// theme.value (the precondition key itself) must remain unchanged
	if !strings.Contains(string(resolvedValue), `"useThemeId"`) {
		t.Error("expected theme.value to remain the literal string \"useThemeId\"")
	}
}

// TestResolveEmbeddedReferences_Precondition_NoMatch verifies that when the
// precondition key resolves to a value other than PreconditionValue, the
// rule no-ops: no change, no fallback variable, no graph edge.
func TestResolveEmbeddedReferences_Precondition_NoMatch(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_branding_theme", "cccccccc-0000-4000-8000-000000000002", "pingcli__My_Theme")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:        "pingone_davinci_flow",
		AttributePath:       "graph_data.elements.nodes.*.data.properties",
		TargetResourceType:  "pingone_branding_theme",
		JSONKeyPath:         "themeId.value",
		ReferenceField:      "id",
		Strategy:            "reference_with_fallback",
		VariablePrefix:      "davinci_theme",
		VariableNamingPath:  "nodeTitle.value",
		PreconditionKeyPath: "theme.value",
		PreconditionValue:   "useThemeId",
	}

	// theme.value is a direct UUID (case 1 shape), not "useThemeId" — precondition fails
	propertiesJSON := `jsonencode({"theme": {"value": "cccccccc-0000-4000-8000-000000000003"}, "themeId": {"value": "cccccccc-0000-4000-8000-000000000002"}, "nodeTitle": {"value": "Theme Node"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	if resolvedValue != originalValue {
		t.Errorf("expected value unchanged when precondition doesn't match, got: %s", resolvedValue)
	}
	if len(fallbackVars) != 0 {
		t.Errorf("expected 0 FallbackVariables when precondition doesn't match, got %d", len(fallbackVars))
	}

	deps := g.GetDependencies("pingone_davinci_flow", "parent-flow-id")
	if len(deps) != 0 {
		t.Errorf("expected no graph edges when precondition doesn't match, got %d", len(deps))
	}
}

// TestResolveEmbeddedReferences_Precondition_KeyAbsent verifies that when the
// precondition key is entirely absent from the JSON, it is treated as a
// non-match — the rule no-ops.
func TestResolveEmbeddedReferences_Precondition_KeyAbsent(t *testing.T) {
	g := graph.New()
	g.AddResource("pingone_branding_theme", "cccccccc-0000-4000-8000-000000000004", "pingcli__My_Theme")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := EmbeddedReferenceRule{
		ResourceType:        "pingone_davinci_flow",
		AttributePath:       "graph_data.elements.nodes.*.data.properties",
		TargetResourceType:  "pingone_branding_theme",
		JSONKeyPath:         "themeId.value",
		ReferenceField:      "id",
		Strategy:            "reference_with_fallback",
		VariablePrefix:      "davinci_theme",
		VariableNamingPath:  "nodeTitle.value",
		PreconditionKeyPath: "theme.value",
		PreconditionValue:   "useThemeId",
	}

	// "theme" key is entirely absent (customForm-capability-like shape)
	propertiesJSON := `jsonencode({"themeId": {"value": "cccccccc-0000-4000-8000-000000000004"}, "nodeTitle": {"value": "Theme Node"}})`

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	if resolvedValue != originalValue {
		t.Errorf("expected value unchanged when precondition key absent, got: %s", resolvedValue)
	}
	if len(fallbackVars) != 0 {
		t.Errorf("expected 0 FallbackVariables when precondition key absent, got %d", len(fallbackVars))
	}
}

// themeRichTextRule mirrors the theme.value/themeId.value case-3 rule shape
// that Task 2 will register against pingone_branding_theme.
func themeRichTextRule() EmbeddedReferenceRule {
	return EmbeddedReferenceRule{
		ResourceType:        "pingone_davinci_flow",
		AttributePath:       "graph_data.elements.nodes.*.data.properties",
		TargetResourceType:  "pingone_branding_theme",
		JSONKeyPath:         "themeId.value",
		ReferenceField:      "id",
		Strategy:            "reference_with_fallback",
		VariablePrefix:      "davinci_theme",
		VariableNamingPath:  "nodeTitle.value",
		PreconditionKeyPath: "theme.value",
		PreconditionValue:   "useThemeId",
		UnwrapMode:          "rich_text",
	}
}

// richTextPropertiesJSON builds a properties RawHCLValue blob shaped like a
// showForm node using the "useThemeId" mode flag, with the UUID embedded
// inside a Slate-style rich-text wrapper at themeId.value — matching the
// double-JSON-encoding that transformJSONEncodeRaw produces in production.
func richTextPropertiesJSON(t *testing.T, uuidStr string) string {
	t.Helper()

	wrapper := `[{"children":[{"text":"` + uuidStr + `"}]}]`
	escapedWrapper, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("failed to marshal wrapper: %v", err)
	}

	return `jsonencode({"theme": {"value": "useThemeId"}, "themeId": {"value": ` + string(escapedWrapper) + `}, "nodeTitle": {"value": "Theme Node"}})`
}

// TestResolveEmbeddedReferences_RichTextUnwrap_ResolvesToVariable verifies
// that UnwrapMode "rich_text" correctly extracts the UUID from the Slate
// wrapper, resolves it via reference_with_fallback when not in the graph,
// and re-embeds the variable reference back inside the wrapper, preserving
// its outer JSON structure.
func TestResolveEmbeddedReferences_RichTextUnwrap_ResolvesToVariable(t *testing.T) {
	g := graph.New()
	// Target UUID not in graph — no pingone_branding_theme resources registered
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := themeRichTextRule()
	const themeUUID = "dddddddd-0000-4000-8000-000000000001"
	propertiesJSON := richTextPropertiesJSON(t, themeUUID)

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	// theme.value must remain the literal string "useThemeId"
	if !strings.Contains(string(resolvedValue), `"useThemeId"`) {
		t.Error("expected theme.value to remain the literal string \"useThemeId\"")
	}

	// The raw UUID must no longer appear
	if strings.Contains(string(resolvedValue), themeUUID) {
		t.Errorf("expected raw UUID to be removed from wrapper, still found in: %s", resolvedValue)
	}

	const expectedVarRef = "${var.davinci_theme_theme_node}"
	if !strings.Contains(string(resolvedValue), expectedVarRef) {
		t.Errorf("expected wrapper to contain var reference %q, got: %s", expectedVarRef, resolvedValue)
	}

	// Verify the wrapper's outer JSON structure round-trips: unmarshal the
	// outer properties JSON, then unmarshal themeId.value (a JSON string)
	// back into the Slate array shape with the resolved value swapped in.
	jsonStr := extractJSONFromRawHCL(resolvedValue)
	var outer map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &outer); err != nil {
		t.Fatalf("expected resolved properties to remain valid JSON: %v", err)
	}
	themeIDValue := outer["themeId"].(map[string]interface{})["value"].(string)

	var wrapperArr []map[string]interface{}
	if err := json.Unmarshal([]byte(themeIDValue), &wrapperArr); err != nil {
		t.Fatalf("expected themeId.value to round-trip as a JSON array: %v", err)
	}
	if len(wrapperArr) != 1 {
		t.Fatalf("expected wrapper array to have exactly 1 element, got %d", len(wrapperArr))
	}
	children := wrapperArr[0]["children"].([]interface{})
	text := children[0].(map[string]interface{})["text"].(string)
	if text != expectedVarRef {
		t.Errorf("expected wrapper inner text to be %q, got %q", expectedVarRef, text)
	}

	if len(fallbackVars) != 1 {
		t.Fatalf("expected 1 FallbackVariable, got %d", len(fallbackVars))
	}
	if fallbackVars[0].Name != "davinci_theme_theme_node" {
		t.Errorf("expected FallbackVariable.Name %q, got %q", "davinci_theme_theme_node", fallbackVars[0].Name)
	}
}

// TestResolveEmbeddedReferences_RichTextUnwrap_ResolvesToReference verifies
// that UnwrapMode "rich_text" resolves to a direct resource reference (with
// a graph edge) when the unwrapped UUID IS present in the graph.
func TestResolveEmbeddedReferences_RichTextUnwrap_ResolvesToReference(t *testing.T) {
	g := graph.New()
	const themeUUID = "dddddddd-0000-4000-8000-000000000002"
	g.AddResource("pingone_branding_theme", themeUUID, "pingcli__My_Theme")
	g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

	rule := themeRichTextRule()
	propertiesJSON := richTextPropertiesJSON(t, themeUUID)

	attrs := map[string]interface{}{
		"graph_data": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": map[string]interface{}{
					"node1": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(propertiesJSON),
						},
					},
				},
			},
		},
	}

	resourceData := &ResourceData{
		ResourceType: "pingone_davinci_flow",
		ID:           "parent-flow-id",
		Label:        "pingcli__Parent-Flow",
		Attributes:   attrs,
	}

	exportedData := &ExportedResourceData{
		ResourceType: "pingone_davinci_flow",
		Definition:   testResourceDef("pingone_davinci_flow"),
		Resources:    []*ResourceData{resourceData},
	}

	fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

	resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

	if !strings.Contains(string(resolvedValue), `"useThemeId"`) {
		t.Error("expected theme.value to remain the literal string \"useThemeId\"")
	}

	const expectedRef = "${pingone_branding_theme.pingcli__My_Theme.id}"
	if !strings.Contains(string(resolvedValue), expectedRef) {
		t.Errorf("expected wrapper to contain resource reference %q, got: %s", expectedRef, resolvedValue)
	}

	if strings.Contains(string(resolvedValue), themeUUID) {
		t.Errorf("expected raw UUID to be removed from wrapper, still found in: %s", resolvedValue)
	}

	if len(fallbackVars) != 0 {
		t.Errorf("expected 0 FallbackVariables when UUID resolved to reference, got %d", len(fallbackVars))
	}

	deps := g.GetDependencies("pingone_davinci_flow", "parent-flow-id")
	if len(deps) == 0 {
		t.Error("expected graph edge to be created for rich-text-unwrapped reference")
	}
}

// TestResolveEmbeddedReferences_RichTextUnwrap_MalformedWrapper verifies
// that a variety of malformed/unexpected wrapper shapes leave the value
// unchanged, emit no fallback variable/graph edge, and never panic.
func TestResolveEmbeddedReferences_RichTextUnwrap_MalformedWrapper(t *testing.T) {
	tests := []struct {
		name          string
		themeIDValueJ string // JSON-encoded (escaped) themeId.value content
	}{
		{"not an array", `"{\"children\":[{\"text\":\"dddddddd-0000-4000-8000-000000000003\"}]}"`},
		{"empty array", `"[]"`},
		{"missing children", `"[{}]"`},
		{"non-string text", `"[{\"children\":[{\"text\":123}]}]"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := graph.New()
			g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

			rule := themeRichTextRule()
			propertiesJSON := `jsonencode({"theme": {"value": "useThemeId"}, "themeId": {"value": ` + tt.themeIDValueJ + `}, "nodeTitle": {"value": "Theme Node"}})`

			attrs := map[string]interface{}{
				"graph_data": map[string]interface{}{
					"elements": map[string]interface{}{
						"nodes": map[string]interface{}{
							"node1": map[string]interface{}{
								"data": map[string]interface{}{
									"properties": RawHCLValue(propertiesJSON),
								},
							},
						},
					},
				},
			}

			resourceData := &ResourceData{
				ResourceType: "pingone_davinci_flow",
				ID:           "parent-flow-id",
				Label:        "pingcli__Parent-Flow",
				Attributes:   attrs,
			}

			exportedData := &ExportedResourceData{
				ResourceType: "pingone_davinci_flow",
				Definition:   testResourceDef("pingone_davinci_flow"),
				Resources:    []*ResourceData{resourceData},
			}

			originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

			var fallbackVars []FallbackVariable
			require.NotPanics(t, func() {
				fallbackVars = ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})
			})

			resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

			if resolvedValue != originalValue {
				t.Errorf("expected value unchanged for malformed wrapper %q, got: %s", tt.name, resolvedValue)
			}
			if len(fallbackVars) != 0 {
				t.Errorf("expected 0 FallbackVariables for malformed wrapper %q, got %d", tt.name, len(fallbackVars))
			}

			deps := g.GetDependencies("pingone_davinci_flow", "parent-flow-id")
			if len(deps) != 0 {
				t.Errorf("expected no graph edges for malformed wrapper %q, got %d", tt.name, len(deps))
			}
		})
	}
}

// TestResolveEmbeddedReferences_UUIDFormatGuard_NoOpPerStrategy verifies that
// the unconditional UUID-format guard rejects a non-UUID-shaped extracted
// value identically to "no value found" — regardless of Strategy.
func TestResolveEmbeddedReferences_UUIDFormatGuard_NoOpPerStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
	}{
		{"default strategy", ""},
		{"reference_with_fallback strategy", "reference_with_fallback"},
		{"variable strategy", "variable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := graph.New()
			// Register a resource keyed by the literal sentinel string so that,
			// absent the format guard, "reference"/"reference_with_fallback"
			// strategies would otherwise resolve it as a direct graph hit.
			g.AddResource("pingone_branding_theme", "activeTheme", "pingcli__Should_Not_Resolve")
			g.AddResource("pingone_davinci_flow", "parent-flow-id", "pingcli__Parent-Flow")

			rule := EmbeddedReferenceRule{
				ResourceType:       "pingone_davinci_flow",
				AttributePath:      "graph_data.elements.nodes.*.data.properties",
				TargetResourceType: "pingone_branding_theme",
				JSONKeyPath:        "theme.value",
				ReferenceField:     "id",
				Strategy:           tt.strategy,
				VariablePrefix:     "davinci_theme",
				VariableNamingPath: "nodeTitle.value",
			}

			propertiesJSON := `jsonencode({"theme": {"value": "activeTheme"}, "nodeTitle": {"value": "Theme Node"}})`

			attrs := map[string]interface{}{
				"graph_data": map[string]interface{}{
					"elements": map[string]interface{}{
						"nodes": map[string]interface{}{
							"node1": map[string]interface{}{
								"data": map[string]interface{}{
									"properties": RawHCLValue(propertiesJSON),
								},
							},
						},
					},
				},
			}

			resourceData := &ResourceData{
				ResourceType: "pingone_davinci_flow",
				ID:           "parent-flow-id",
				Label:        "pingcli__Parent-Flow",
				Attributes:   attrs,
			}

			exportedData := &ExportedResourceData{
				ResourceType: "pingone_davinci_flow",
				Definition:   testResourceDef("pingone_davinci_flow"),
				Resources:    []*ResourceData{resourceData},
			}

			originalValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

			fallbackVars := ResolveEmbeddedReferences([]*ExportedResourceData{exportedData}, g, []EmbeddedReferenceRule{rule})

			resolvedValue := attrs["graph_data"].(map[string]interface{})["elements"].(map[string]interface{})["nodes"].(map[string]interface{})["node1"].(map[string]interface{})["data"].(map[string]interface{})["properties"].(RawHCLValue)

			if resolvedValue != originalValue {
				t.Errorf("expected value unchanged for non-UUID sentinel under strategy %q, got: %s", tt.strategy, resolvedValue)
			}
			if len(fallbackVars) != 0 {
				t.Errorf("expected 0 FallbackVariables for non-UUID sentinel under strategy %q, got %d", tt.strategy, len(fallbackVars))
			}

			deps := g.GetDependencies("pingone_davinci_flow", "parent-flow-id")
			if len(deps) != 0 {
				t.Errorf("expected no graph edges for non-UUID sentinel under strategy %q, got %d", tt.strategy, len(deps))
			}
		})
	}
}
