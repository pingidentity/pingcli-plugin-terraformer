package pingone

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── API dispatch tests ──────────────────────────────────────────

func TestSupportedResourceTypes(t *testing.T) {
	expected := []string{
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
		"pingone_davinci_connector_instance",
		"pingone_davinci_flow",
		"pingone_davinci_flow_deploy",
		"pingone_davinci_flow_enable",
		"pingone_davinci_variable",
		"pingone_environment",
	}
	assert.Equal(t, expected, SupportedResourceTypes())
}

func TestIsSupportedTrue(t *testing.T) {
	for _, rt := range SupportedResourceTypes() {
		assert.True(t, isSupported(rt), "expected %s to be supported", rt)
	}
}

func TestIsSupportedFalse(t *testing.T) {
	assert.False(t, isSupported("unsupported_type"))
}

// ── Custom handler dispatch tests ──────────────────────────────

func TestRegisteredHandlerNames(t *testing.T) {
	names := RegisteredHandlerNames()
	// No custom handlers are currently registered (flow stubs removed).
	assert.Empty(t, names)
}

func TestRegisteredTransformNames(t *testing.T) {
	names := RegisteredTransformNames()
	expected := []string{
		"handleConnectorProperties",
	}
	for _, name := range expected {
		assert.Contains(t, names, name, "missing transform: %s", name)
	}
	// Verify flow stubs are no longer registered.
	assert.NotContains(t, names, "handleFlowSettings", "flow stubs should be removed")
	assert.NotContains(t, names, "handleFlowGraphData", "flow stubs should be removed")
	assert.NotContains(t, names, "handleFlowInputSchema", "flow stubs should be removed")
	assert.NotContains(t, names, "handleFlowOutputSchema", "flow stubs should be removed")
}

func TestRegisterCustomHandlersLoadsAll(t *testing.T) {
	reg := core.NewCustomHandlerRegistry()
	RegisterCustomHandlers(reg)

	// No custom handlers registered.
	assert.False(t, reg.HasHandler("generateFlowHCL"), "flow handler should be removed")

	// Only connector properties transform remains.
	assert.True(t, reg.HasTransform("handleConnectorProperties"))
	assert.False(t, reg.HasTransform("handleFlowSettings"), "flow stubs should be removed")
}

func TestHandleConnectorPropertiesRealTransform(t *testing.T) {
	reg := core.NewCustomHandlerRegistry()
	RegisterCustomHandlers(reg)

	fn, err := reg.GetTransform("handleConnectorProperties")
	require.NoError(t, err)

	// With nil value, it returns nil (no properties).
	result, err := fn(nil, nil, &schema.AttributeDefinition{Name: "properties"}, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}
