package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The de-interpolation equivalence (issue #158): the exporter renders bare
// Terraform traversals (var.x, type.label.field) inside jsonencode blobs where
// older output rendered quoted interpolation-only strings. The two forms
// evaluate identically, so the compare tool treats them as equal — while
// leaving partial interpolations and non-traversal expressions untouched.

func TestDeinterpolateValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"var ref", "${var.x}", "var.x"},
		{"resource ref", "${pingone_davinci_flow.my_flow.id}", "pingone_davinci_flow.my_flow.id"},
		{"multi-segment ref", "${a.b.c.d}", "a.b.c.d"},
		{"bare value unchanged", "var.x", "var.x"},
		{"plain string unchanged", "hello", "hello"},
		{"partial prefix not normalized", "prefix-${var.x}", "prefix-${var.x}"},
		{"partial suffix not normalized", "${var.x} suffix", "${var.x} suffix"},
		{"embedded mid-string not normalized", "a ${var.x} b", "a ${var.x} b"},
		{"two interpolations not normalized", "${var.x}${var.y}", "${var.x}${var.y}"},
		{"template literal not normalized", "${jsonencode({})}", "${jsonencode({})}"},
		{"func call not normalized", "${lookup(var.x)}", "${lookup(var.x)}"},
		{"index not normalized", "${var.list[0]}", "${var.list[0]}"},
		{"empty string unchanged", "", ""},
		{"unclosed not normalized", "${var.x", "${var.x"},
		{"unopened not normalized", "var.x}", "var.x}"},
		{"empty segments not normalized", "${var..x}", "${var..x}"},
		{"space in ref not normalized", "${var. x}", "${var. x}"},
		{"dollar alone unchanged", "$", "$"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, deinterpolateValue(tt.input))
		})
	}
}

func TestDeinterpolateBlobText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"connector-shaped blob",
			`jsonencode({ "code" : { "type" : "string", "value" : "${var.x}" } })`,
			`jsonencode({ "code" : { "type" : "string", "value" : var.x } })`,
		},
		{
			"multiple refs in one blob",
			`jsonencode({ "value" : "${var.a}", "other" : "${var.b}" })`,
			`jsonencode({ "value" : var.a, "other" : var.b })`,
		},
		{
			"resource refs in blob",
			`jsonencode({ "subFlowId" : { "value" : "${pingone_davinci_flow.f.id}" } })`,
			`jsonencode({ "subFlowId" : { "value" : pingone_davinci_flow.f.id } })`,
		},
		{
			"bare blob unchanged",
			`jsonencode({ "value" : var.x })`,
			`jsonencode({ "value" : var.x })`,
		},
		{
			"plain string values untouched",
			`jsonencode({ "key" : "value" })`,
			`jsonencode({ "key" : "value" })`,
		},
		{
			"partial interpolation inside blob untouched",
			`jsonencode({ "text" : "prefix-${var.x}" })`,
			`jsonencode({ "text" : "prefix-${var.x}" })`,
		},
		{
			"rich-text Slate wrapper untouched",
			`jsonencode({ "value" : "<p>hello ${var.x}</p>" })`,
			`jsonencode({ "value" : "<p>hello ${var.x}</p>" })`,
		},
		{
			"jsonencode call inside string not stripped",
			`jsonencode({ "expr" : "${jsonencode({})}" })`,
			`jsonencode({ "expr" : "${jsonencode({})}" })`,
		},
		{
			"no quotes at all unchanged",
			`jsonencode(123)`,
			`jsonencode(123)`,
		},
		{
			"unterminated quote unchanged",
			`jsonencode({ "value" : "${var.x }`,
			`jsonencode({ "value" : "${var.x }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, deinterpolateBlobText(tt.input))
		})
	}
}

func TestTraversalShaped(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"var ref", "var.x", true},
		{"resource ref", "pingone_davinci_flow.my_flow.id", true},
		{"single ident", "x", true},
		{"digits and underscores", "var.my_var_2", true},
		{"dashes", "pingone_davinci_flow.my-flow", true},
		{"empty", "", false},
		{"leading dot", ".var.x", false},
		{"trailing dot", "var.x.", false},
		{"double dot", "var..x", false},
		{"space", "var. x", false},
		{"brackets", "var.list[0]", false},
		{"parens", "lookup(var.x)", false},
		{"quotes", "var.\"x\"", false},
		{"braces", "var.x}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, traversalShaped(tt.input))
		})
	}
}

// ── End-to-end equivalence through the HCL comparator ──

func TestCompareHCL_DeInterpolationEquivalence(t *testing.T) {
	tests := []struct {
		name      string
		expected  string
		actual    string
		wantDiffs bool
	}{
		{
			"connector property: quoted interpolation vs bare var",
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "code" : {
      "type" : "string",
      "value" : "${var.x}"
    }
  })
}`,
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "code" : {
      "type" : "string",
      "value" : var.x
    }
  })
}`,
			false,
		},
		{
			"flow node property: quoted resource ref vs bare traversal",
			`resource "pingone_davinci_flow" "f" {
  graph_data = {
    nodes = {
      n1 = {
        properties = jsonencode({
          "subFlowId" : { "value" : "${pingone_davinci_flow.other.id}" }
        })
      }
    }
  }
}`,
			`resource "pingone_davinci_flow" "f" {
  graph_data = {
    nodes = {
      n1 = {
        properties = jsonencode({
          "subFlowId" : { "value" : pingone_davinci_flow.other.id }
        })
      }
    }
  }
}`,
			false,
		},
		{
			"plain attribute: quoted interpolation vs bare var",
			`resource "pingone_davinci_variable" "v" {
  value = "${var.x}"
}`,
			`resource "pingone_davinci_variable" "v" {
  value = var.x
}`,
			false,
		},
		{
			"partial interpolation still a mismatch",
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "text" : { "value" : "prefix-${var.x}" }
  })
}`,
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "text" : { "value" : "prefix-var.x" }
  })
}`,
			true,
		},
		{
			"partial interpolation suffix still a mismatch",
			`resource "pingone_davinci_variable" "v" {
  value = "${var.x} suffix"
}`,
			`resource "pingone_davinci_variable" "v" {
  value = "var.x suffix"
}`,
			true,
		},
		{
			"different var names still a mismatch",
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "code" : { "type" : "string", "value" : "${var.old_name}" }
  })
}`,
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "code" : { "type" : "string", "value" : var.new_name }
  })
}`,
			true,
		},
		{
			"bare var vs plain string still a mismatch",
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "code" : { "type" : "string", "value" : "${var.x}" }
  })
}`,
			`resource "pingone_davinci_connector_instance" "c" {
  properties = jsonencode({
    "code" : { "type" : "string", "value" : "literal" }
  })
}`,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareHCL(tt.expected, tt.actual)
			require.NoError(t, err)
			if tt.wantDiffs {
				assert.True(t, result.HasBreakingDiffs(),
					"expected breaking diffs, got: %s", result.Summary())
			} else {
				assert.False(t, result.HasBreakingDiffs(),
					"expected no breaking diffs, got: %s", result.Summary())
			}
		})
	}
}

