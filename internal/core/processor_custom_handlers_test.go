package core_test

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTransformResource is a test struct for custom transform tests.
type MockTransformResource struct {
	ID    string
	Name  string
	Value string
}

func TestProcessorCustomTransformInvoked(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_ct",
			APIType:      "TestCT",
			Name:         "Test Custom Transform",
			ShortName:    "test_ct",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestCT",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{
				Name:            "Value",
				TerraformName:   "value",
				Type:            "string",
				Transform:       "custom",
				CustomTransform: "uppercaseValue",
			},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	// Build custom handler registry with the uppercase transform.
	chr := core.NewCustomHandlerRegistry()
	chr.RegisterTransform("uppercaseValue", func(value interface{}, apiData interface{}, attr *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
		if s, ok := value.(string); ok {
			return "UPPER:" + s, nil
		}
		return value, nil
	})

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockTransformResource{ID: "ct-1", Name: "test", Value: "hello"}
	result, err := processor.ProcessResource("pingone_davinci_test_ct", mock)
	require.NoError(t, err)

	// Custom transform should have been applied.
	assert.Equal(t, "UPPER:hello", result.Attributes["value"])
	assert.Equal(t, "ct-1", result.Attributes["id"])
	assert.Equal(t, "test", result.Attributes["name"])
}

func TestProcessorCustomTransformNotRegistered(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_ctnr",
			APIType:      "TestCTNR",
			Name:         "Test Custom Transform Not Registered",
			ShortName:    "test_ctnr",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestCTNR",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{
				Name:            "Value",
				TerraformName:   "value",
				Type:            "string",
				Transform:       "custom",
				CustomTransform: "nonExistentTransform",
			},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	// Registry exists but does NOT contain the named transform.
	chr := core.NewCustomHandlerRegistry()
	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockTransformResource{ID: "ct-2", Name: "test2", Value: "world"}
	result, err := processor.ProcessResource("pingone_davinci_test_ctnr", mock)
	require.NoError(t, err)

	// Unregistered custom transform passes through the raw value.
	assert.Equal(t, "world", result.Attributes["value"])
}

func TestProcessorNoHandlerRegistryPassthrough(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_nhr",
			APIType:      "TestNHR",
			Name:         "Test No Handler Registry",
			ShortName:    "test_nhr",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestNHR",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{
				Name:            "Value",
				TerraformName:   "value",
				Type:            "string",
				Transform:       "custom",
				CustomTransform: "someTransform",
			},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	// No custom handler registry at all.
	processor := core.NewProcessor(reg)

	mock := &MockTransformResource{ID: "ct-3", Name: "test3", Value: "raw"}
	result, err := processor.ProcessResource("pingone_davinci_test_nhr", mock)
	require.NoError(t, err)

	// No registry means custom transform passes through.
	assert.Equal(t, "raw", result.Attributes["value"])
}

// MockHandlerResource is a test struct for resource-level handler tests.
type MockHandlerResource struct {
	ID    string
	Name  string
	Extra string
}

func TestProcessorResourceLevelTransformerHandler(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_rlt",
			APIType:      "TestRLT",
			Name:         "Test Resource Level Transformer",
			ShortName:    "test_rlt",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestRLT",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
		},
		CustomHandlers: &schema.CustomHandlerDefinition{
			Transformer: "addComputedAttrs",
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	chr := core.NewCustomHandlerRegistry()
	chr.RegisterHandler("addComputedAttrs", func(data interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error) {
		if mock, ok := data.(*MockHandlerResource); ok {
			return map[string]interface{}{
				"computed_field": "computed_" + mock.Extra,
			}, nil
		}
		return nil, nil
	})

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockHandlerResource{ID: "rlt-1", Name: "test", Extra: "bonus"}
	result, err := processor.ProcessResource("pingone_davinci_test_rlt", mock)
	require.NoError(t, err)

	// Standard attributes extracted.
	assert.Equal(t, "rlt-1", result.Attributes["id"])
	assert.Equal(t, "test", result.Attributes["name"])

	// Custom handler should have added computed_field.
	assert.Equal(t, "computed_bonus", result.Attributes["computed_field"])
}

