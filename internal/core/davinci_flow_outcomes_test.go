package core_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	pingone "github.com/pingidentity/pingone-go-client/pingone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	hclformatter "github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters/hcl"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// davinciFlowRegistry loads the real flow.yaml definition (not a synthetic
// mini-schema) so these tests catch drift between the AdditionalProperties
// fallback (Task 1) and the actual outcomes attribute declared in
// definitions/pingone/davinci/flow.yaml (Task 2).
func davinciFlowRegistry(t *testing.T) *schema.Registry {
	t.Helper()
	registry := schema.NewRegistry()
	require.NoError(t, registry.LoadPlatform("../../definitions", "pingone"))
	return registry
}

// buildFlowResponse constructs a minimal but valid *pingone.DaVinciFlowResponse
// with two nodes in GraphData.Elements.Nodes: one whose Data carries a typed
// Outcomes slice (mirroring the sample flow's esqd1w6k6h node, requirements.md
// "Sample flow evidence"), and one without any outcomes (mirroring
// frbglr02tp), so a single processed resource proves both the presence and
// absence cases end-to-end through the real definition.
//
// pingone-go-client v0.12.0 added a typed Outcomes field to
// DaVinciFlowGraphDataResponseElementsNodeData (terraform-provider-pingone#1342,
// pingone-go-client#86, CDI-1370 all shipped/landed) -- outcomes is no longer
// SDK-untyped, so this constructs the field directly rather than stuffing it
// into AdditionalProperties. The AdditionalProperties fallback added in
// internal/core/processor.go for this field is now dormant (verified by the
// regression tests in processor_test.go), staying available for whatever
// SDK field next lags its JSON shape.
func buildFlowResponse() *pingone.DaVinciFlowResponse {
	links := pingone.NewDaVinciFlowResponseLinks(
		*pingone.NewJSONHALLink("https://example.com/environment"),
		*pingone.NewJSONHALLink("https://example.com/self"),
		*pingone.NewJSONHALLink("https://example.com/connectorInstances"),
		*pingone.NewJSONHALLink("https://example.com/connectors"),
		*pingone.NewJSONHALLink("https://example.com/flow.deploy"),
		*pingone.NewJSONHALLink("https://example.com/flow.clone"),
		*pingone.NewJSONHALLink("https://example.com/flow.enabled"),
		*pingone.NewJSONHALLink("https://example.com/version"),
	)
	environment := pingone.NewResourceRelationshipReadOnly(uuid.MustParse("00000000-0000-0000-0000-000000000001"))

	nodeWithOutcomes := pingone.NewDaVinciFlowGraphDataResponseElementsNode(
		func() pingone.DaVinciFlowGraphDataResponseElementsNodeData {
			d := pingone.NewDaVinciFlowGraphDataResponseElementsNodeData("esqd1w6k6h", "CONNECTION")
			d.SetLabel("Save/Resend")
			d.Outcomes = []pingone.DaVinciFlowGraphDataResponseElementsNodeDataOutcome{
				*pingone.NewDaVinciFlowGraphDataResponseElementsNodeDataOutcome("submit", "Save", "0qw160q8zo"),
				*pingone.NewDaVinciFlowGraphDataResponseElementsNodeDataOutcome("resend", "Didn't receive an email? Resend", "k0hv0wr75q"),
			}
			return *d
		}(),
		true, "", false, true,
		*pingone.NewDaVinciFlowGraphDataResponseElementsNodePosition(0, 0),
		false, true, false,
	)

	nodeWithoutOutcomes := pingone.NewDaVinciFlowGraphDataResponseElementsNode(
		func() pingone.DaVinciFlowGraphDataResponseElementsNodeData {
			d := pingone.NewDaVinciFlowGraphDataResponseElementsNodeData("frbglr02tp", "CONNECTION")
			d.SetLabel("Single Outcome Form")
			return *d
		}(),
		true, "", false, true,
		*pingone.NewDaVinciFlowGraphDataResponseElementsNodePosition(0, 0),
		false, true, false,
	)

	elements := pingone.NewDaVinciFlowGraphDataResponseElements(
		[]pingone.DaVinciFlowGraphDataResponseElementsNode{*nodeWithOutcomes, *nodeWithoutOutcomes},
	)

	graphData := pingone.NewDaVinciFlowGraphDataResponse(
		true,
		*elements,
		*pingone.NewDaVinciFlowGraphDataResponsePan(0, 0),
		true, true, true, 1.0,
	)

	flow := pingone.NewDaVinciFlowResponse(*links, *environment, "flow-1", "Test Flow")
	flow.SetGraphData(*graphData)
	return flow
}

