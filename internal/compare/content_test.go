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

// ── Tests for DiffExtraAttribute ──

func TestCompareHCL_ExtraAttribute(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "v" {
  name = "test"
}
`
	actual := `
resource "pingone_davinci_variable" "v" {
  name = "test"
  extra_field = "extra_value"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffExtraAttribute, result.Diffs[0].Kind)
	assert.Equal(t, "pingone_davinci_variable.v", result.Diffs[0].Resource)
	assert.Equal(t, "extra_field", result.Diffs[0].Attribute)
}

func TestCompareHCL_MultipleExtraAttributes(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "v" {
  name = "test"
}
`
	actual := `
resource "pingone_davinci_variable" "v" {
  name = "test"
  extra_1 = "value1"
  extra_2 = "value2"
}
`
	result, err := CompareHCL(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 2)
	kinds := make(map[DiffKind]int)
	for _, d := range result.Diffs {
		kinds[d.Kind]++
	}
	assert.Equal(t, 2, kinds[DiffExtraAttribute])
}

// ── Tests for ClassifyDiff ──

func TestClassifyDiff_ExtraResourceIsAcceptable(t *testing.T) {
	d := Diff{Kind: DiffExtraResource}
	severity := ClassifyDiff(d)
	assert.Equal(t, SeverityAcceptable, severity)
}

func TestClassifyDiff_ExtraAttributeIsAcceptable(t *testing.T) {
	d := Diff{Kind: DiffExtraAttribute}
	severity := ClassifyDiff(d)
	assert.Equal(t, SeverityAcceptable, severity)
}

func TestClassifyDiff_MissingResourceIsBreaking(t *testing.T) {
	d := Diff{Kind: DiffMissingResource}
	severity := ClassifyDiff(d)
	assert.Equal(t, SeverityBreaking, severity)
}

func TestClassifyDiff_MissingAttributeIsBreaking(t *testing.T) {
	d := Diff{Kind: DiffMissingAttribute}
	severity := ClassifyDiff(d)
	assert.Equal(t, SeverityBreaking, severity)
}

func TestClassifyDiff_ValueMismatchIsBreaking(t *testing.T) {
	d := Diff{Kind: DiffValueMismatch}
	severity := ClassifyDiff(d)
	assert.Equal(t, SeverityBreaking, severity)
}

func TestClassifyDiff_MissingBlockIsBreaking(t *testing.T) {
	d := Diff{Kind: DiffMissingBlock}
	severity := ClassifyDiff(d)
	assert.Equal(t, SeverityBreaking, severity)
}

func TestClassifyDiff_BlockMismatchIsBreaking(t *testing.T) {
	d := Diff{Kind: DiffBlockMismatch}
	severity := ClassifyDiff(d)
	assert.Equal(t, SeverityBreaking, severity)
}

// ── Tests for Result.HasBreakingDiffs ──

func TestResult_HasBreakingDiffs_NoDiffs(t *testing.T) {
	r := &Result{Diffs: []Diff{}}
	assert.False(t, r.HasBreakingDiffs())
}

func TestResult_HasBreakingDiffs_OnlyAcceptable(t *testing.T) {
	r := &Result{
		Diffs: []Diff{
			{Kind: DiffExtraResource},
			{Kind: DiffExtraAttribute},
		},
	}
	assert.False(t, r.HasBreakingDiffs())
}

func TestResult_HasBreakingDiffs_WithBreaking(t *testing.T) {
	r := &Result{
		Diffs: []Diff{
			{Kind: DiffExtraResource},
			{Kind: DiffMissingAttribute},
		},
	}
	assert.True(t, r.HasBreakingDiffs())
}

func TestResult_HasBreakingDiffs_OnlyBreaking(t *testing.T) {
	r := &Result{
		Diffs: []Diff{
			{Kind: DiffMissingResource},
			{Kind: DiffValueMismatch},
		},
	}
	assert.True(t, r.HasBreakingDiffs())
}

// ── Tests for CompareHCLGeneric ──

func TestCompareHCLGeneric_ResourceBlocks(t *testing.T) {
	hcl := `
resource "pingone_davinci_variable" "v" {
  name = "test"
}
`
	result, err := CompareHCLGeneric(hcl, hcl)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_ModuleBlocks(t *testing.T) {
	expected := `
module "my_module" {
  source = "./path"
  var1   = "value1"
}
`
	actual := `
module "my_module" {
  source = "./path"
  var1   = "value1"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_ModuleBlocks_MissingModule(t *testing.T) {
	expected := `
module "mod_a" {
  source = "./a"
}
module "mod_b" {
  source = "./b"
}
`
	actual := `
module "mod_a" {
  source = "./a"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	assert.Equal(t, "module.mod_b", result.Diffs[0].Resource)
}

func TestCompareHCLGeneric_VariableBlocks(t *testing.T) {
	expected := `
variable "my_var" {
  type = "string"
}
`
	actual := `
variable "my_var" {
  type = "string"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_VariableBlocks_MissingVariable(t *testing.T) {
	expected := `
variable "var_a" { type = "string" }
variable "var_b" { type = "string" }
`
	actual := `
variable "var_a" { type = "string" }
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	assert.Equal(t, "variable.var_b", result.Diffs[0].Resource)
}

func TestCompareHCLGeneric_OutputBlocks(t *testing.T) {
	expected := `
output "my_output" {
  value = "test"
}
`
	actual := `
output "my_output" {
  value = "test"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_OutputBlocks_MissingOutput(t *testing.T) {
	expected := `
output "out_a" { value = "a" }
output "out_b" { value = "b" }
`
	actual := `
output "out_a" { value = "a" }
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	assert.Equal(t, "output.out_b", result.Diffs[0].Resource)
}

func TestCompareHCLGeneric_ProviderBlocks(t *testing.T) {
	expected := `
provider "pingone" {
  client_id = "123"
}
`
	actual := `
provider "pingone" {
  client_id = "123"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_ProviderBlocks_AttributeMismatch(t *testing.T) {
	expected := `
provider "pingone" {
  client_id = "123"
}
`
	actual := `
provider "pingone" {
  client_id = "456"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffValueMismatch, result.Diffs[0].Kind)
	assert.Equal(t, "provider.pingone", result.Diffs[0].Resource)
	assert.Equal(t, "client_id", result.Diffs[0].Attribute)
}

func TestCompareHCLGeneric_DataBlocks(t *testing.T) {
	expected := `
data "pingone_resource_type" "my_data" {
  filter = "name"
}
`
	actual := `
data "pingone_resource_type" "my_data" {
  filter = "name"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_DataBlocks_MissingData(t *testing.T) {
	expected := `
data "type" "d1" { id = "1" }
data "type" "d2" { id = "2" }
`
	actual := `
data "type" "d1" { id = "1" }
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	assert.Equal(t, "data.type.d2", result.Diffs[0].Resource)
}

func TestCompareHCLGeneric_LocalsBlock(t *testing.T) {
	expected := `
locals {
  var_a = "value_a"
  var_b = "value_b"
}
`
	actual := `
locals {
  var_a = "value_a"
  var_b = "value_b"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_LocalsBlock_AttributeMismatch(t *testing.T) {
	expected := `
locals {
  var_a = "old_value"
}
`
	actual := `
locals {
  var_a = "new_value"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffValueMismatch, result.Diffs[0].Kind)
	assert.Equal(t, "locals", result.Diffs[0].Resource)
	assert.Equal(t, "var_a", result.Diffs[0].Attribute)
}

func TestCompareHCLGeneric_TerraformBlock(t *testing.T) {
	expected := `
terraform {
  required_version = ">= 1.0"
}
`
	actual := `
terraform {
  required_version = ">= 1.0"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_TerraformBlock_AttributeMismatch(t *testing.T) {
	expected := `
terraform {
  required_version = ">= 1.0"
}
`
	actual := `
terraform {
  required_version = ">= 2.0"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffValueMismatch, result.Diffs[0].Kind)
	assert.Equal(t, "terraform", result.Diffs[0].Resource)
	assert.Equal(t, "required_version", result.Diffs[0].Attribute)
}

func TestCompareHCLGeneric_ImportBlocks(t *testing.T) {
	expected := `
import {
  to = "pingone_davinci_variable.v"
  id = "12345"
}
`
	actual := `
import {
  to = "pingone_davinci_variable.v"
  id = "12345"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareHCLGeneric_ImportBlocks_Missing(t *testing.T) {
	expected := `
import {
  to = "pingone_davinci_variable.v1"
  id = "id1"
}
import {
  to = "pingone_davinci_variable.v2"
  id = "id2"
}
`
	actual := `
import {
  to = "pingone_davinci_variable.v1"
  id = "id1"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	// Resource key should use 'to' attribute
	assert.Equal(t, "import.pingone_davinci_variable.v2", result.Diffs[0].Resource)
}

func TestCompareHCLGeneric_MixedBlockTypes(t *testing.T) {
	expected := `
resource "pingone_davinci_variable" "v" {
  name = "test"
}
module "m" {
  source = "./path"
}
variable "var" {
  type = "string"
}
`
	actual := `
resource "pingone_davinci_variable" "v" {
  name = "test"
}
module "m" {
  source = "./path"
}
variable "var" {
  type = "string"
}
`
	result, err := CompareHCLGeneric(expected, actual)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}
