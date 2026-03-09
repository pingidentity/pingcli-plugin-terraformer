package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareHCL_Identical(t *testing.T) {
	hcl := `
resource "pingone_davinci_variable" "my_var" {
  environment_id = "env-123"
  name           = "myVar"
  flow_count     = 2
}
`
	result, err := CompareHCL(hcl, hcl)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
	assert.Equal(t, "no content differences", result.Summary())
}

func TestCompareHCL_MissingResource(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "var_a" {
  name = "a"
}
resource "pingone_davinci_variable" "var_b" {
  name = "b"
}
`
	actual := `
resource "pingone_davinci_variable" "var_a" {
  name = "a"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	assert.Equal(t, "pingone_davinci_variable.var_b", result.Diffs[0].Resource)
}

func TestCompareHCL_ExtraResource(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "var_a" {
  name = "a"
}
`
	actual := `
resource "pingone_davinci_variable" "var_a" {
  name = "a"
}
resource "pingone_davinci_variable" "var_extra" {
  name = "extra"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffExtraResource, result.Diffs[0].Kind)
}

func TestCompareHCL_MissingAttribute(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "v" {
  name           = "test"
  environment_id = "env-1"
}
`
	actual := `
resource "pingone_davinci_variable" "v" {
  name = "test"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingAttribute, result.Diffs[0].Kind)
	assert.Equal(t, "environment_id", result.Diffs[0].Attribute)
}

func TestCompareHCL_ValueMismatch(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "v" {
  name = "old_name"
}
`
	actual := `
resource "pingone_davinci_variable" "v" {
  name = "new_name"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffValueMismatch, result.Diffs[0].Kind)
	assert.Equal(t, "old_name", result.Diffs[0].Expected)
	assert.Equal(t, "new_name", result.Diffs[0].Actual)
}

func TestCompareHCL_FormattingIgnored(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "v" {
  name           = "test"
  environment_id = "env-1"
}
`
	actual := `
resource "pingone_davinci_variable" "v" {
  name = "test"
  environment_id = "env-1"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs(), "formatting differences should be ignored")
}

func TestCompareHCL_NestedBlock(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "v" {
  name = "test"
  lifecycle {
    ignore_changes = ["value"]
  }
}
`
	actual := `
resource "pingone_davinci_variable" "v" {
  name = "test"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingBlock, result.Diffs[0].Kind)
	assert.Equal(t, "lifecycle", result.Diffs[0].Attribute)
}

func TestCompareHCL_EmptyInputs(t *testing.T) {
	result, err := CompareHCL("", "")
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareModuleResources(t *testing.T) {
	expected := map[string]string{
		"variables": `resource "pingone_davinci_variable" "v" { name = "a" }`,
	}
	actual := map[string]string{
		"variables": `resource "pingone_davinci_variable" "v" { name = "b" }`,
	}
	result, err := CompareModuleResources(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffValueMismatch, result.Diffs[0].Kind)
}

func TestResult_Summary(t *testing.T) {
	r := &Result{
		Diffs: []Diff{
			{Resource: "a.b", Kind: DiffMissingResource},
			{Resource: "c.d", Kind: DiffExtraResource},
			{Resource: "e.f", Kind: DiffMissingAttribute, Attribute: "name", Expected: "x"},
			{Resource: "g.h", Kind: DiffValueMismatch, Attribute: "val", Expected: "1", Actual: "2"},
			{Resource: "i.j", Kind: DiffMissingBlock, Attribute: "lifecycle"},
			{Resource: "k.l", Kind: DiffBlockMismatch, Attribute: "value"},
		},
	}
	summary := r.Summary()
	assert.Contains(t, summary, "6 content difference(s)")
	assert.Contains(t, summary, "MISSING resource a.b")
	assert.Contains(t, summary, "EXTRA   resource c.d")
	assert.Contains(t, summary, "attribute \"name\" missing")
	assert.Contains(t, summary, "expected \"1\", got \"2\"")
	assert.Contains(t, summary, "block \"lifecycle\" missing")
	assert.Contains(t, summary, "block \"value\" content differs")
}
