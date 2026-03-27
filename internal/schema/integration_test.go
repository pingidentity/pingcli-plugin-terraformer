package schema_test

import (
	"testing"
	"testing/fstest"

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

	// Should have loaded all definitions (7 enabled ones, 1 environment is disabled)
	assert.Equal(t, 7, registry.Count())

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

	// Should have loaded all definitions (7 enabled ones, 1 environment is disabled).
	assert.Equal(t, 7, registry.Count())

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
	assert.Len(t, defs, 7)

	// Verify each definition has valid metadata.
	for _, def := range defs {
		assert.Equal(t, "pingone", def.Metadata.Platform)
		assert.NotEmpty(t, def.Metadata.ResourceType)
	}
}

func TestRegistryLoadFromFSFiltersDisabledDefinitions(t *testing.T) {
	// Create an in-memory filesystem with mix of enabled, disabled, and default definitions
	mapFS := fstest.MapFS{
		"pingone/enabled.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_test_enabled
  api_type: Enabled
  name: Enabled Resource
  short_name: enabled
  version: 1.0
  enabled: true
api:
  sdk_package: test.package
  sdk_type: TestType
  list_method: TestList
  id_field: id
  name_field: name
  pagination_type: cursor
attributes: []
dependencies: {}
variables: {}
`),
		},
		"pingone/disabled.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_test_disabled
  api_type: Disabled
  name: Disabled Resource
  short_name: disabled
  version: 1.0
  enabled: false
api:
  sdk_package: test.package
  sdk_type: TestType
  list_method: TestList
  id_field: id
  name_field: name
  pagination_type: cursor
attributes: []
dependencies: {}
variables: {}
`),
		},
		"pingone/default.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_test_default
  api_type: Default
  name: Default Resource
  short_name: default
  version: 1.0
api:
  sdk_package: test.package
  sdk_type: TestType
  list_method: TestList
  id_field: id
  name_field: name
  pagination_type: cursor
attributes: []
dependencies: {}
variables: {}
`),
		},
	}

	registry := schema.NewRegistry()
	err := registry.LoadFromFS(mapFS, "pingone")
	require.NoError(t, err)

	// Should register only enabled and default definitions, not disabled ones
	assert.Equal(t, 2, registry.Count(), "should register 2 definitions (enabled and default), excluding disabled")

	// Verify specific definitions
	_, err = registry.Get("pingone_test_enabled")
	assert.NoError(t, err, "enabled definition should be registered")

	_, err = registry.Get("pingone_test_default")
	assert.NoError(t, err, "default (nil enabled) definition should be registered")

	_, err = registry.Get("pingone_test_disabled")
	assert.Error(t, err, "disabled definition should NOT be registered")
}

func TestRegistryListByPlatformExcludesDisabled(t *testing.T) {
	// Verify that ListByPlatform only returns enabled definitions
	mapFS := fstest.MapFS{
		"pingone/res1.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_res1
  api_type: Type1
  name: Resource 1
  short_name: res1
  version: 1.0
  enabled: true
api:
  sdk_package: test.package
  sdk_type: TestType
  list_method: TestList
  id_field: id
  name_field: name
  pagination_type: cursor
attributes: []
dependencies: {}
variables: {}
`),
		},
		"pingone/res2.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_res2
  api_type: Type2
  name: Resource 2
  short_name: res2
  version: 1.0
  enabled: false
api:
  sdk_package: test.package
  sdk_type: TestType
  list_method: TestList
  id_field: id
  name_field: name
  pagination_type: cursor
attributes: []
dependencies: {}
variables: {}
`),
		},
		"pingone/res3.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_res3
  api_type: Type3
  name: Resource 3
  short_name: res3
  version: 1.0
api:
  sdk_package: test.package
  sdk_type: TestType
  list_method: TestList
  id_field: id
  name_field: name
  pagination_type: cursor
attributes: []
dependencies: {}
variables: {}
`),
		},
	}

	registry := schema.NewRegistry()
	err := registry.LoadFromFS(mapFS, "pingone")
	require.NoError(t, err)

	// Get all definitions for the platform
	defs := registry.ListByPlatform("pingone")

	// Should only have enabled and default definitions (res1 and res3), not disabled (res2)
	assert.Len(t, defs, 2, "ListByPlatform should return only enabled and default definitions")

	resourceTypes := make(map[string]bool)
	for _, def := range defs {
		resourceTypes[def.Metadata.ResourceType] = true
	}

	assert.True(t, resourceTypes["pingone_res1"], "should contain res1 (enabled)")
	assert.True(t, resourceTypes["pingone_res3"], "should contain res3 (default)")
	assert.False(t, resourceTypes["pingone_res2"], "should NOT contain res2 (disabled)")
}
