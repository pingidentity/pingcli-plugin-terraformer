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
		"pingone_application",
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
		"pingone_davinci_connector_instance",
		"pingone_davinci_flow",
		"pingone_davinci_flow_deploy",
		"pingone_davinci_flow_enable",
		"pingone_davinci_variable",
		"pingone_environment",
		"pingone_group",
		"pingone_population",
		"pingone_resource",
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
	expected := []string{
		"handleFlowVariableDependencies",
	}
	for _, name := range expected {
		assert.Contains(t, names, name, "missing handler: %s", name)
	}
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

	// handleFlowVariableDependencies should be registered as a handler.
	assert.True(t, reg.HasHandler("handleFlowVariableDependencies"))

	// Only connector properties transform remains.
	assert.True(t, reg.HasTransform("handleConnectorProperties"))
	assert.False(t, reg.HasTransform("handleFlowSettings"), "flow stubs should be removed")
}

// ── Embedded reference rule dispatch tests ──────────────────────

// TestEmbeddedReferenceRulesRegistered confirms the pingone_branding_theme
// rules for theme.value (case 1) and themeId.value (case 3) are queued in
// embeddedRefRules, mirroring TestRegisteredHandlerNames/TestRegisteredTransformNames.
func TestEmbeddedReferenceRulesRegistered(t *testing.T) {
	reg := NewEmbeddedReferenceRegistry()
	rules := reg.Rules()

	var themeValueRule, themeIDValueRule *core.EmbeddedReferenceRule
	for i := range rules {
		r := &rules[i]
		if r.TargetResourceType != "pingone_branding_theme" {
			continue
		}
		switch r.JSONKeyPath {
		case "theme.value":
			themeValueRule = r
		case "themeId.value":
			themeIDValueRule = r
		}
	}

	require.NotNil(t, themeValueRule, "expected a pingone_branding_theme rule with JSONKeyPath \"theme.value\" to be registered")
	assert.Equal(t, "pingone_davinci_flow", themeValueRule.ResourceType)
	assert.Equal(t, "reference_with_fallback", themeValueRule.Strategy)
	assert.Equal(t, "davinci_theme", themeValueRule.VariablePrefix)
	assert.Equal(t, "nodeTitle.value", themeValueRule.VariableNamingPath)
	assert.Empty(t, themeValueRule.PreconditionKeyPath, "case 1 rule must have no precondition")
	assert.Empty(t, themeValueRule.UnwrapMode, "case 1 rule must not use rich-text unwrap")

	require.NotNil(t, themeIDValueRule, "expected a pingone_branding_theme rule with JSONKeyPath \"themeId.value\" to be registered")
	assert.Equal(t, "pingone_davinci_flow", themeIDValueRule.ResourceType)
	assert.Equal(t, "reference_with_fallback", themeIDValueRule.Strategy)
	assert.Equal(t, "davinci_theme", themeIDValueRule.VariablePrefix)
	assert.Equal(t, "nodeTitle.value", themeIDValueRule.VariableNamingPath)
	assert.Equal(t, "theme.value", themeIDValueRule.PreconditionKeyPath)
	assert.Equal(t, "useThemeId", themeIDValueRule.PreconditionValue)
	assert.Equal(t, "rich_text", themeIDValueRule.UnwrapMode)
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
