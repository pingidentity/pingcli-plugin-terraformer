package core

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomHandlerQueueEmpty(t *testing.T) {
	q := NewCustomHandlerQueue()
	assert.Empty(t, q.HandlerNames())
	assert.Empty(t, q.TransformNames())
}

func TestCustomHandlerQueueAddHandler(t *testing.T) {
	q := NewCustomHandlerQueue()
	q.AddHandler("beta", StubHandler("beta"))
	q.AddHandler("alpha", StubHandler("alpha"))
	assert.Equal(t, []string{"alpha", "beta"}, q.HandlerNames())
}

func TestCustomHandlerQueueAddTransform(t *testing.T) {
	q := NewCustomHandlerQueue()
	q.AddTransform("zebra", StubTransform())
	q.AddTransform("apple", StubTransform())
	assert.Equal(t, []string{"apple", "zebra"}, q.TransformNames())
}

func TestCustomHandlerQueueLoadInto(t *testing.T) {
	q := NewCustomHandlerQueue()
	q.AddHandler("myHandler", StubHandler("myHandler"))
	q.AddTransform("myTransform", StubTransform())

	reg := NewCustomHandlerRegistry()
	q.LoadInto(reg)

	assert.True(t, reg.HasHandler("myHandler"))
	assert.True(t, reg.HasTransform("myTransform"))
}

func TestCustomHandlerQueueLoadIntoMultiple(t *testing.T) {
	q := NewCustomHandlerQueue()
	q.AddHandler("h1", StubHandler("h1"))
	q.AddHandler("h2", StubHandler("h2"))
	q.AddTransform("t1", StubTransform())
	q.AddTransform("t2", StubTransform())
	q.AddTransform("t3", StubTransform())

	reg := NewCustomHandlerRegistry()
	q.LoadInto(reg)

	assert.Equal(t, 2, len(reg.ListHandlers()))
	assert.Equal(t, 3, len(reg.ListTransforms()))
}

func TestStubHandlerReturnsError(t *testing.T) {
	fn := StubHandler("testHandler")
	_, err := fn(nil, &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{ResourceType: "test_resource"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
	assert.Contains(t, err.Error(), "testHandler")
	assert.Contains(t, err.Error(), "test_resource")
}

func TestStubTransformPassesThrough(t *testing.T) {
	fn := StubTransform()
	result, err := fn("hello", nil, &schema.AttributeDefinition{Name: "test"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestStubTransformPassesThroughNil(t *testing.T) {
	fn := StubTransform()
	result, err := fn(nil, nil, &schema.AttributeDefinition{Name: "test"}, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}
