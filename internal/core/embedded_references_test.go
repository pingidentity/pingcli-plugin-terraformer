package core

import (
	"strings"
	"testing"

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
	g.AddResource("pingone_davinci_flow", "flow-abc123", "pingcli__My-0020-Flow")
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
							"properties": RawHCLValue("jsonencode({\n  \"nodeTitle\": {\n    \"value\": \"Sign On Flow\"\n  },\n  \"subFlowId\": {\n    \"value\": {\n      \"label\": \"My Flow\",\n      \"value\": \"flow-abc123\"\n    }\n  }\n})"),
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
	if strings.Contains(string(resolvedValue), "\"flow-abc123\"") {
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
	g.AddResource("pingone_davinci_flow", "flow-sub1", "pingcli__Sub-0020-Flow-1")
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
							"properties": RawHCLValue("jsonencode({\n  \"subFlowId\": {\n    \"value\": {\n      \"label\": \"Sub Flow 1\",\n      \"value\": \"flow-sub1\"\n    }\n  }\n})"),
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
	g.AddResource("pingone_davinci_flow", "flow-a", "pingcli__Flow-A")
	g.AddResource("pingone_davinci_flow", "flow-b", "pingcli__Flow-B")
	g.AddResource("pingone_davinci_flow", "flow-c", "pingcli__Flow-C")
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
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-a"}}})`),
						},
					},
					"nodeB": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-b"}}})`),
						},
					},
					"nodeC": map[string]interface{}{
						"data": map[string]interface{}{
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-c"}}})`),
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
	g.AddResource("pingone_davinci_connector", "conn-custom1", "pingcli__Custom-Connector")
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
							"properties": RawHCLValue(`jsonencode({"customConnectorId": {"value": "conn-custom1"}})`),
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
	g.AddResource("pingone_davinci_flow", "flow-sub1", "pingcli__Sub-Flow")
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
							"properties": RawHCLValue(`jsonencode({"subFlowId": {"value": {"value": "flow-sub1"}}})`),
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
