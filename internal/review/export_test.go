package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
)

func closureSeeds(types ...string) []string { return types }

func TestComputeClosure_LeafType(t *testing.T) {
	// pingone_group references pingone_environment (always) and
	// pingone_population (via the user_filter embedded rule is absent here,
	// but the YAML declares the reference) — the population pulls
	// branding_theme, which references pingone_image (undefined → fallback).
	result, err := ComputeClosure(closureSeeds("pingone_group"), nil)
	require.NoError(t, err)

	assert.Contains(t, result.Types, "pingone_group")
	assert.Contains(t, result.Types, "pingone_environment")
	assert.Contains(t, result.Types, "pingone_population")
	assert.Empty(t, result.SkippedLive)
	assert.Contains(t, result.FallbackTargets, "pingone_image")
}

func TestComputeClosure_DependencyChain(t *testing.T) {
	// pingone_resource_scope references pingone_resource, which references
	// pingone_environment — all three must be in the closure.
	result, err := ComputeClosure(closureSeeds("pingone_resource_scope"), nil)
	require.NoError(t, err)

	for _, want := range []string{"pingone_resource_scope", "pingone_resource", "pingone_environment"} {
		assert.Contains(t, result.Types, want)
	}
	assert.GreaterOrEqual(t, len(result.Types), 3)
}

func TestComputeClosure_AlwaysIncludesEnvironment(t *testing.T) {
	result, err := ComputeClosure(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"pingone_environment"}, result.Types)
}

func TestComputeClosure_DaVinciExcluded(t *testing.T) {
	// pingone_davinci_variable is a DaVinci type: it must land in
	// SkippedLive, not Types.
	result, err := ComputeClosure(closureSeeds("pingone_davinci_variable"), nil)
	require.NoError(t, err)

	assert.NotContains(t, result.Types, "pingone_davinci_variable")
	require.Contains(t, result.SkippedLive, "pingone_davinci_variable")
	assert.Contains(t, result.SkippedLive["pingone_davinci_variable"], "DaVinci")
	assert.Contains(t, result.Types, "pingone_environment")
}

func TestComputeClosure_DaVinciDepTransitive(t *testing.T) {
	// A type whose closure contains a DaVinci member: the DaVinci member is
	// skipped but the review type still exports.
	result, err := ComputeClosure(closureSeeds("pingone_application_flow_policy_assignment"), nil)
	require.NoError(t, err)

	assert.Contains(t, result.Types, "pingone_application_flow_policy_assignment")
	assert.NotEmpty(t, result.SkippedLive, "flow-policy assignment's DaVinci dependency must be skipped live")
}

func TestComputeClosure_MissingTargetIsFallback(t *testing.T) {
	// pingone_application references pingone_key (no definition exists) and
	// pingone_key_rotation_policy (no definition) — both are fallback targets.
	result, err := ComputeClosure(closureSeeds("pingone_application"), nil)
	require.NoError(t, err)

	assert.Contains(t, result.Types, "pingone_application")
	assert.Contains(t, result.FallbackTargets, "pingone_key")
	assert.Contains(t, result.FallbackTargets, "pingone_key_rotation_policy")
}

func TestComputeClosure_EmbeddedRuleTargetPulled(t *testing.T) {
	// A rule on pingone_group targeting pingone_population pulls the
	// population into the closure.
	rules := []core.EmbeddedReferenceRule{
		{
			ResourceType:       "pingone_group",
			AttributePath:      "user_filter",
			TargetResourceType: "pingone_population",
			ReferenceField:     "id",
			PlainStringPattern: `population\.id eq "(...)"`,
			Strategy:           "reference_with_fallback",
		},
	}
	result, err := ComputeClosure(closureSeeds("pingone_group"), rules)
	require.NoError(t, err)

	assert.Contains(t, result.Types, "pingone_population")
	assert.NotContains(t, result.FallbackTargets, "pingone_population")
}

func TestComputeClosure_RegistryContainsExactlyClosure(t *testing.T) {
	result, err := ComputeClosure(closureSeeds("pingone_resource_attribute"), nil)
	require.NoError(t, err)

	// Registry must contain every closure type and nothing more.
	assert.Equal(t, len(result.Types), result.Registry.Count())
	for _, rt := range result.Types {
		_, err := result.Registry.Get(rt)
		assert.NoError(t, err, "registry missing %s", rt)
	}
}