func TestProcessorResourceLevelHCLGeneratorHandler(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_hcl",
			APIType:      "TestHCL",
			Name:         "Test HCL Generator",
			ShortName:    "test_hcl",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestHCL",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
		},
		CustomHandlers: &schema.CustomHandlerDefinition{
			HCLGenerator: "genFlowHCL",
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	chr := core.NewCustomHandlerRegistry()
	chr.RegisterHandler("genFlowHCL", func(data interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{
			"graph_data": "generated_hcl_content",
		}, nil
	})

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockHandlerResource{ID: "hcl-1", Name: "flow1", Extra: "x"}
	result, err := processor.ProcessResource("pingone_davinci_test_hcl", mock)
	require.NoError(t, err)

	assert.Equal(t, "hcl-1", result.Attributes["id"])
	assert.Equal(t, "flow1", result.Attributes["name"])
	assert.Equal(t, "generated_hcl_content", result.Attributes["graph_data"])
}

func TestProcessorBothHandlerTypes(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_both",
			APIType:      "TestBoth",
			Name:         "Test Both Handlers",
			ShortName:    "test_both",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestBoth",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
		},
		CustomHandlers: &schema.CustomHandlerDefinition{
			Transformer:  "addExtra",
			HCLGenerator: "addHCL",
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	chr := core.NewCustomHandlerRegistry()
	chr.RegisterHandler("addExtra", func(data interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{"extra_attr": "from_transformer"}, nil
	})
	chr.RegisterHandler("addHCL", func(data interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{"hcl_block": "from_hcl_generator"}, nil
	})

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockHandlerResource{ID: "both-1", Name: "combined", Extra: "y"}
	result, err := processor.ProcessResource("pingone_davinci_test_both", mock)
	require.NoError(t, err)

	assert.Equal(t, "both-1", result.Attributes["id"])
	assert.Equal(t, "combined", result.Attributes["name"])
	assert.Equal(t, "from_transformer", result.Attributes["extra_attr"])
	assert.Equal(t, "from_hcl_generator", result.Attributes["hcl_block"])
}

func TestProcessorCustomHandlerNotRegistered(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_chnr",
			APIType:      "TestCHNR",
			Name:         "Test Custom Handler Not Registered",
			ShortName:    "test_chnr",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestCHNR",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
		},
		CustomHandlers: &schema.CustomHandlerDefinition{
			Transformer:  "nonExistentHandler",
			HCLGenerator: "nonExistentHCLGen",
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	// Registry exists but handlers are not registered.
	chr := core.NewCustomHandlerRegistry()
	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockHandlerResource{ID: "chnr-1", Name: "test_nr", Extra: "z"}
	result, err := processor.ProcessResource("pingone_davinci_test_chnr", mock)
	require.NoError(t, err)

	// Only standard attributes should be present.
	assert.Equal(t, "chnr-1", result.Attributes["id"])
	assert.Equal(t, "test_nr", result.Attributes["name"])
	assert.Len(t, result.Attributes, 2)
}

func TestProcessorCustomTransformReceivesAPIData(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_api",
			APIType:      "TestAPI",
			Name:         "Test API Data Access",
			ShortName:    "test_api",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestAPI",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{
				Name:            "Value",
				TerraformName:   "value",
				Type:            "string",
				Transform:       "custom",
				CustomTransform: "contextAwareTransform",
			},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	// Custom transform that uses apiData to compute the transformed value.
	chr := core.NewCustomHandlerRegistry()
	chr.RegisterTransform("contextAwareTransform", func(value interface{}, apiData interface{}, attr *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
		// Access the original API data to combine fields.
		if mock, ok := apiData.(*MockTransformResource); ok {
			return mock.Name + ":" + value.(string), nil
		}
		return value, nil
	})

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockTransformResource{ID: "api-1", Name: "myvar", Value: "orig"}
	result, err := processor.ProcessResource("pingone_davinci_test_api", mock)
	require.NoError(t, err)

	// Transform combined name + value.
	assert.Equal(t, "myvar:orig", result.Attributes["value"])
}