// TestDaVinciFlowOutcomes_ProcessorPresentAndAbsent exercises the full
// flow.yaml definition (loaded via schema.Registry, not a synthetic schema)
// through ProcessResource, proving requirements.md acceptance criteria 1 and
// 2: a node with AdditionalProperties["outcomes"] produces
// graph_data.elements.nodes["<id>"].data.outcomes as an ordered list with
// id/result/label preserved verbatim, and a node with no outcomes key has no
// "outcomes" key in its data map at all (not an empty list, not null).
func TestDaVinciFlowOutcomes_ProcessorPresentAndAbsent(t *testing.T) {
	registry := davinciFlowRegistry(t)
	p := core.NewProcessor(registry)

	result, err := p.ProcessResource("pingone_davinci_flow", buildFlowResponse())
	require.NoError(t, err)

	graphData, ok := result.Attributes["graph_data"].(map[string]interface{})
	require.True(t, ok, "graph_data should be a map")
	elements, ok := graphData["elements"].(map[string]interface{})
	require.True(t, ok, "graph_data.elements should be a map")
	nodes, ok := elements["nodes"].(map[string]interface{})
	require.True(t, ok, "graph_data.elements.nodes should be a map")
	require.Len(t, nodes, 2)

	nodeWithOutcomes, ok := nodes["esqd1w6k6h"].(map[string]interface{})
	require.True(t, ok, "node esqd1w6k6h should be present")
	dataWithOutcomes, ok := nodeWithOutcomes["data"].(map[string]interface{})
	require.True(t, ok)

	outcomes, ok := dataWithOutcomes["outcomes"].([]interface{})
	require.True(t, ok, "outcomes should be a []interface{}, got %T", dataWithOutcomes["outcomes"])
	require.Len(t, outcomes, 2)

	first, ok := outcomes[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "0qw160q8zo", first["id"])
	assert.Equal(t, "submit", first["result"])
	assert.Equal(t, "Save", first["label"])

	second, ok := outcomes[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "k0hv0wr75q", second["id"])
	assert.Equal(t, "resend", second["result"])
	assert.Equal(t, "Didn't receive an email? Resend", second["label"])

	nodeWithoutOutcomes, ok := nodes["frbglr02tp"].(map[string]interface{})
	require.True(t, ok, "node frbglr02tp should be present")
	dataWithoutOutcomes, ok := nodeWithoutOutcomes["data"].(map[string]interface{})
	require.True(t, ok)

	_, hasOutcomes := dataWithoutOutcomes["outcomes"]
	assert.False(t, hasOutcomes, "node without an outcomes key must have no outcomes attribute at all -- not an empty list, not null")
}

// TestDaVinciFlowOutcomes_HCLRendering verifies the HCL formatter, driven by
// the real flow.yaml definition, renders an outcomes = [ {...}, {...} ] block
// for the node that has outcomes, with id/result/label preserved verbatim,
// and emits no "outcomes" attribute at all for the node that doesn't --
// covering requirements.md acceptance criterion 1 and 2 at the rendered-HCL
// layer, not just the intermediate representation.
func TestDaVinciFlowOutcomes_HCLRendering(t *testing.T) {
	registry := davinciFlowRegistry(t)
	p := core.NewProcessor(registry)
	def, err := registry.Get("pingone_davinci_flow")
	require.NoError(t, err)

	result, err := p.ProcessResource("pingone_davinci_flow", buildFlowResponse())
	require.NoError(t, err)
	result.Label = "pingcli__Test-Flow"

	f := hclformatter.NewFormatter()
	output, err := f.Format(result, def, hclformatter.FormatOptions{SkipDependencies: true})
	require.NoError(t, err)

	// Node with outcomes: block present with id/result/label preserved verbatim.
	assert.Contains(t, output, "outcomes")
	assert.Contains(t, output, `"0qw160q8zo"`)
	assert.Contains(t, output, `"submit"`)
	assert.Contains(t, output, `"Save"`)
	assert.Contains(t, output, `"k0hv0wr75q"`)
	assert.Contains(t, output, `"resend"`)
	assert.Contains(t, output, `"Didn't receive an email? Resend"`)

	// Node without outcomes: no outcomes attribute rendered for that node's
	// data block. Since nested_attributes rendering is keyed by presence in
	// the map (nestedObjectTokens skips missing keys), the absence is already
	// proven by the processor test above; here we confirm the overall output
	// contains exactly the one outcomes block (not two, and not an empty one
	// for frbglr02tp).
	assert.Equal(t, 1, strings.Count(output, "outcomes = ["),
		"exactly one outcomes block should be rendered, only for the node that has outcomes")
}
