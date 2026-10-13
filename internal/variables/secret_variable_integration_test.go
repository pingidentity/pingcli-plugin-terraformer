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

// mockDaVinciVariable mirrors the SDK's DaVinciVariableResponse for the fields
// the variable.yaml definition reads. Value mirrors the SDK union struct —
// nil for secret variables, a *string for string-typed values.
type mockDaVinciVariable struct {
	ID       string
	Name     string
	Context  *string
	DataType string
	Mutable  *bool
	Min      *int
	Max      *int
	FlowID   *string
	Value    *string
}

// TestSecretVariablePipeline is the end-to-end check for secret DaVinci
// variable export: the real variable.yaml definition drives the processor and
// extractor, and a data_type=secret resource must come out with
// value = { secret_string = var.davinci_variable_<name>_value } while a plain
// string variable keeps its exportable value.
func TestSecretVariablePipeline(t *testing.T) {
	registry := schema.NewRegistry()
	require.NoError(t, registry.LoadFromFS(definitions.FS, "pingone"))
	customReg := core.NewCustomHandlerRegistry()
	pingone.RegisterCustomHandlers(customReg)
	proc := core.NewProcessor(registry, core.WithCustomHandlers(customReg))

	mutable := true
	flowCtx := "company"
	mock := &mockDaVinciVariable{
		ID:       "37b88458-74c3-4c4d-aa42-295e29c29f2b",
		Name:     "secret-a",
		Context:  &flowCtx,
		DataType: "secret",
		Mutable:  &mutable,
	}

	result, err := proc.ProcessResource("pingone_davinci_variable", mock)
	require.NoError(t, err)
	assert.Equal(t, "secret", result.Attributes["data_type"])

	// The conditional default re-emitted the suppressed block with the
	// provider-required secret_string key.
	valueMap, ok := result.Attributes["value"].(map[string]interface{})
	require.True(t, ok, "secret variable must have a value block")
	assert.Equal(t, "REPLACED_BY_VARIABLE_EXTRACTOR", valueMap["secret_string"])

	// Extractor: the value variable is secret + sensitive, and the injection
	// contract (single-key tdb map) holds for cmd/export.go.
	ext := variables.NewVariableExtractor(registry)
	vars, err := ext.Extract("pingone_davinci_variable", result.Attributes, result.Name)
	require.NoError(t, err)
	require.Len(t, vars, 1)

	v := vars[0]
	assert.True(t, v.IsSecret)
	assert.True(t, v.Sensitive)
	assert.Equal(t, "value", v.AttributePath)
	assert.Equal(t, "davinci_variable_secret_a_value", v.VariableName)
	assert.Equal(t, "REPLACED_BY_VARIABLE_EXTRACTOR", v.CurrentValue)
}

func TestSecretVariablePipeline_PlainStringUnchanged(t *testing.T) {
	registry := schema.NewRegistry()
	require.NoError(t, registry.LoadFromFS(definitions.FS, "pingone"))
	customReg := core.NewCustomHandlerRegistry()
	pingone.RegisterCustomHandlers(customReg)
	proc := core.NewProcessor(registry, core.WithCustomHandlers(customReg))

	mutable := true
	flowCtx := "company"
	value := "abcd"
	mock := &mockDaVinciVariable{
		ID:       "11111111-2222-4333-8444-555555555555",
		Name:     "string-a",
		Context:  &flowCtx,
		DataType: "string",
		Mutable:  &mutable,
		Value:    &value,
	}

	result, err := proc.ProcessResource("pingone_davinci_variable", mock)
	require.NoError(t, err)

	valueMap, ok := result.Attributes["value"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "abcd", valueMap["string"], "plain variables keep their runtime-typed key")
	assert.Nil(t, valueMap["secret_string"])

	ext := variables.NewVariableExtractor(registry)
	vars, err := ext.Extract("pingone_davinci_variable", result.Attributes, result.Name)
	require.NoError(t, err)
	require.Len(t, vars, 1)

	v := vars[0]
	assert.False(t, v.IsSecret, "plain variable values keep their tfvars value")
	assert.False(t, v.Sensitive)
	assert.Equal(t, "abcd", v.CurrentValue)
}
