package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalAttributeKey_ReturnsTerraformName(t *testing.T) {
	attr := AttributeDefinition{
		Name:          "MyField",
		TerraformName: "my_field",
		Type:          "string",
	}
	assert.Equal(t, "my_field", CanonicalAttributeKey(attr))
}

func TestCanonicalAttributeKey_PanicsWhenEmpty(t *testing.T) {
	attr := AttributeDefinition{
		Name: "MyField",
		Type: "string",
	}
	assert.Panics(t, func() {
		CanonicalAttributeKey(attr)
	})
}
