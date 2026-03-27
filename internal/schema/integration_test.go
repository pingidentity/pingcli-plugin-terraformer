package schema_test

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/definitions"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadVariableDefinition(t *testing.T) {
	loader := schema.NewLoader()

	// Load the actual variable.yaml definition
	def, err := loader.LoadDefinition("../../definitions/pingone/davinci/variable.yaml")
	require.NoError(t, err)
	assert.NotNil(t, def)

	// Verify metadata
	assert.Equal(t, "pingone", def.Metadata.Platform)
	assert.Equal(t, "pingone_davinci_variable", def.Metadata.ResourceType)
	assert.Equal(t, "Variable", def.Metadata.APIType)
	assert.Equal(t, "DaVinci Variable", def.Metadata.Name)
	assert.Equal(t, "variable", def.Metadata.ShortName)
	assert.Equal(t, "1.0", def.Metadata.Version)

	// Verify API configuration
	assert.Equal(t, "github.com/pingidentity/pingone-go-client/davinci", def.API.SDKPackage)
	assert.Equal(t, "Variable", def.API.SDKType)
	assert.Equal(t, "EnvironmentVariables.GetAll", def.API.ListMethod)
	assert.Equal(t, "id", def.API.IDField)
	assert.Equal(t, "name", def.API.NameField)
	assert.Equal(t, "cursor", def.API.PaginationType)

	// Verify attributes exist
	assert.Greater(t, len(def.Attributes), 0, "should have attributes")

	// Find specific attributes
	var foundID, foundName, foundValue bool
	for _, attr := range def.Attributes {
		switch attr.Name {
		case "ID":
			foundID = true
			assert.True(t, attr.Computed)
		case "Name":
			foundName = true
			assert.True(t, attr.Required)
		case "Value":
			foundValue = true
			assert.True(t, attr.VariableEligible)
		}
	}

	assert.True(t, foundID, "should have ID attribute")
	assert.True(t, foundName, "should have Name attribute")
	assert.True(t, foundValue, "should have Value attribute")

	// Verify dependencies
	assert.Equal(t, "{env_id}/{resource_id}", def.Dependencies.ImportIDFormat)

	// Verify variable extraction rules
	assert.Len(t, def.Variables.EligibleAttributes, 1)
	assert.Equal(t, "value", def.Variables.EligibleAttributes[0].AttributePath)
	assert.Equal(t, "davinci_variable_", def.Variables.EligibleAttributes[0].VariablePrefix)
}

func TestRegistryLoadPlatform(t *testing.T) {
	registry := schema.NewRegistry()

	// Load all PingOne definitions
	err := registry.LoadPlatform("../../definitions", "pingone")
	require.NoError(t, err)

	// Should have loaded all definitions
	assert.Equal(t, 8, registry.Count())

	// Get the variable definition
	def, err := registry.Get("pingone_davinci_variable")
	require.NoError(t, err)
	assert.Equal(t, "variable", def.Metadata.ShortName)
}

func TestRegistryLoadFromFS(t *testing.T) {
	registry := schema.NewRegistry()

	// Load all PingOne definitions from embedded FS.
	err := registry.LoadFromFS(definitions.FS, "pingone")
	require.NoError(t, err)

	// Should have loaded all definitions.
	assert.Equal(t, 8, registry.Count())

	// Verify a specific definition loaded correctly.
	def, err := registry.Get("pingone_davinci_variable")
	require.NoError(t, err)
	assert.Equal(t, "variable", def.Metadata.ShortName)
	assert.Equal(t, "pingone", def.Metadata.Platform)
}

func TestLoaderLoadFromFS(t *testing.T) {
	loader := schema.NewLoader()

	// Load from embedded FS.
	defs, err := loader.LoadFromFS(definitions.FS, "pingone")
	require.NoError(t, err)
	assert.Len(t, defs, 8)

	// Verify each definition has valid metadata.
	for _, def := range defs {
		assert.Equal(t, "pingone", def.Metadata.Platform)
		assert.NotEmpty(t, def.Metadata.ResourceType)
	}
}
