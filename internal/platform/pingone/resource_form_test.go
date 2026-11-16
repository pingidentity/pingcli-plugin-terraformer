package pingone

import (
	"regexp"
	"strings"
	"testing"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

func TestFormResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_form"))
}

func TestFormResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_form"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

func TestHandleFormFieldProjection_PreservesVariantFields(t *testing.T) {
	// A TEXT field: common core (type/position/key/label/required) plus
	// validation, which must survive the projection.
	text := management.NewFormFieldText(
		management.ENUMFORMFIELDTYPE_TEXT,
		*management.NewFormFieldCommonPosition(2, 0),
		"user.username",
		`[{"type":"paragraph","children":[{"text":""}]}]`,
		*management.NewFormElementValidation(),
	)
	required := true
	text.Required = &required
	validationType := management.ENUMFORMELEMENTVALIDATIONTYPE_NONE
	text.Validation.Type = &validationType

	components := management.NewFormComponents([]management.FormField{
		management.FormFieldTextAsFormField(text),
	})

	out, err := handleFormFieldProjection(components, nil, &schema.AttributeDefinition{}, nil)
	require.NoError(t, err)

	raw, ok := out.(core.RawHCLValue)
	require.True(t, ok, "expected RawHCLValue, got %T", out)
	s := string(raw)

	// HCL object shape: components = { fields = [ { ... } ] }.
	assert.True(t, strings.HasPrefix(s, "{\n  fields = ["), "must emit an HCL object with a fields list")
	// Common core fields preserved.
	assert.Contains(t, s, `type = "TEXT"`)
	assert.Contains(t, s, `key = "user.username"`)
	assert.Contains(t, s, "required = true")
	assert.Contains(t, s, "row = 2")
	// Variant-specific validation preserved.
	assert.Contains(t, s, "validation")
	// Slate label re-parsed as jsonencode(<escaped JSON>), matching the
	// provider's examples.
	assert.Contains(t, s, `label = jsonencode(`)
}

func TestHandleFormFieldProjection_NilInput(t *testing.T) {
	out, err := handleFormFieldProjection(nil, nil, &schema.AttributeDefinition{}, nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestHandleFormFieldProjection_EscapesHCLTemplateMarkers(t *testing.T) {
	slate := management.NewFormFieldSlateTextblob(
		management.ENUMFORMFIELDTYPE_SLATE_TEXTBLOB,
		*management.NewFormFieldCommonPosition(0, 0),
	)
	// Slate content containing an HCL template marker.
	content := `[{"type":"paragraph","children":[{"text":"code ${otp}"}]}]`
	slate.Content = &content

	components := management.NewFormComponents([]management.FormField{
		management.FormFieldSlateTextblobAsFormField(slate),
	})

	out, err := handleFormFieldProjection(components, nil, &schema.AttributeDefinition{}, nil)
	require.NoError(t, err)

	s := string(out.(core.RawHCLValue))
	assert.Contains(t, s, "$${otp}", "template markers must be escaped")
	// Every template marker must be doubled: an unescaped marker is a "$"
	// immediately before "{" without another "$" in front of it.
	unescaped := regexp.MustCompile(`(^|[^$])\$\{`)
	assert.NotRegexp(t, unescaped, s)
}