// TestProcessor_DependsOnFromCustomHandler verifies that when a custom handler
// returns a map with key "__depends_on" containing []RuntimeDependsOn entries,
// the processor populates result.DependsOnResources instead of adding it to
// result.Attributes.
func TestProcessor_DependsOnFromCustomHandler(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_flow_dep",
			APIType:      "FlowDep",
			Name:         "DaVinci Flow Dep",
			ShortName:    "flow_dep",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "FlowDep",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
		},
		CustomHandlers: &schema.CustomHandlerDefinition{
			Transformer: "handleFlowVariableDependencies",
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	chr := core.NewCustomHandlerRegistry()
	chr.RegisterHandler("handleFlowVariableDependencies", func(data interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{
			"__depends_on": []core.RuntimeDependsOn{
				{ResourceType: "pingone_davinci_variable", ResourceID: "var-uuid-1"},
				{ResourceType: "pingone_davinci_variable", ResourceID: "var-uuid-2"},
			},
		}, nil
	})

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockTransformResource{ID: "flow-dep-1", Name: "my_flow", Value: ""}
	result, err := processor.ProcessResource("pingone_davinci_flow_dep", mock)
	require.NoError(t, err)

	// DependsOnResources should be populated.
	require.Len(t, result.DependsOnResources, 2)
	assert.Equal(t, "pingone_davinci_variable", result.DependsOnResources[0].ResourceType)
	assert.Equal(t, "var-uuid-1", result.DependsOnResources[0].ResourceID)
	assert.Equal(t, "pingone_davinci_variable", result.DependsOnResources[1].ResourceType)
	assert.Equal(t, "var-uuid-2", result.DependsOnResources[1].ResourceID)

	// "__depends_on" sentinel key must NOT appear in Attributes.
	_, inAttrs := result.Attributes["__depends_on"]
	assert.False(t, inAttrs, "__depends_on sentinel key must not appear in Attributes")
}

// TestProcessor_DependsOnEmpty verifies that when no __depends_on key is returned
// by a custom handler, DependsOnResources remains empty.
func TestProcessor_DependsOnEmpty(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_flow_nodep",
			APIType:      "FlowNoDep",
			Name:         "Flow No Dep",
			ShortName:    "flow_nodep",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "FlowNoDep",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
		},
		CustomHandlers: &schema.CustomHandlerDefinition{
			Transformer: "noDepHandler",
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	chr := core.NewCustomHandlerRegistry()
	chr.RegisterHandler("noDepHandler", func(data interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{"some_attr": "some_value"}, nil
	})

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockTransformResource{ID: "flow-nodep-1", Name: "nodep_flow", Value: ""}
	result, err := processor.ProcessResource("pingone_davinci_flow_nodep", mock)
	require.NoError(t, err)

	// DependsOnResources should remain empty when __depends_on key is absent.
	assert.Empty(t, result.DependsOnResources)
	assert.Equal(t, "some_value", result.Attributes["some_attr"])
}

func TestProcessorHandlerQueueIntegration(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_qi",
			APIType:      "TestQI",
			Name:         "Test Queue Integration",
			ShortName:    "test_qi",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestQI",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{
				Name:            "Value",
				TerraformName:   "value",
				Type:            "string",
				Transform:       "custom",
				CustomTransform: "queuedTransform",
			},
		},
		CustomHandlers: &schema.CustomHandlerDefinition{
			Transformer: "queuedHandler",
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))

	// Simulate init()-time queuing (as resource_*.go files do).
	queue := core.NewCustomHandlerQueue()
	queue.AddTransform("queuedTransform", func(value interface{}, apiData interface{}, attr *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
		return "queued:" + value.(string), nil
	})
	queue.AddHandler("queuedHandler", func(data interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{"queued_attr": "from_queue"}, nil
	})

	// LoadInto transfers queue entries to the runtime registry.
	chr := core.NewCustomHandlerRegistry()
	queue.LoadInto(chr)

	processor := core.NewProcessor(reg, core.WithCustomHandlers(chr))

	mock := &MockTransformResource{ID: "qi-1", Name: "queued", Value: "data"}
	result, err := processor.ProcessResource("pingone_davinci_test_qi", mock)
	require.NoError(t, err)

	assert.Equal(t, "queued:data", result.Attributes["value"])
	assert.Equal(t, "from_queue", result.Attributes["queued_attr"])
}

