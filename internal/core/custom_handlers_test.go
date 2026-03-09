package core

import (
	"sync"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomHandlerRegistryRegisterAndGet(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	called := false
	reg.RegisterHandler("testHandler", func(_ interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"key": "value"}, nil
	})

	fn, err := reg.GetHandler("testHandler")
	require.NoError(t, err)
	require.NotNil(t, fn)

	result, err := fn(nil, nil)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "value", result["key"])
}

func TestCustomHandlerRegistryGetMissing(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	_, err := reg.GetHandler("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestCustomTransformRegistryRegisterAndGet(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	reg.RegisterTransform("testTransform", func(value interface{}, _ interface{}, _ *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
		s, _ := value.(string)
		return s + "_transformed", nil
	})

	fn, err := reg.GetTransform("testTransform")
	require.NoError(t, err)
	require.NotNil(t, fn)

	result, err := fn("input", nil, attrDef("test"), nil)
	require.NoError(t, err)
	assert.Equal(t, "input_transformed", result)
}

func TestCustomTransformRegistryGetMissing(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	_, err := reg.GetTransform("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestCustomHandlerRegistryHas(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	assert.False(t, reg.HasHandler("h1"))
	assert.False(t, reg.HasTransform("t1"))

	reg.RegisterHandler("h1", func(_ interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
		return nil, nil
	})
	reg.RegisterTransform("t1", func(_ interface{}, _ interface{}, _ *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
		return nil, nil
	})

	assert.True(t, reg.HasHandler("h1"))
	assert.True(t, reg.HasTransform("t1"))
}

func TestCustomHandlerRegistryList(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	reg.RegisterHandler("a", func(_ interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) { return nil, nil })
	reg.RegisterHandler("b", func(_ interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) { return nil, nil })
	reg.RegisterTransform("x", func(_ interface{}, _ interface{}, _ *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
		return nil, nil
	})

	handlers := reg.ListHandlers()
	assert.Len(t, handlers, 2)
	assert.ElementsMatch(t, []string{"a", "b"}, handlers)

	transforms := reg.ListTransforms()
	assert.Len(t, transforms, 1)
	assert.ElementsMatch(t, []string{"x"}, transforms)
}

func TestCustomHandlerRegistryOverwrite(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	reg.RegisterHandler("h", func(_ interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{"v": 1}, nil
	})

	// Overwrite
	reg.RegisterHandler("h", func(_ interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
		return map[string]interface{}{"v": 2}, nil
	})

	fn, err := reg.GetHandler("h")
	require.NoError(t, err)
	result, err := fn(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, result["v"])
}

func TestCustomHandlerRegistryConcurrency(t *testing.T) {
	reg := NewCustomHandlerRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := "handler"
			reg.RegisterHandler(name, func(_ interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
				return map[string]interface{}{"n": n}, nil
			})
			_, _ = reg.GetHandler(name)
			reg.HasHandler(name)
			_ = reg.ListHandlers()
		}(i)
	}
	wg.Wait()

	assert.True(t, reg.HasHandler("handler"))
}
