package core_test

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockVariable represents a mock API response for a DaVinci Variable
type MockVariable struct {
	ID          string
	Name        string
	Context     string
	DataType    string
	Mutable     bool
	DisplayName *string
	Min         *float32
	Max         *float32
	Value       *string
}

func TestProcessorBasic(t *testing.T) {
	// Create a minimal schema definition
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_variable",
			APIType:      "Variable",
			Name:         "DaVinci Variable",
			ShortName:    "variable",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/pingidentity/pingone-go-client/davinci",
			SDKType:    "Variable",
			IDField:    "ID",
			NameField:  "Name",
		},
		Attributes: []schema.AttributeDefinition{
			{
				Name:          "ID",
				TerraformName: "id",
				Type:          "string",
				Computed:      true,
			},
			{
				Name:          "Name",
				TerraformName: "name",
				Type:          "string",
				Required:      true,
			},
			{
				Name:          "Context",
				TerraformName: "context",
				Type:          "string",
				Required:      true,
			},
			{
				Name:          "DataType",
				TerraformName: "data_type",
				Type:          "string",
				Required:      true,
			},
			{
				Name:          "Mutable",
				TerraformName: "mutable",
				Type:          "bool",
			},
			{
				Name:          "DisplayName",
				TerraformName: "display_name",
				Type:          "string",
			},
		},
	}

	// Create registry and register the definition
	registry := schema.NewRegistry()
	err := registry.Register(def)
	require.NoError(t, err)

	// Create processor
	processor := core.NewProcessor(registry)

	// Create mock API data
	displayName := "My Variable"
	mockData := MockVariable{
		ID:          "abc123",
		Name:        "test_var",
		Context:     "company",
		DataType:    "string",
		Mutable:     true,
		DisplayName: &displayName,
	}

	// Process the resource
	result, err := processor.ProcessResource("pingone_davinci_variable", &mockData)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify basic fields
	assert.Equal(t, "pingone_davinci_variable", result.ResourceType)
	assert.Equal(t, "abc123", result.ID)
	assert.Equal(t, "test_var", result.Name)

	// Verify attributes
	assert.Equal(t, "abc123", result.Attributes["id"])
	assert.Equal(t, "test_var", result.Attributes["name"])
	assert.Equal(t, "company", result.Attributes["context"])
	assert.Equal(t, "string", result.Attributes["data_type"])
	assert.Equal(t, true, result.Attributes["mutable"])
	assert.Equal(t, "My Variable", result.Attributes["display_name"])
}

func TestProcessorListProcessing(t *testing.T) {
	// Create a minimal schema definition
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_variable",
			APIType:      "Variable",
			Name:         "DaVinci Variable",
			ShortName:    "variable",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/pingidentity/pingone-go-client/davinci",
			SDKType:    "Variable",
			IDField:    "ID",
			NameField:  "Name",
		},
		Attributes: []schema.AttributeDefinition{
			{
				Name:          "ID",
				TerraformName: "id",
				Type:          "string",
				Computed:      true,
			},
			{
				Name:          "Name",
				TerraformName: "name",
				Type:          "string",
				Required:      true,
			},
		},
	}

	// Create registry and register the definition
	registry := schema.NewRegistry()
	err := registry.Register(def)
	require.NoError(t, err)

	// Create processor
	processor := core.NewProcessor(registry)

	// Create mock API data list
	mockDataList := []MockVariable{
		{ID: "id1", Name: "var1", Context: "company", DataType: "string", Mutable: true},
		{ID: "id2", Name: "var2", Context: "flow", DataType: "number", Mutable: false},
		{ID: "id3", Name: "var3", Context: "user", DataType: "boolean", Mutable: true},
	}

	// Process the resource list
	results, err := processor.ProcessResourceList("pingone_davinci_variable", mockDataList)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify first resource
	assert.Equal(t, "id1", results[0].ID)
	assert.Equal(t, "var1", results[0].Name)

	// Verify second resource
	assert.Equal(t, "id2", results[1].ID)
	assert.Equal(t, "var2", results[1].Name)

	// Verify third resource
	assert.Equal(t, "id3", results[2].ID)
	assert.Equal(t, "var3", results[2].Name)
}

