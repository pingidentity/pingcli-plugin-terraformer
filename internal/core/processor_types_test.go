package core_test

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AllTypesResource covers all attribute types supported by the processor.
type AllTypesResource struct {
	ID       string
	Name     string
	StrVal   string
	IntVal   int64
	UintVal  uint64
	FloatVal float64
	BoolVal  bool
	ObjVal   struct{ Inner string }
	ListVal  []string
	MapVal   map[string]string
	SetVal   []int
}

func TestProcessorAllAttributeTypes(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_types",
			APIType:      "TestTypes",
			Name:         "Test All Types",
			ShortName:    "test_types",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestTypes",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{Name: "StrVal", TerraformName: "str_val", Type: "string"},
			{Name: "IntVal", TerraformName: "int_val", Type: "number"},
			{Name: "UintVal", TerraformName: "uint_val", Type: "number"},
			{Name: "FloatVal", TerraformName: "float_val", Type: "number"},
			{Name: "BoolVal", TerraformName: "bool_val", Type: "bool"},
			{Name: "ObjVal", TerraformName: "obj_val", Type: "object"},
			{Name: "ListVal", TerraformName: "list_val", Type: "list"},
			{Name: "MapVal", TerraformName: "map_val", Type: "map"},
			{Name: "SetVal", TerraformName: "set_val", Type: "set"},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))
	processor := core.NewProcessor(reg)

	mock := &AllTypesResource{
		ID:       "types-1",
		Name:     "all_types",
		StrVal:   "hello",
		IntVal:   42,
		UintVal:  99,
		FloatVal: 3.14,
		BoolVal:  true,
		ObjVal:   struct{ Inner string }{Inner: "nested"},
		ListVal:  []string{"a", "b", "c"},
		MapVal:   map[string]string{"k1": "v1", "k2": "v2"},
		SetVal:   []int{1, 2, 3},
	}

	result, err := processor.ProcessResource("pingone_davinci_test_types", mock)
	require.NoError(t, err)

	// String
	assert.Equal(t, "hello", result.Attributes["str_val"])

	// Number: int64
	assert.Equal(t, int64(42), result.Attributes["int_val"])

	// Number: uint64 -> int64
	assert.Equal(t, int64(99), result.Attributes["uint_val"])

	// Number: float64
	assert.Equal(t, 3.14, result.Attributes["float_val"])

	// Bool
	assert.Equal(t, true, result.Attributes["bool_val"])

	// Object - returned as-is
	assert.NotNil(t, result.Attributes["obj_val"])

	// List
	listVal, ok := result.Attributes["list_val"].([]interface{})
	require.True(t, ok, "list_val should be []interface{}")
	assert.Len(t, listVal, 3)

	// Map
	assert.NotNil(t, result.Attributes["map_val"])

	// Set
	setVal, ok := result.Attributes["set_val"].([]interface{})
	require.True(t, ok, "set_val should be []interface{}")
	assert.Len(t, setVal, 3)
}

// IntAsStringResource tests string conversion of non-string kinds.
type IntAsStringResource struct {
	ID   string
	Name string
	Code int
}

func TestProcessorNumberAsString(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_nas",
			APIType:      "TestNAS",
			Name:         "Test Number As String",
			ShortName:    "test_nas",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestNAS",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{Name: "Code", TerraformName: "code", Type: "string"},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))
	processor := core.NewProcessor(reg)

	mock := &IntAsStringResource{ID: "nas-1", Name: "test", Code: 404}
	result, err := processor.ProcessResource("pingone_davinci_test_nas", mock)
	require.NoError(t, err)

	// Non-string kind converted to string via fmt.Sprintf.
	assert.Equal(t, "404", result.Attributes["code"])
}

func TestProcessorUnknownType(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_unk",
			APIType:      "TestUnk",
			Name:         "Test Unknown Type",
			ShortName:    "test_unk",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestUnk",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			{Name: "StrVal", TerraformName: "str_val", Type: "totally_unknown_type"},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))
	processor := core.NewProcessor(reg)

	mock := &AllTypesResource{ID: "unk-1", Name: "test_unk", StrVal: "hello"}
	result, err := processor.ProcessResource("pingone_davinci_test_unk", mock)
	// Unknown type for a valid field causes extraction to skip (error).
	require.NoError(t, err)
	// str_val should be absent because convertValue returned error.
	_, has := result.Attributes["str_val"]
	assert.False(t, has, "unknown type should skip the attribute")
}

func TestProcessorTypeMismatchSkipsAttribute(t *testing.T) {
	def := &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_davinci_test_mm",
			APIType:      "TestMM",
			Name:         "Test Type Mismatch",
			ShortName:    "test_mm",
			Version:      "1.0",
		},
		API: schema.APIDefinition{
			SDKPackage: "github.com/test",
			SDKType:    "TestMM",
			IDField:    "id",
			NameField:  "name",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string"},
			{Name: "Name", TerraformName: "name", Type: "string"},
			// StrVal is a string field but we declare it as bool.
			{Name: "StrVal", TerraformName: "str_as_bool", Type: "bool"},
			// StrVal declared as list.
			{Name: "IntVal", TerraformName: "int_as_list", Type: "list"},
			// StrVal declared as number.
			{Name: "BoolVal", TerraformName: "bool_as_number", Type: "number"},
			// String as map.
			{Name: "FloatVal", TerraformName: "float_as_map", Type: "map"},
			// String as set.
			{Name: "IntVal", TerraformName: "int_as_set", Type: "set"},
		},
	}

	reg := schema.NewRegistry()
	require.NoError(t, reg.Register(def))
	processor := core.NewProcessor(reg)

	mock := &AllTypesResource{
		ID:       "mm-1",
		Name:     "mismatch",
		StrVal:   "notabool",
		IntVal:   42,
		BoolVal:  true,
		FloatVal: 1.5,
	}

	result, err := processor.ProcessResource("pingone_davinci_test_mm", mock)
	require.NoError(t, err)

	// Mismatched types should be skipped (extraction error logged as continue).
	_, hasBool := result.Attributes["str_as_bool"]
	assert.False(t, hasBool, "string field declared as bool should be skipped")

	_, hasList := result.Attributes["int_as_list"]
	assert.False(t, hasList, "int field declared as list should be skipped")

	_, hasNumber := result.Attributes["bool_as_number"]
	assert.False(t, hasNumber, "bool field declared as number should be skipped")

	_, hasMap := result.Attributes["float_as_map"]
	assert.False(t, hasMap, "float field declared as map should be skipped")
}

func TestProcessorUnregisteredResourceType(t *testing.T) {
	reg := schema.NewRegistry()
	processor := core.NewProcessor(reg)

	_, err := processor.ProcessResource("nonexistent_resource", struct{ ID string }{ID: "1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent_resource")
}

func TestProcessorResourceListNonSlice(t *testing.T) {
	reg := schema.NewRegistry()
	processor := core.NewProcessor(reg)

	_, err := processor.ProcessResourceList("test", "not_a_slice")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected slice")
}
