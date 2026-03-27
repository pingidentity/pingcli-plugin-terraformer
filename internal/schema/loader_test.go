package schema

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoaderLoadFromFSFiltersDisabledDefinitions(t *testing.T) {
	// Create an in-memory filesystem with definitions
	mapFS := fstest.MapFS{
		"pingone/enabled.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_test_enabled
  api_type: TestEnabled
  name: Test Enabled
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
  api_type: TestDisabled
  name: Test Disabled
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
  api_type: TestDefault
  name: Test Default
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

	loader := NewLoader()
	defs, err := loader.LoadFromFS(mapFS, "pingone")
	require.NoError(t, err)

	// Should load only enabled and default (nil) definitions, not disabled ones
	// Expected: enabled.yaml and default.yaml
	assert.Len(t, defs, 2, "should load 2 definitions (enabled and default), excluding disabled")

	resourceTypes := make(map[string]*ResourceDefinition)
	for _, def := range defs {
		resourceTypes[def.Metadata.ResourceType] = def
	}

	assert.Contains(t, resourceTypes, "pingone_test_enabled", "should contain enabled definition")
	assert.Contains(t, resourceTypes, "pingone_test_default", "should contain default (nil) definition")
	assert.NotContains(t, resourceTypes, "pingone_test_disabled", "should NOT contain disabled definition")
}

func TestLoaderLoadFromFSOnlyEnabledTrue(t *testing.T) {
	// Verify that only explicitly enabled (true) definitions are loaded, along with defaults (nil)
	mapFS := fstest.MapFS{
		"pingone/explicitly_true.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_explicit_true
  api_type: ExplicitTrue
  name: Explicit True
  short_name: explicit_true
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
		"pingone/explicitly_false.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_explicit_false
  api_type: ExplicitFalse
  name: Explicit False
  short_name: explicit_false
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
	}

	loader := NewLoader()
	defs, err := loader.LoadFromFS(mapFS, "pingone")
	require.NoError(t, err)

	assert.Len(t, defs, 1, "should load only 1 definition (explicitly enabled)")
	assert.Equal(t, "pingone_explicit_true", defs[0].Metadata.ResourceType)
}

func TestLoaderLoadFromFSDefaultBehavior(t *testing.T) {
	// When enabled is omitted (nil), the definition should load normally (default behavior is enabled)
	mapFS := fstest.MapFS{
		"pingone/no_enabled_field.yaml": &fstest.MapFile{
			Data: []byte(`
metadata:
  platform: pingone
  resource_type: pingone_no_enabled
  api_type: NoEnabled
  name: No Enabled Field
  short_name: no_enabled
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

	loader := NewLoader()
	defs, err := loader.LoadFromFS(mapFS, "pingone")
	require.NoError(t, err)

	assert.Len(t, defs, 1, "should load definition when enabled field is omitted")
	assert.Nil(t, defs[0].Metadata.Enabled, "enabled should be nil when omitted")
}

func TestLoaderLoadFromDirectoryFiltersDisabledDefinitions(t *testing.T) {
	// Test that LoadFromDirectory respects the enabled field by creating actual YAML files
	tempDir := t.TempDir()

	// Write an enabled definition
	enabledYAML := []byte(`
metadata:
  platform: pingone
  resource_type: pingone_enabled_resource
  api_type: EnabledResource
  name: Enabled Resource
  short_name: enabled_resource
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
`)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "enabled.yaml"), enabledYAML, 0o644))

	// Write a disabled definition
	disabledYAML := []byte(`
metadata:
  platform: pingone
  resource_type: pingone_disabled_resource
  api_type: DisabledResource
  name: Disabled Resource
  short_name: disabled_resource
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
`)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "disabled.yaml"), disabledYAML, 0o644))

	// Write a default (no enabled field) definition
	defaultYAML := []byte(`
metadata:
  platform: pingone
  resource_type: pingone_default_resource
  api_type: DefaultResource
  name: Default Resource
  short_name: default_resource
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
`)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "default.yaml"), defaultYAML, 0o644))

	loader := NewLoader()
	defs, err := loader.LoadFromDirectory(tempDir)
	require.NoError(t, err)

	// Should load only enabled and default (nil) definitions, not disabled ones
	assert.Len(t, defs, 2, "should load 2 definitions (enabled and default), excluding disabled")

	resourceTypes := make(map[string]*ResourceDefinition)
	for _, def := range defs {
		resourceTypes[def.Metadata.ResourceType] = def
	}

	assert.Contains(t, resourceTypes, "pingone_enabled_resource", "should contain enabled definition")
	assert.Contains(t, resourceTypes, "pingone_default_resource", "should contain default (nil) definition")
	assert.NotContains(t, resourceTypes, "pingone_disabled_resource", "should NOT contain disabled definition")
}