func TestProcessorOptionalFields(t *testing.T) {
	// Create a schema with optional fields
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_variable",
			APIType:      "Variable",
			Name:         "DaVinci Variable",
			ShortName:    "variable",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/pingidentity/pingone-go-client/davinci",
			SDKType:    "Variable",
			IDField:    "ID",
			NameField:  "Name",
		},
		Attributes: []schema.AttributeDefinition{
			{
				Name:          "ID",
				TerraformName: "id",
				Type:          "string",
				Computed:      true,
			},
			{
				Name:          "Name",
				TerraformName: "name",
				Type:          "string",
				Required:      true,
			},
			{
				Name:          "DisplayName",
				TerraformName: "display_name",
				Type:          "string",
			},
			{
				Name:          "Min",
				TerraformName: "min",
				Type:          "number",
			},
		},
	}

	// Create registry and register the definition
	registry := schema.NewRegistry()
	err := registry.Register(def)
	require.NoError(t, err)

	// Create processor
	processor := core.NewProcessor(registry)

	// Create mock data without optional fields
	mockData := MockVariable{
		ID:       "abc123",
		Name:     "test_var",
		Context:  "company",
		DataType: "string",
		Mutable:  true,
		// DisplayName, Min, Max not set
	}

	// Process the resource
	result, err := processor.ProcessResource("pingone_davinci_variable", &mockData)
	require.NoError(t, err)

	// Verify required fields are present
	assert.Equal(t, "abc123", result.Attributes["id"])
	assert.Equal(t, "test_var", result.Attributes["name"])

	// Verify optional fields are not present when nil
	_, hasDisplayName := result.Attributes["display_name"]
	assert.False(t, hasDisplayName, "display_name should not be in attributes when nil")

	_, hasMin := result.Attributes["min"]
	assert.False(t, hasMin, "min should not be in attributes when nil")
}

// MockNestedResource represents a struct with nested fields for dot-notation testing
type MockNestedResource struct {
	ID          string
	Name        string
	Environment struct {
		ID string
	}
	Flow *struct {
		ID   string
		Name string
	}
}

func TestProcessorDotNotationTraversal(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test",
			APIType:      "Test",
			Name:         "Test Resource",
			ShortName:    "test",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/pingidentity/test",
			SDKType:    "Test",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{
				Name:          "ID",
				TerraformName: "id",
				Type:          "string",
				SourcePath:    "ID",
			},
			{
				Name:          "Name",
				TerraformName: "name",
				Type:          "string",
				SourcePath:    "Name",
			},
			{
				Name:          "EnvironmentID",
				TerraformName: "environment_id",
				Type:          "string",
				SourcePath:    "Environment.ID",
			},
			{
				Name:          "FlowID",
				TerraformName: "flow_id",
				Type:          "string",
				SourcePath:    "Flow.ID",
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	processor := core.NewProcessor(registry)

	t.Run("nested struct field", func(t *testing.T) {
		mock := &MockNestedResource{
			ID:   "res-123",
			Name: "test_resource",
		}
		mock.Environment.ID = "env-456"

		result, err := processor.ProcessResource("pingone_davinci_test", mock)
		require.NoError(t, err)
		assert.Equal(t, "res-123", result.Attributes["id"])
		assert.Equal(t, "test_resource", result.Attributes["name"])
		assert.Equal(t, "env-456", result.Attributes["environment_id"])
	})

	t.Run("nil pointer in path", func(t *testing.T) {
		mock := &MockNestedResource{
			ID:   "res-789",
			Name: "test_no_flow",
		}
		mock.Environment.ID = "env-abc"
		// Flow is nil

		result, err := processor.ProcessResource("pingone_davinci_test", mock)
		require.NoError(t, err)
		assert.Equal(t, "res-789", result.Attributes["id"])
		// flow_id should not be present when Flow is nil
		_, hasFlowID := result.Attributes["flow_id"]
		assert.False(t, hasFlowID)
	})

	t.Run("pointer with value", func(t *testing.T) {
		mock := &MockNestedResource{
			ID:   "res-def",
			Name: "test_with_flow",
			Flow: &struct {
				ID   string
				Name string
			}{ID: "flow-999", Name: "my_flow"},
		}
		mock.Environment.ID = "env-ghi"

		result, err := processor.ProcessResource("pingone_davinci_test", mock)
		require.NoError(t, err)
		assert.Equal(t, "res-def", result.Attributes["id"])
		assert.Equal(t, "env-ghi", result.Attributes["environment_id"])
		assert.Equal(t, "flow-999", result.Attributes["flow_id"])
	})
}

// ── type_discriminated_block tests ──────────────────────────────

// MockTDB mirrors a struct with a Value interface{} field and a DataType string.
type MockTDB struct {
	ID       string
	Name     string
	DataType string
	Mutable  bool
	Value    interface{}
}

func tdbDef(skipConditions []schema.SkipCondition) *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_tdb",
			APIType:      "TDB",
			Name:         "TDB Test",
			ShortName:    "tdb",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:  "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{Name: "Mutable", TerraformName: "mutable", Type: "bool", SourcePath: "Mutable"},
			{
				Name:          "Value",
				TerraformName: "value",
				Type:          "type_discriminated_block",
				SourcePath:    "Value",
				TypeDiscriminatedBlock: &schema.TypeDiscriminatedBlockConfig{
					TypeKeyMap: map[string]string{
						"string":  "string",
						"bool":    "bool",
						"float64": "float32",
						"int":     "float32",
						"map":     "json_object",
						"slice":   "json_object",
					},
					JSONEncodeKeys: []string{"json_object"},
					SkipConditions: skipConditions,
				},
			},
		},
	}
}

