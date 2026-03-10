package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type reflectTestStruct struct {
	Name     string
	Enabled  bool
	Value    interface{}
	Count    int
}

func TestReadStringField_Valid(t *testing.T) {
	s := &reflectTestStruct{Name: "hello"}
	assert.Equal(t, "hello", ReadStringField(s, "Name"))
}

func TestReadStringField_Missing(t *testing.T) {
	s := &reflectTestStruct{}
	assert.Equal(t, "", ReadStringField(s, "Missing"))
}

func TestReadStringField_WrongType(t *testing.T) {
	s := &reflectTestStruct{Enabled: true}
	assert.Equal(t, "", ReadStringField(s, "Enabled"))
}

func TestReadStringField_Nil(t *testing.T) {
	assert.Equal(t, "", ReadStringField(nil, "Name"))
}

func TestReadStringField_NonStruct(t *testing.T) {
	assert.Equal(t, "", ReadStringField("not_a_struct", "Name"))
}

func TestReadStringField_NilPointer(t *testing.T) {
	var p *reflectTestStruct
	assert.Equal(t, "", ReadStringField(p, "Name"))
}

func TestReadBoolField_Valid(t *testing.T) {
	s := &reflectTestStruct{Enabled: true}
	assert.True(t, ReadBoolField(s, "Enabled"))
}

func TestReadBoolField_Missing(t *testing.T) {
	s := &reflectTestStruct{}
	assert.False(t, ReadBoolField(s, "Missing"))
}

func TestReadBoolField_WrongType(t *testing.T) {
	s := &reflectTestStruct{Name: "hello"}
	assert.False(t, ReadBoolField(s, "Name"))
}

func TestReadBoolField_Nil(t *testing.T) {
	assert.False(t, ReadBoolField(nil, "Enabled"))
}

func TestReadInterfaceField_String(t *testing.T) {
	s := &reflectTestStruct{Value: "hello"}
	assert.Equal(t, "hello", ReadInterfaceField(s, "Value"))
}

func TestReadInterfaceField_Nil(t *testing.T) {
	s := &reflectTestStruct{Value: nil}
	assert.Nil(t, ReadInterfaceField(s, "Value"))
}

func TestReadInterfaceField_NilInput(t *testing.T) {
	assert.Nil(t, ReadInterfaceField(nil, "Value"))
}

func TestReadInterfaceField_Missing(t *testing.T) {
	s := &reflectTestStruct{}
	assert.Nil(t, ReadInterfaceField(s, "NonExistent"))
}

func TestReadInterfaceField_Map(t *testing.T) {
	m := map[string]interface{}{"key": "val"}
	s := &reflectTestStruct{Value: m}
	result := ReadInterfaceField(s, "Value")
	assert.Equal(t, m, result)
}

func TestReadInterfaceField_NonInterface(t *testing.T) {
	// Reading a non-interface field should still return its value.
	s := &reflectTestStruct{Name: "test"}
	result := ReadInterfaceField(s, "Name")
	assert.Equal(t, "test", result)
}
