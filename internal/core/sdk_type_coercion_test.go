package core_test

import (
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sdkTestDef creates a minimal ResourceDefinition for SDK coercion tests.
func sdkTestDef(resourceType string, attrs []schema.AttributeDefinition) *schema.ResourceDefinition {
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: resourceType,
			APIType:      "Test",
			Name:         "Test",
			ShortName:    "test",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			IDField:   "id",
			NameField: "name",
		},
		Attributes: attrs,
	}
}

// TestSDK_UUIDFieldConvertedToString verifies that uuid.UUID fields
// (as used by DaVinciVariableResponse.Id) are converted to their string representation.
func TestSDK_UUIDFieldConvertedToString(t *testing.T) {
	type mockWithUUID struct {
		Id   uuid.UUID
		Name string
	}

	def := sdkTestDef("test_uuid", []schema.AttributeDefinition{
		{Name: "Id", TerraformName: "id", Type: "string", SourcePath: "Id"},
		{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
	})

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mock := &mockWithUUID{Id: testUUID, Name: "test"}
	result, err := p.ProcessResource("test_uuid", mock)
	require.NoError(t, err)

	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", result.Attributes["id"])
	assert.Equal(t, "test", result.Attributes["name"])
}

// TestSDK_NestedUUIDPath verifies dot-notation path to nested uuid.UUID fields
// (as used by DaVinciVariableResponse.Environment.Id).
func TestSDK_NestedUUIDPath(t *testing.T) {
	type mockEnv struct {
		Id uuid.UUID
	}
	type mockResource struct {
		Environment mockEnv
		Name        string
	}

	def := sdkTestDef("test_nested_uuid", []schema.AttributeDefinition{
		{Name: "EnvironmentID", TerraformName: "environment_id", Type: "string", SourcePath: "Environment.Id"},
		{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
	})

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	envUUID := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	mock := &mockResource{
		Environment: mockEnv{Id: envUUID},
		Name:        "my_resource",
	}
	result, err := p.ProcessResource("test_nested_uuid", mock)
	require.NoError(t, err)

	assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", result.Attributes["environment_id"])
}

// TestSDK_NamedStringTypeTreatedAsString verifies that named string types
// (SDK enum types like DaVinciVariableResponseDataType) are extracted as plain strings.
func TestSDK_NamedStringTypeTreatedAsString(t *testing.T) {
	type MyEnum string
	type mockWithEnum struct {
		Id       string
		Name     string
		DataType MyEnum
	}

	def := sdkTestDef("test_enum", []schema.AttributeDefinition{
		{Name: "DataType", TerraformName: "data_type", Type: "string", SourcePath: "DataType"},
		{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
	})

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &mockWithEnum{Id: "x", Name: "test", DataType: MyEnum("string")}
	result, err := p.ProcessResource("test_enum", mock)
	require.NoError(t, err)

	assert.Equal(t, "string", result.Attributes["data_type"])
}

// TestSDK_TypedStringSliceCoercedToStrings verifies that slices of named
// string types (e.g., []DaVinciApplicationResponseOAuthGrantType) are coerced
// to []interface{} with plain string elements.
func TestSDK_TypedStringSliceCoercedToStrings(t *testing.T) {
	type GrantType string
	type mockOAuth struct {
		Name       string
		GrantTypes []GrantType
	}

	def := sdkTestDef("test_typed_slice", []schema.AttributeDefinition{
		{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
		{Name: "GrantTypes", TerraformName: "grant_types", Type: "list", SourcePath: "GrantTypes"},
	})

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	mock := &mockOAuth{
		Name:       "test",
		GrantTypes: []GrantType{"authorizationCode", "implicit"},
	}
	result, err := p.ProcessResource("test_typed_slice", mock)
	require.NoError(t, err)

	grants := result.Attributes["grant_types"]
	require.IsType(t, []interface{}{}, grants)
	grantSlice := grants.([]interface{})
	assert.Len(t, grantSlice, 2)
	assert.Equal(t, "authorizationCode", grantSlice[0])
	assert.Equal(t, "implicit", grantSlice[1])
	// Verify the type is plain string, not the typed alias.
	_, isStr := grantSlice[0].(string)
	assert.True(t, isStr, "list element should be plain string, not typed alias")
}

// TestSDK_UnionStructUnwrapping verifies that union-type structs (all pointer
// fields, one non-nil) are unwrapped in processTypeDiscriminatedBlock.
func TestSDK_UnionStructUnwrapping(t *testing.T) {
	type UnionValue struct {
		Bool    *bool
		Float32 *float32
		Object  *map[string]interface{}
		String  *string
	}

	type mockVar struct {
		Id       string
		Name     string
		DataType string
		Value    *UnionValue
	}

	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform: "pingone", Service: "davinci",
			ResourceType: "test_union", APIType: "Test",
			Name: "Test", ShortName: "test", Version: "1.0",
		},
		API: schema.APIDefinition{IDField: "id", NameField: "name"},
		Attributes: []schema.AttributeDefinition{
			{Name: "Id", TerraformName: "id", Type: "string", SourcePath: "Id"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{Name: "DataType", TerraformName: "data_type", Type: "string", SourcePath: "DataType"},
			{
				Name: "Value", TerraformName: "value", Type: "type_discriminated_block",
				SourcePath: "Value",
				TypeDiscriminatedBlock: &schema.TypeDiscriminatedBlockConfig{
					TypeKeyMap: map[string]string{
						"string":  "string",
						"bool":    "bool",
						"float32": "float32",
						"float64": "float32",
						"map":     "json_object",
					},
					SkipConditions: []schema.SkipCondition{
						{SourceField: "DataType", Equals: "secret"},
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	tests := []struct {
		name     string
		mock     *mockVar
		wantKey  string
		wantVal  interface{}
		wantSkip bool
	}{
		{
			name: "string union field",
			mock: func() *mockVar {
				s := "hello"
				return &mockVar{Id: "v1", Name: "test", DataType: "string", Value: &UnionValue{String: &s}}
			}(),
			wantKey: "string",
			wantVal: "hello",
		},
		{
			name: "bool union field",
			mock: func() *mockVar {
				b := true
				return &mockVar{Id: "v2", Name: "bvar", DataType: "boolean", Value: &UnionValue{Bool: &b}}
			}(),
			wantKey: "bool",
			wantVal: true,
		},
		{
			name:     "nil value",
			mock:     &mockVar{Id: "v3", Name: "nilvar", DataType: "string", Value: nil},
			wantSkip: true,
		},
		{
			name: "secret skip condition",
			mock: func() *mockVar {
				s := "masked"
				return &mockVar{Id: "v4", Name: "secret", DataType: "secret", Value: &UnionValue{String: &s}}
			}(),
			wantSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ProcessResource("test_union", tt.mock)
			require.NoError(t, err)
			if tt.wantSkip {
				_, hasValue := result.Attributes["value"]
				assert.False(t, hasValue, "expected value attribute to be absent")
				return
			}
			valBlock, ok := result.Attributes["value"].(map[string]interface{})
			require.True(t, ok, "value should be a map")
			assert.Equal(t, tt.wantVal, valBlock[tt.wantKey])
		})
	}
}

// TestSDK_Float32InTypeDiscriminatedBlock verifies float32 values are recognized
// in the type key map and converted to float64 for consistent downstream handling.
func TestSDK_Float32InTypeDiscriminatedBlock(t *testing.T) {
	type mockWithFloat32 struct {
		Id    string
		Name  string
		Value interface{}
	}

	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform: "pingone", Service: "davinci",
			ResourceType: "test_f32", APIType: "Test",
			Name: "Test", ShortName: "test", Version: "1.0",
		},
		API: schema.APIDefinition{IDField: "id", NameField: "name"},
		Attributes: []schema.AttributeDefinition{
			{Name: "Id", TerraformName: "id", Type: "string", SourcePath: "Id"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{
				Name: "Value", TerraformName: "value", Type: "type_discriminated_block",
				SourcePath: "Value",
				TypeDiscriminatedBlock: &schema.TypeDiscriminatedBlockConfig{
					TypeKeyMap: map[string]string{
						"float32": "float32",
						"float64": "float64",
						"string":  "string",
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	// float32 value should be recognized and converted to float64
	mock := &mockWithFloat32{Id: "v1", Name: "f32", Value: float32(3.14)}
	result, err := p.ProcessResource("test_f32", mock)
	require.NoError(t, err)
	valBlock, ok := result.Attributes["value"].(map[string]interface{})
	require.True(t, ok, "value should be a map")
	val := valBlock["float32"]
	_, isF64 := val.(float64)
	assert.True(t, isF64, "float32 value should be converted to float64")
}

// TestSDK_Float32UnionUnwrapInTDB verifies that a union struct with a *float32
// field is correctly unwrapped and processed in a type_discriminated_block.
func TestSDK_Float32UnionUnwrapInTDB(t *testing.T) {
	type UnionValue struct {
		Bool    *bool
		Float32 *float32
		String  *string
	}

	type mockVar struct {
		Id       string
		Name     string
		DataType string
		Value    *UnionValue
	}

	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform: "pingone", Service: "davinci",
			ResourceType: "test_f32_union", APIType: "Test",
			Name: "Test", ShortName: "test", Version: "1.0",
		},
		API: schema.APIDefinition{IDField: "id", NameField: "name"},
		Attributes: []schema.AttributeDefinition{
			{Name: "Id", TerraformName: "id", Type: "string", SourcePath: "Id"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
			{Name: "DataType", TerraformName: "data_type", Type: "string", SourcePath: "DataType"},
			{
				Name: "Value", TerraformName: "value", Type: "type_discriminated_block",
				SourcePath: "Value",
				TypeDiscriminatedBlock: &schema.TypeDiscriminatedBlockConfig{
					TypeKeyMap: map[string]string{
						"string":  "string",
						"bool":    "bool",
						"float32": "float32",
					},
				},
			},
		},
	}

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	// Float32 value via union struct
	f32Val := float32(42.0)
	mock := &mockVar{Id: "v1", Name: "numvar", DataType: "number", Value: &UnionValue{Float32: &f32Val}}
	result, err := p.ProcessResource("test_f32_union", mock)
	require.NoError(t, err)
	valBlock, ok := result.Attributes["value"].(map[string]interface{})
	require.True(t, ok, "value should be a map")
	// float32(42.0) should be converted via float32 path
	_, hasFloat32 := valBlock["float32"]
	assert.True(t, hasFloat32, "should have float32 key in value block")
}

// TestSDK_PointerFieldsHandled verifies that optional pointer fields
// (e.g., *string, *bool, *float32) are correctly dereferenced.
func TestSDK_PointerFieldsHandled(t *testing.T) {
	type mockWithPtrs struct {
		Id          string
		Name        string
		DisplayName *string
		Mutable     *bool
		Min         *float32
	}

	def := sdkTestDef("test_ptrs", []schema.AttributeDefinition{
		{Name: "Id", TerraformName: "id", Type: "string", SourcePath: "Id"},
		{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
		{Name: "DisplayName", TerraformName: "display_name", Type: "string", SourcePath: "DisplayName"},
		{Name: "Mutable", TerraformName: "mutable", Type: "bool", SourcePath: "Mutable"},
		{Name: "Min", TerraformName: "min", Type: "number", SourcePath: "Min"},
	})

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	// Non-nil pointers
	display := "My Variable"
	mutable := true
	min := float32(0)
	mock := &mockWithPtrs{Id: "v1", Name: "test", DisplayName: &display, Mutable: &mutable, Min: &min}
	result, err := p.ProcessResource("test_ptrs", mock)
	require.NoError(t, err)
	assert.Equal(t, "My Variable", result.Attributes["display_name"])
	assert.Equal(t, true, result.Attributes["mutable"])

	// Nil pointers should produce no attribute
	mock2 := &mockWithPtrs{Id: "v2", Name: "test2"}
	result2, err := p.ProcessResource("test_ptrs", mock2)
	require.NoError(t, err)
	_, hasDisplay := result2.Attributes["display_name"]
	assert.False(t, hasDisplay, "nil *string should produce no attribute")
	_, hasMutable := result2.Attributes["mutable"]
	assert.False(t, hasMutable, "nil *bool should produce no attribute")
}

// BigFloatWrapper mirrors SDK types like BigFloatUnquoted that embed *big.Float.
type BigFloatWrapper struct {
	*big.Float
}

// TestSDK_BigFloatNumberConversion verifies that SDK wrapper types embedding
// *big.Float (e.g., BigFloatUnquoted used by MinZoom/MaxZoom) are correctly
// converted to float64 for "number" type attributes.
func TestSDK_BigFloatNumberConversion(t *testing.T) {
	type mockGraphData struct {
		Id      string
		Name    string
		MinZoom *BigFloatWrapper
		MaxZoom *BigFloatWrapper
		Zoom    float32
	}

	def := sdkTestDef("test_bigfloat", []schema.AttributeDefinition{
		{Name: "Id", TerraformName: "id", Type: "string", SourcePath: "Id"},
		{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name"},
		{Name: "MinZoom", TerraformName: "min_zoom", Type: "number", SourcePath: "MinZoom"},
		{Name: "MaxZoom", TerraformName: "max_zoom", Type: "number", SourcePath: "MaxZoom"},
		{Name: "Zoom", TerraformName: "zoom", Type: "number", SourcePath: "Zoom"},
	})

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register(def))
	p := core.NewProcessor(registry)

	minZoom := &BigFloatWrapper{big.NewFloat(1e-50)}
	maxZoom := &BigFloatWrapper{big.NewFloat(1e+50)}
	mock := &mockGraphData{
		Id:      "gd1",
		Name:    "test-graph",
		MinZoom: minZoom,
		MaxZoom: maxZoom,
		Zoom:    1.0,
	}

	result, err := p.ProcessResource("test_bigfloat", mock)
	require.NoError(t, err)

	// MinZoom and MaxZoom should be extracted as float64
	assert.InDelta(t, 1e-50, result.Attributes["min_zoom"], 1e-55,
		"MinZoom BigFloatWrapper should convert to float64")
	assert.InDelta(t, 1e+50, result.Attributes["max_zoom"], 1e+45,
		"MaxZoom BigFloatWrapper should convert to float64")
	// Regular float32 still works
	assert.InDelta(t, 1.0, result.Attributes["zoom"], 0.001,
		"float32 Zoom should convert to float64")

	// Nil BigFloatWrapper should produce no attribute
	mock2 := &mockGraphData{Id: "gd2", Name: "test2", Zoom: 1.0}
	result2, err := p.ProcessResource("test_bigfloat", mock2)
	require.NoError(t, err)
	_, hasMin := result2.Attributes["min_zoom"]
	assert.False(t, hasMin, "nil *BigFloatWrapper should produce no attribute")
}