func TestProcessorTDB_StringValue(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Mutable: true, Value: "hello"}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	val := result.Attributes["value"]
	require.NotNil(t, val)
	m, ok := val.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "hello", m["string"])
}

func TestProcessorTDB_BoolValue(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: true}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	m := result.Attributes["value"].(map[string]interface{})
	assert.Equal(t, true, m["bool"])
}

func TestProcessorTDB_Float64Integer(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: float64(42)}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	m := result.Attributes["value"].(map[string]interface{})
	assert.Equal(t, int64(42), m["float32"])
}

func TestProcessorTDB_Float64Decimal(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: float64(3.14)}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	m := result.Attributes["value"].(map[string]interface{})
	assert.Equal(t, float64(3.14), m["float32"])
}

func TestProcessorTDB_MapValue(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: map[string]interface{}{"key": "val"}}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	m := result.Attributes["value"].(map[string]interface{})
	raw, ok := m["json_object"].(core.RawHCLValue)
	require.True(t, ok, "expected RawHCLValue")
	assert.Equal(t, core.RawHCLValue(`{"key":"val"}`), raw)
}

func TestProcessorTDB_SliceValue(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: []interface{}{"a", "b"}}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	m := result.Attributes["value"].(map[string]interface{})
	_, ok := m["json_object"].(core.RawHCLValue)
	require.True(t, ok)
}

func TestProcessorTDB_NilValue(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: nil}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	_, exists := result.Attributes["value"]
	assert.False(t, exists, "nil value should not produce attribute")
}

func TestProcessorTDB_EmptyString(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: ""}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	_, exists := result.Attributes["value"]
	assert.False(t, exists, "empty string should not produce attribute")
}

func TestProcessorTDB_SkipCondition(t *testing.T) {
	def := tdbDef([]schema.SkipCondition{
		{SourceField: "DataType", Equals: "secret"},
	})
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", DataType: "secret", Value: "masked"}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	_, exists := result.Attributes["value"]
	assert.False(t, exists, "skip_condition should suppress the block")
}

func TestProcessorTDB_SkipConditionNotTriggered(t *testing.T) {
	def := tdbDef([]schema.SkipCondition{
		{SourceField: "DataType", Equals: "secret"},
	})
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", DataType: "string", Value: "hello"}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	m := result.Attributes["value"].(map[string]interface{})
	assert.Equal(t, "hello", m["string"])
}