// ── End-to-end equivalence through the JSON comparator ──

func TestCompareJSON_DeInterpolationEquivalence(t *testing.T) {
	tests := []struct {
		name      string
		expected  string
		actual    string
		wantDiffs bool
	}{
		{
			"blob wrapping quoted ref vs bare token",
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"${\"${var.x}\"}"}}}}`,
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"${var.x}"}}}}`,
			false,
		},
		{
			"jsonencode blob: quoted ref inside vs bare token inside",
			`{"resource":{"pingone_davinci_connector_instance":{"c":{"properties":"${jsonencode({\"code\":{\"value\":\"${var.x}\"}})}"}}}}`,
			`{"resource":{"pingone_davinci_connector_instance":{"c":{"properties":"${jsonencode({\"code\":{\"value\":var.x}})}"}}}}`,
			false,
		},
		{
			"plain quoted interpolation vs bare token",
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"${var.x}"}}}}`,
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"var.x"}}}}`,
			false,
		},
		{
			"partial interpolation still a mismatch",
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"prefix-${var.x}"}}}}`,
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"prefix-var.x"}}}}`,
			true,
		},
		{
			"different var names still a mismatch",
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"${var.old}"}}}}`,
			`{"resource":{"pingone_davinci_variable":{"v":{"expr":"var.new"}}}}`,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareJSON(tt.expected, tt.actual)
			require.NoError(t, err)
			if tt.wantDiffs {
				assert.True(t, result.HasBreakingDiffs(),
					"expected breaking diffs, got: %s", result.Summary())
			} else {
				assert.False(t, result.HasBreakingDiffs(),
					"expected no breaking diffs, got: %s", result.Summary())
			}
		})
	}
}

// The committed regression snapshot still carries the old interpolation-only
// form; a base-vs-PR directory comparison over representative blob content
// must classify the de-interpolation as non-breaking.
func TestCompareHCLGeneric_SnapshotStyleBlob_Equivalence(t *testing.T) {
	expected := `resource "pingone_davinci_connector_instance" "codeSnippet" {
  environment_id = var.pingone_environment_id
  name           = "codeSnippet"
  connector = {
    id = "codeSnippetConnector"
  }
  properties = jsonencode({
    "code" : {
      "type" : "string",
      "value" : "${var.davinci_connection_Code_0020_Snippet_code}"
    },
    "inputSchema" : {
      "type" : "string",
      "value" : "${var.davinci_connection_Code_0020_Snippet_inputSchema}"
    }
  })
}`
	actual := `resource "pingone_davinci_connector_instance" "codeSnippet" {
  environment_id = var.pingone_environment_id
  name           = "codeSnippet"
  connector = {
    id = "codeSnippetConnector"
  }
  properties = jsonencode({
    "code" : {
      "type" : "string",
      "value" : var.davinci_connection_Code_0020_Snippet_code
    },
    "inputSchema" : {
      "type" : "string",
      "value" : var.davinci_connection_Code_0020_Snippet_inputSchema
    }
  })
}`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingDiffs(),
		"de-interpolation of snapshot-shaped blobs must be non-breaking, got: %s", result.Summary())
}
