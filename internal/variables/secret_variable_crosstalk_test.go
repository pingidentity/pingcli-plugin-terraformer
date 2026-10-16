package variables_test

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/definitions"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/platform/pingone"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretVariablePipeline_TwoSecretsNoCrossWire reproduces the live-export
// bug where two secret variables both rendered var.davinci_variable_secret_b_value:
// the conditional_defaults SetValue map is a single shared map[string]interface{}
// value from the parsed YAML. Assigning it into each resource's Attributes map
// aliases the same underlying map across all resources — the last resource to
// have its ResolvedReference injected wins for every resource of the type.
func TestSecretVariablePipeline_TwoSecretsNoCrossWire(t *testing.T) {
	registry := schema.NewRegistry()
	require.NoError(t, registry.LoadFromFS(definitions.FS, "pingone"))
	customReg := core.NewCustomHandlerRegistry()
	pingone.RegisterCustomHandlers(customReg)
	proc := core.NewProcessor(registry, core.WithCustomHandlers(customReg))

	mutable := true
	company := "company"

	// Process a LIST of two secret variables, like the live orchestrator does.
	list := []*mockDaVinciVariable{
		{ID: "id-a", Name: "secret-a", Context: &company, DataType: "secret", Mutable: &mutable},
		{ID: "id-b", Name: "secret-b", Context: &company, DataType: "secret", Mutable: &mutable},
	}

	results, err := proc.ProcessResourceList("pingone_davinci_variable", list)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Simulate cmd/export.go's injection loop on each resource.
	ext := variables.NewVariableExtractor(registry)
	names := make(map[string]string) // resource name -> referenced variable name
	for _, rd := range results {
		extracted, extErr := ext.Extract("pingone_davinci_variable", rd.Attributes, rd.Name)
		require.NoError(t, extErr)
		require.Len(t, extracted, 1)

		ev := extracted[0]
		names[rd.Name] = ev.VariableName

		ref := core.ResolvedReference{IsVariable: true, VariableName: ev.VariableName}
		if ev.CurrentValue != nil {
			if s, ok := ev.CurrentValue.(string); ok {
				ref.OriginalValue = s
			}
		}
		if m, ok := rd.Attributes[ev.AttributePath].(map[string]interface{}); ok && len(m) == 1 {
			for k := range m {
				m[k] = ref
			}
		}
	}

	// Each resource must reference its OWN variable.
	assert.Equal(t, "davinci_variable_secret_a_value", names["secret-a"])
	assert.Equal(t, "davinci_variable_secret_b_value", names["secret-b"])

	// And the in-attribute reference must match the per-resource name — the
	// aliasing bug made both resources carry the LAST injected reference.
	for _, rd := range results {
		m := rd.Attributes["value"].(map[string]interface{})
		ref := m["secret_string"].(core.ResolvedReference)
		assert.Equal(t, names[rd.Name], ref.VariableName,
			"resource %s must reference its own variable", rd.Name)
	}
}