func TestProcessorTDB_EmptyMap(t *testing.T) {
	def := tdbDef(nil)
	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockTDB{ID: "1", Name: "v", Value: map[string]interface{}{}}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)

	_, exists := result.Attributes["value"]
	assert.False(t, exists, "empty map should not produce attribute")
}

// ── conditional_defaults tests ──────────────────────────────────

func TestProcessorConditionalDefaults_MutableOverride(t *testing.T) {
	def := tdbDef(nil)
	def.ConditionalDefaults = []schema.ConditionalDefault{
		{
			TargetAttribute: "mutable",
			SetValue:        true,
			WhenAll: []schema.DefaultCondition{
				{AttributeEmpty: "value"},
				{AttributeEquals: &schema.AttributeValueCondition{Name: "mutable", Value: false}},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	// No value + mutable false → should be overridden to true.
	mock := &MockTDB{ID: "1", Name: "v", Mutable: false, Value: nil}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)
	assert.Equal(t, true, result.Attributes["mutable"])
}

func TestProcessorConditionalDefaults_NoOverrideWhenValueSet(t *testing.T) {
	def := tdbDef(nil)
	def.ConditionalDefaults = []schema.ConditionalDefault{
		{
			TargetAttribute: "mutable",
			SetValue:        true,
			WhenAll: []schema.DefaultCondition{
				{AttributeEmpty: "value"},
				{AttributeEquals: &schema.AttributeValueCondition{Name: "mutable", Value: false}},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	// Has value + mutable false → conditions not all met, no override.
	mock := &MockTDB{ID: "1", Name: "v", Mutable: false, Value: "hello"}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)
	assert.Equal(t, false, result.Attributes["mutable"])
}

func TestProcessorConditionalDefaults_NoOverrideWhenMutableTrue(t *testing.T) {
	def := tdbDef(nil)
	def.ConditionalDefaults = []schema.ConditionalDefault{
		{
			TargetAttribute: "mutable",
			SetValue:        true,
			WhenAll: []schema.DefaultCondition{
				{AttributeEmpty: "value"},
				{AttributeEquals: &schema.AttributeValueCondition{Name: "mutable", Value: false}},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	// No value + mutable true → mutable condition not met, no override.
	mock := &MockTDB{ID: "1", Name: "v", Mutable: true, Value: nil}
	result, err := p.ProcessResource("test_tdb", mock)
	require.NoError(t, err)
	assert.Equal(t, true, result.Attributes["mutable"])
}

// ── Choice wrapper unwrapping tests ──────────────────────────────

// MockChoiceWrapper mimics PingOne SDK choice types like
// DaVinciFlowSettingsResponseCustomErrorShowFooter where one pointer
// field is non-nil.
type MockChoiceWrapperBool struct {
	Bool   *bool
	Object *map[string]interface{}
}

type MockChoiceWrapperString struct {
	String *string
	Object *map[string]interface{}
}

type MockWithChoiceWrappers struct {
	ID       string
	Name     string
	Settings *MockSettingsWithChoiceWrappers
}

type MockSettingsWithChoiceWrappers struct {
	ShowFooter   *MockChoiceWrapperBool
	ErrorMessage *MockChoiceWrapperString
	LogLevel     *int32
}

func TestProcessorChoiceWrapperBool(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_choice",
			APIType:      "Choice",
			Name:         "Choice Test",
			ShortName:    "choice",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "Settings",
				TerraformName: "settings",
				Type:          "object",
				SourcePath:    "Settings",
				NestedAttributes: []schema.AttributeDefinition{
					{
						Name:          "ShowFooter",
						TerraformName: "custom_error_show_footer",
						Type:          "bool",
						SourcePath:    "ShowFooter",
					},
					{
						Name:          "ErrorMessage",
						TerraformName: "custom_error_message",
						Type:          "string",
						SourcePath:    "ErrorMessage",
					},
					{
						Name:          "LogLevel",
						TerraformName: "log_level",
						Type:          "number",
						SourcePath:    "LogLevel",
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	t.Run("bool choice wrapper", func(t *testing.T) {
		boolVal := true
		mock := &MockWithChoiceWrappers{
			ID:   "1",
			Name: "test",
			Settings: &MockSettingsWithChoiceWrappers{
				ShowFooter: &MockChoiceWrapperBool{Bool: &boolVal},
			},
		}

		result, err := p.ProcessResource("test_choice", mock)
		require.NoError(t, err)
		settings, ok := result.Attributes["settings"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, settings["custom_error_show_footer"])
	})

	t.Run("string choice wrapper", func(t *testing.T) {
		strVal := "custom error text"
		mock := &MockWithChoiceWrappers{
			ID:   "2",
			Name: "test2",
			Settings: &MockSettingsWithChoiceWrappers{
				ErrorMessage: &MockChoiceWrapperString{String: &strVal},
			},
		}

		result, err := p.ProcessResource("test_choice", mock)
		require.NoError(t, err)
		settings, ok := result.Attributes["settings"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "custom error text", settings["custom_error_message"])
	})

	t.Run("nil choice wrapper omitted", func(t *testing.T) {
		mock := &MockWithChoiceWrappers{
			ID:   "3",
			Name: "test3",
			Settings: &MockSettingsWithChoiceWrappers{
				ShowFooter: &MockChoiceWrapperBool{}, // all fields nil
			},
		}

		result, err := p.ProcessResource("test_choice", mock)
		require.NoError(t, err)
		// Settings should either be absent or not contain show_footer
		if settings, ok := result.Attributes["settings"].(map[string]interface{}); ok {
			_, hasFooter := settings["custom_error_show_footer"]
			assert.False(t, hasFooter)
		}
	})

	t.Run("regular pointer field still works", func(t *testing.T) {
		logLevel := int32(4)
		mock := &MockWithChoiceWrappers{
			ID:   "4",
			Name: "test4",
			Settings: &MockSettingsWithChoiceWrappers{
				LogLevel: &logLevel,
			},
		}

		result, err := p.ProcessResource("test_choice", mock)
		require.NoError(t, err)
		settings, ok := result.Attributes["settings"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, int64(4), settings["log_level"])
	})
}

// MockChoiceWrapperBoolString mimics SDK union types with Bool + Choice2(string)
// variants where the Choice2 is a named string type.
type MockChoiceWrapperBoolString struct {
	Choice2 *string
	Bool    *bool
}

type MockWithBoolStringWrappers struct {
	ID   string
	Name string
	Flag *MockChoiceWrapperBoolString
}

func TestProcessorChoiceWrapperStringToBool(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_choice_str",
			APIType:      "ChoiceStr",
			Name:         "Choice String Test",
			ShortName:    "choicestr",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{Name: "Flag", TerraformName: "flag", Type: "bool", SourcePath: "Flag"},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	t.Run("Choice2 string true", func(t *testing.T) {
		strTrue := "true"
		mock := &MockWithBoolStringWrappers{
			ID:   "1",
			Name: "test",
			Flag: &MockChoiceWrapperBoolString{Choice2: &strTrue},
		}
		result, err := p.ProcessResource("test_choice_str", mock)
		require.NoError(t, err)
		assert.Equal(t, true, result.Attributes["flag"])
	})

	t.Run("Choice2 string false", func(t *testing.T) {
		strFalse := "false"
		mock := &MockWithBoolStringWrappers{
			ID:   "2",
			Name: "test2",
			Flag: &MockChoiceWrapperBoolString{Choice2: &strFalse},
		}
		result, err := p.ProcessResource("test_choice_str", mock)
		require.NoError(t, err)
		assert.Equal(t, false, result.Attributes["flag"])
	})
}

// ── Slice-to-map keying tests ──────────────────────────────

type MockNode struct {
	Data *MockNodeData
}

type MockNodeData struct {
	Id           string
	Name         string
	ConnectionID string
}

type MockEdge struct {
	Data *MockEdgeData
}

type MockEdgeData struct {
	Id string
}

type MockElements struct {
	Nodes []MockNode
	Edges []MockEdge
}

type MockGraphData struct {
	ID       string
	Name     string
	Elements *MockElements
}

func TestProcessorSliceToMapKeying(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_slice_map",
			APIType:      "SliceMap",
			Name:         "Slice Map Test",
			ShortName:    "slicemap",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "Elements",
				TerraformName: "elements",
				Type:          "object",
				SourcePath:    "Elements",
				NestedAttributes: []schema.AttributeDefinition{
					{
						Name:          "Nodes",
						TerraformName: "nodes",
						Type:          "map",
						SourcePath:    "Nodes",
						MapKeyPath:    "Data.Id",
						NestedAttributes: []schema.AttributeDefinition{
							{
								Name:          "Data",
								TerraformName: "data",
								Type:          "object",
								SourcePath:    "Data",
								NestedAttributes: []schema.AttributeDefinition{
									{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
									{Name: "ConnectionID", TerraformName: "connection_id", Type: "string", SourcePath: "ConnectionID"},
								},
							},
						},
					},
					{
						Name:          "Edges",
						TerraformName: "edges",
						Type:          "map",
						SourcePath:    "Edges",
						MapKeyPath:    "Data.Id",
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	t.Run("slice to map with nested attributes", func(t *testing.T) {
		mock := &MockGraphData{
			ID:   "g1",
			Name: "graph",
			Elements: &MockElements{
				Nodes: []MockNode{
					{Data: &MockNodeData{Id: "node1", Name: "Login", ConnectionID: "conn-abc"}},
					{Data: &MockNodeData{Id: "node2", Name: "Register", ConnectionID: "conn-def"}},
				},
				Edges: []MockEdge{
					{Data: &MockEdgeData{Id: "edge1"}},
					{Data: &MockEdgeData{Id: "edge2"}},
				},
			},
		}

		result, err := p.ProcessResource("test_slice_map", mock)
		require.NoError(t, err)

		elements, ok := result.Attributes["elements"].(map[string]interface{})
		require.True(t, ok)

		// Nodes should be a map keyed by Data.Id
		nodes, ok := elements["nodes"].(map[string]interface{})
		require.True(t, ok, "nodes should be map[string]interface{}")
		assert.Len(t, nodes, 2)

		node1, ok := nodes["node1"].(map[string]interface{})
		require.True(t, ok, "node1 should be map[string]interface{}")

		node1Data, ok := node1["data"].(map[string]interface{})
		require.True(t, ok, "node1.data should be map")
		assert.Equal(t, "Login", node1Data["name"])
		assert.Equal(t, "conn-abc", node1Data["connection_id"])

		node2, ok := nodes["node2"].(map[string]interface{})
		require.True(t, ok)
		node2Data, ok := node2["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Register", node2Data["name"])

		// Edges should be a map keyed by Data.Id (no nested attrs → raw map entries)
		edges, ok := elements["edges"].(map[string]interface{})
		require.True(t, ok, "edges should be map[string]interface{}")
		assert.Len(t, edges, 2)
		assert.Contains(t, edges, "edge1")
		assert.Contains(t, edges, "edge2")
	})

	t.Run("empty slice produces empty map", func(t *testing.T) {
		mock := &MockGraphData{
			ID:   "g2",
			Name: "empty",
			Elements: &MockElements{
				Nodes: []MockNode{},
				Edges: []MockEdge{},
			},
		}

		result, err := p.ProcessResource("test_slice_map", mock)
		require.NoError(t, err)

		// Empty slices should result in no elements attribute (or empty map)
		if elements, ok := result.Attributes["elements"].(map[string]interface{}); ok {
			if nodes, ok := elements["nodes"].(map[string]interface{}); ok {
				assert.Empty(t, nodes)
			}
		}
	})

	t.Run("nil elements omitted", func(t *testing.T) {
		mock := &MockGraphData{
			ID:       "g3",
			Name:     "nil_elements",
			Elements: nil,
		}

		result, err := p.ProcessResource("test_slice_map", mock)
		require.NoError(t, err)

		_, hasElements := result.Attributes["elements"]
		assert.False(t, hasElements)
	})
}

// ── Nested attribute transform tests ─────────────────────────

type MockWithNestedTransform struct {
	ID   string
	Name string
	Outer *MockOuterObj
}

type MockOuterObj struct {
	Data     map[string]interface{}
	Label    string
}

func TestProcessorNestedAttributeTransform(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_nested_xform",
			APIType:      "NestedXform",
			Name:         "Nested Transform Test",
			ShortName:    "nestxform",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "Outer",
				TerraformName: "outer",
				Type:          "object",
				SourcePath:    "Outer",
				NestedAttributes: []schema.AttributeDefinition{
					{
						Name:          "Data",
						TerraformName: "data",
						Type:          "object",
						SourcePath:    "Data",
						Transform:     "jsonencode_raw",
					},
					{
						Name:          "Label",
						TerraformName: "label",
						Type:          "string",
						SourcePath:    "Label",
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockWithNestedTransform{
		ID:   "1",
		Name: "test",
		Outer: &MockOuterObj{
			Data:  map[string]interface{}{"key": "val"},
			Label: "myLabel",
		},
	}

	result, err := p.ProcessResource("test_nested_xform", mock)
	require.NoError(t, err)

	outer, ok := result.Attributes["outer"].(map[string]interface{})
	require.True(t, ok)

	// Data should be wrapped in jsonencode() via the transform
	dataVal, ok := outer["data"].(core.RawHCLValue)
	require.True(t, ok, "data should be RawHCLValue, got %T", outer["data"])
	assert.Equal(t, core.RawHCLValue("jsonencode({\n  \"key\": \"val\"\n})"), dataVal)

	// Label should be a plain string (no transform)
	assert.Equal(t, "myLabel", outer["label"])
}

// ── List with nested attribute tests ─────────────────────────

type MockInputSchemaItem struct {
	PropertyName         string
	PreferredDataType    string
	PreferredControlType string
	Required             bool
}

type MockFlowWithInputSchema struct {
	ID          string
	Name        string
	InputSchema []MockInputSchemaItem
}

func TestProcessorListWithNestedAttributes(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_list_nested",
			APIType:      "ListNested",
			Name:         "List Nested Test",
			ShortName:    "listnested",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "InputSchema",
				TerraformName: "input_schema",
				Type:          "list",
				SourcePath:    "InputSchema",
				NestedAttributes: []schema.AttributeDefinition{
					{Name: "PropertyName", TerraformName: "property_name", Type: "string", SourcePath: "PropertyName"},
					{
						Name:          "PreferredDataType",
						TerraformName: "preferred_data_type",
						Type:          "string",
						SourcePath:    "PreferredDataType",
						Transform:     "value_map",
						ValueMap: map[string]string{
							"bool": "boolean",
							"int":  "number",
						},
						ValueMapDefault: "string",
					},
					{Name: "PreferredControlType", TerraformName: "preferred_control_type", Type: "string", SourcePath: "PreferredControlType"},
					{Name: "Required", TerraformName: "required", Type: "bool", SourcePath: "Required"},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockFlowWithInputSchema{
		ID:   "f1",
		Name: "flow1",
		InputSchema: []MockInputSchemaItem{
			{PropertyName: "username", PreferredDataType: "string", PreferredControlType: "textField", Required: true},
			{PropertyName: "remember_me", PreferredDataType: "bool", PreferredControlType: "checkBox", Required: false},
			{PropertyName: "age", PreferredDataType: "int", PreferredControlType: "numberField", Required: false},
		},
	}

	result, err := p.ProcessResource("test_list_nested", mock)
	require.NoError(t, err)

	schemaList, ok := result.Attributes["input_schema"].([]interface{})
	require.True(t, ok, "input_schema should be []interface{}, got %T", result.Attributes["input_schema"])
	require.Len(t, schemaList, 3)

	item0, ok := schemaList[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "username", item0["property_name"])
	assert.Equal(t, "string", item0["preferred_data_type"]) // "string" preserved
	assert.Equal(t, true, item0["required"])

	item1, ok := schemaList[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "remember_me", item1["property_name"])
	assert.Equal(t, "boolean", item1["preferred_data_type"]) // "bool" → "boolean" via value_map

	item2, ok := schemaList[2].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "age", item2["property_name"])
	assert.Equal(t, "number", item2["preferred_data_type"]) // "int" → "number" via value_map
}

// ── override_value tests ────────────────────────────────────────

// MockOverrideResource is used by override_value tests.
type MockOverrideResource struct {
	ID      string
	Name    string
	Version int
	Nested  struct {
		Code    string
		Version int
	}
	Items []MockOverrideItem
}

// MockOverrideItem is a list element for override_value tests.
type MockOverrideItem struct {
	Label   string
	Version int
}

func TestOverrideValue_TopLevelAttribute(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_override",
			APIType:      "Override",
			Name:         "Override Test",
			ShortName:    "override",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "Version",
				TerraformName: "version",
				Type:          "number",
				SourcePath:    "Version",
				OverrideValue: -1,
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockOverrideResource{ID: "r1", Name: "res", Version: 42}
	result, err := p.ProcessResource("test_override", mock)
	require.NoError(t, err)

	// The API returns 42 but override_value forces -1.
	assert.Equal(t, -1, result.Attributes["version"])
	// Other attributes are extracted normally.
	assert.Equal(t, "r1", result.Attributes["id"])
	assert.Equal(t, "res", result.Attributes["name"])
}

func TestOverrideValue_NestedObjectAttribute(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_override_nested",
			APIType:      "OverrideNested",
			Name:         "Override Nested Test",
			ShortName:    "override_nested",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "Nested",
				TerraformName: "nested",
				Type:          "object",
				SourcePath:    "Nested",
				NestedAttributes: []schema.AttributeDefinition{
					{Name: "Code", TerraformName: "code", Type: "string", SourcePath: "Code"},
					{
						Name:          "Version",
						TerraformName: "version",
						Type:          "number",
						SourcePath:    "Version",
						OverrideValue: -1,
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockOverrideResource{ID: "r2", Name: "res2"}
	mock.Nested.Code = "ABC"
	mock.Nested.Version = 99

	result, err := p.ProcessResource("test_override_nested", mock)
	require.NoError(t, err)

	nested, ok := result.Attributes["nested"].(map[string]interface{})
	require.True(t, ok, "nested should be map[string]interface{}")
	assert.Equal(t, "ABC", nested["code"])
	assert.Equal(t, -1, nested["version"]) // overridden
}

func TestOverrideValue_ListElementAttribute(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_override_list",
			APIType:      "OverrideList",
			Name:         "Override List Test",
			ShortName:    "override_list",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "Items",
				TerraformName: "items",
				Type:          "list",
				SourcePath:    "Items",
				NestedAttributes: []schema.AttributeDefinition{
					{Name: "Label", TerraformName: "label", Type: "string", SourcePath: "Label"},
					{
						Name:          "Version",
						TerraformName: "version",
						Type:          "number",
						SourcePath:    "Version",
						OverrideValue: -1,
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockOverrideResource{
		ID:   "r3",
		Name: "res3",
		Items: []MockOverrideItem{
			{Label: "first", Version: 5},
			{Label: "second", Version: 10},
		},
	}

	result, err := p.ProcessResource("test_override_list", mock)
	require.NoError(t, err)

	items, ok := result.Attributes["items"].([]interface{})
	require.True(t, ok, "items should be []interface{}")
	require.Len(t, items, 2)

	item0, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "first", item0["label"])
	assert.Equal(t, -1, item0["version"]) // overridden

	item1, ok := items[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "second", item1["label"])
	assert.Equal(t, -1, item1["version"]) // overridden
}

func TestOverrideValue_NoOverrideStillWorks(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "test",
			ResourceType: "test_no_override",
			APIType:      "NoOverride",
			Name:         "No Override Test",
			ShortName:    "no_override",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "ID"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name:          "Version",
				TerraformName: "version",
				Type:          "number",
				SourcePath:    "Version",
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &MockOverrideResource{ID: "r4", Name: "res4", Version: 42}
	result, err := p.ProcessResource("test_no_override", mock)
	require.NoError(t, err)

	// Without override_value, the API value is used directly.
	assert.Equal(t, int64(42), result.Attributes["version"])
}
