package pingone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVariableResourceRegistered verifies the variable handler is in the dispatch table.
func TestVariableResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_davinci_variable"))
}

// TestVariableResourceHandlerFunctions verifies list and get functions are set.
func TestVariableResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_davinci_variable"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}
