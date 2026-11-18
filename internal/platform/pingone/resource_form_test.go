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
	// Slate label re-parsed and emitted as jsonencode(<the JSON object>),
	// matching the provider's examples — NOT jsonencode("<escaped string>"),
	// which would double-encode the value. (Inner key order is
	// map-iteration-dependent, so assert on semantic substrings.)
	assert.Contains(t, s, `label = jsonencode([{"`)
	assert.Contains(t, s, `"type":"paragraph"`)
	assert.NotContains(t, s, `label = jsonencode("`)
}

func TestHandleFormFieldProjection_NilInput(t *testing.T) {
	out, err := handleFormFieldProjection(nil, nil, &schema.AttributeDefinition{}, nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestRemapFormFieldKeys_CamelToSnake(t *testing.T) {
	// Every camelCase key in the table must convert; single-word keys pass
	// through; unknown keys are left untouched.
	in := map[string]interface{}{
		"type":                     "AGREEMENT",
		"key":                      "agreement",
		"inputType":                "READ_ONLY_TEXT",
		"titleEnabled":             true,
		"pollingAppearance":        "DOTS",
		"idpName":                  "Sample LinkedIn",
		"idpType":                  "LINKEDIN_OIDC",
		"idpId":                    "",
		"attributeDisabled":        false,
		"labelMode":                "FLOAT",
		"labelPasswordVerify":      "[{\"type\":\"paragraph\"}]",
		"showPasswordRequirements": true,
		"otherOptionEnabled":       true,
		"otherOptionKey":           "resend",
		"validatePhoneNumber":      false,
		"showExtension":            true,
		"extensionLabel":           "Ext.",
		"defaultCountryCode":       "US",
		"countryCodeLabel":         "Country",
		"errorMessage":             "bad",
		"fallbackText":             "scan me",
		"iconSrc":                  "https://example.com/i.png",
		"useDynamicAgreement":      true,
		"widthUnit":                "PIXEL",
		"unknownCamelKey":          "kept-as-is",
	}
	got := remapFormFieldKeys(in)

	want := map[string]interface{}{
		"type":                       "AGREEMENT",
		"key":                        "agreement",
		"input_type":                 "READ_ONLY_TEXT",
		"title_enabled":              true,
		"polling_appearance":         "DOTS",
		"idp_name":                   "Sample LinkedIn",
		"idp_type":                   "LINKEDIN_OIDC",
		"idp_id":                     "",
		"attribute_disabled":         false,
		"label_mode":                 "FLOAT",
		"label_password_verify":      "[{\"type\":\"paragraph\"}]",
		"show_password_requirements": true,
		"other_option_enabled":       true,
		"other_option_key":           "resend",
		"validate_phone_number":      false,
		"show_extension":             true,
		"extension_label":            "Ext.",
		"default_country_code":       "US",
		"country_code_label":         "Country",
		"error_message":              "bad",
		"fallback_text":              "scan me",
		"icon_src":                   "https://example.com/i.png",
		"use_dynamic_agreement":      true,
		"width_unit":                 "PIXEL",
		"unknownCamelKey":            "kept-as-is",
	}
	assert.Equal(t, want, got)
}

func TestRemapFormFieldKeys_NestedObjectsAndSlices(t *testing.T) {
	// Styles and agreement are nested objects whose keys carry the same
	// camelCase API spelling; options is a list of objects.
	in := map[string]interface{}{
		"type": "SUBMIT_BUTTON",
		"styles": map[string]interface{}{
			"alignment":                          "CENTER",
			"displayDefaultThemeButtonTextColor": false,
			"backgroundColor":                    "#69788b",
			"textColor":                          "#FFF",
			"borderColor":                        "#000",
			"widthUnit":                          "PIXEL",
			"padding": map[string]interface{}{
				"top": 1,
			},
		},
		"agreement": map[string]interface{}{
			"useDynamicAgreement": true,
		},
		"options": []interface{}{
			map[string]interface{}{"label": "A", "value": "a"},
			map[string]interface{}{"label": "B", "value": "b"},
		},
	}
	got := remapFormFieldKeys(in)

	assert.Equal(t, "#69788b", got["styles"].(map[string]interface{})["background_color"])
	assert.Equal(t, "#FFF", got["styles"].(map[string]interface{})["text_color"])
	assert.Equal(t, "#000", got["styles"].(map[string]interface{})["border_color"])
	assert.Equal(t, "PIXEL", got["styles"].(map[string]interface{})["width_unit"])
	assert.Equal(t, false, got["styles"].(map[string]interface{})["display_default_theme_button_text_color"])
	assert.Equal(t, 1, got["styles"].(map[string]interface{})["padding"].(map[string]interface{})["top"])
	assert.Equal(t, true, got["agreement"].(map[string]interface{})["use_dynamic_agreement"])
	assert.Len(t, got["options"], 2)
	assert.Equal(t, "A", got["options"].([]interface{})[0].(map[string]interface{})["label"])
}

func TestHandleFormFieldProjection_EmitsSnakeCaseKeys(t *testing.T) {
	// End-to-end: an AGREEMENT field projected through the handler must emit
	// input_type / title_enabled (snake_case) — the provider's declared
	// attribute names — never the API's camelCase spelling.
	agreement := management.NewFormFieldFormAgreement(
		management.ENUMFORMFIELDTYPE_AGREEMENT,
		*management.NewFormFieldCommonPosition(0, 0),
		"agreement",
		management.ENUMFORMAGREEMENTINPUTTYPE_READ_ONLY_TEXT,
		true,
	)

	components := management.NewFormComponents([]management.FormField{
		management.FormFieldFormAgreementAsFormField(agreement),
	})

	out, err := handleFormFieldProjection(components, nil, &schema.AttributeDefinition{}, nil)
	require.NoError(t, err)

	s := string(out.(core.RawHCLValue))
	assert.Contains(t, s, `input_type = "READ_ONLY_TEXT"`)
	assert.Contains(t, s, `title_enabled = true`)
	assert.NotContains(t, s, `inputType`)
	assert.NotContains(t, s, `titleEnabled`)
}

func TestHandleFormFieldProjection_PlainLabelStaysQuoted(t *testing.T) {
	// A dropdown whose label is plain text (not Slate JSON): the emitted HCL
	// must keep it a quoted string, not jsonencode() of anything.
	dropdown := management.NewFormFieldDropdown(
		management.ENUMFORMFIELDTYPE_DROPDOWN,
		*management.NewFormFieldCommonPosition(0, 1),
		"user.email",
		"Email",
		nil,
	)

	components := management.NewFormComponents([]management.FormField{
		management.FormFieldDropdownAsFormField(dropdown),
	})

	out, err := handleFormFieldProjection(components, nil, &schema.AttributeDefinition{}, nil)
	require.NoError(t, err)

	s := string(out.(core.RawHCLValue))
	assert.Contains(t, s, `label = "Email"`)
	assert.NotContains(t, s, `label = jsonencode(`)
}

func TestHandleFormFieldProjection_SlateLabelNotDoubleEncoded(t *testing.T) {
	// Regression: the API value for a slate label is a string whose CONTENT
	// is JSON. Emitting jsonencode("<escaped>") double-encodes it — the
	// provider stores a literal escaped blob and the UI shows mangled text.
	// The fix emits jsonencode(<the parsed object>).
	text := management.NewFormFieldText(
		management.ENUMFORMFIELDTYPE_TEXT,
		*management.NewFormFieldCommonPosition(0, 0),
		"user.email",
		`[{"type":"paragraph","children":[{"type":"i18n","key":"fields.user.email.label","defaultTranslation":"Email Address","inline":true,"children":[{"text":""}]}]}]`,
		*management.NewFormElementValidation(),
	)

	components := management.NewFormComponents([]management.FormField{
		management.FormFieldTextAsFormField(text),
	})

	out, err := handleFormFieldProjection(components, nil, &schema.AttributeDefinition{}, nil)
	require.NoError(t, err)

	s := string(out.(core.RawHCLValue))
	// The slate label must be emitted as jsonencode(<the parsed object>): the
	// i18n object appears unescaped inside jsonencode(...). (Key order inside
	// the object is map-iteration-dependent, so assert on the semantic
	// substrings rather than exact serialization.)
	assert.Contains(t, s, `label = jsonencode([{"`)
	assert.Contains(t, s, `"type":"i18n"`)
	assert.Contains(t, s, `"defaultTranslation":"Email Address"`)
	// No double-encoded label anywhere in the projection.
	assert.NotContains(t, s, `label = jsonencode("`)
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
