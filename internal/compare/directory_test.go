package compare

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}

func TestCompareDirectories_IdenticalTFFiles(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	hcl := `resource "pingone_davinci_variable" "v" { name = "test" }`
	writeTestFile(t, baseDir, "main.tf", hcl)
	writeTestFile(t, prDir, "main.tf", hcl)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingDiffs())
	assert.Len(t, result.Files, 1)
	assert.Equal(t, FileCompared, result.Files[0].Kind)
}

func TestCompareDirectories_TFFile_MissingResource(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	baseHCL := `
resource "pingone_davinci_variable" "v1" { name = "a" }
resource "pingone_davinci_variable" "v2" { name = "b" }
`
	prHCL := `
resource "pingone_davinci_variable" "v1" { name = "a" }
`
	writeTestFile(t, baseDir, "main.tf", baseHCL)
	writeTestFile(t, prDir, "main.tf", prHCL)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.True(t, result.HasBreakingDiffs())
}

func TestCompareDirectories_MissingFile(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	writeTestFile(t, baseDir, "main.tf", `resource "a" "b" { x = "1" }`)
	// prDir has no files

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.True(t, result.HasBreakingDiffs())
	require.Len(t, result.Files, 1)
	assert.Equal(t, FileMissing, result.Files[0].Kind)
}

func TestCompareDirectories_ExtraFile(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	writeTestFile(t, baseDir, "main.tf", `resource "a" "b" { x = "1" }`)
	writeTestFile(t, prDir, "main.tf", `resource "a" "b" { x = "1" }`)
	writeTestFile(t, prDir, "extra.tf", `resource "c" "d" { y = "2" }`)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingDiffs(), "extra files are additions, not breaking")
	require.Len(t, result.Files, 2)
	extraCount := 0
	for _, f := range result.Files {
		if f.Kind == FileExtra {
			extraCount++
		}
	}
	assert.Equal(t, 1, extraCount)
}

func TestCompareDirectories_MultipleFiles(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	writeTestFile(t, baseDir, "variables.tf", `resource "a" "v1" { name = "var1" }`)
	writeTestFile(t, baseDir, "flows.tf", `resource "b" "f1" { name = "flow1" }`)
	writeTestFile(t, prDir, "variables.tf", `resource "a" "v1" { name = "var1" }`)
	writeTestFile(t, prDir, "flows.tf", `resource "b" "f1" { name = "flow1" }`)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingDiffs())
	assert.Len(t, result.Files, 2)
}

func TestCompareDirectories_TFVarsFile(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	writeTestFile(t, baseDir, "terraform.tfvars", `env_id = "abc"`)
	writeTestFile(t, prDir, "terraform.tfvars", `env_id = "abc"`)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingDiffs())
}

func TestCompareDirectories_TFVarsFile_ValueChange(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	writeTestFile(t, baseDir, "terraform.tfvars", `env_id = "abc"`)
	writeTestFile(t, prDir, "terraform.tfvars", `env_id = "xyz"`)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.True(t, result.HasBreakingDiffs(), "changed tfvars value should be breaking")
}

func TestCompareDirectories_TFJSONFile(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	j := `{"resource": {"type": {"name": {"attr": "val"}}}}`
	writeTestFile(t, baseDir, "main.tf.json", j)
	writeTestFile(t, prDir, "main.tf.json", j)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingDiffs())
}

func TestCompareDirectories_Subdirectories(t *testing.T) {
	baseDir := t.TempDir()
	prDir := t.TempDir()
	hcl := `resource "a" "b" { x = "1" }`
	writeTestFile(t, filepath.Join(baseDir, "module"), "main.tf", hcl)
	writeTestFile(t, filepath.Join(prDir, "module"), "main.tf", hcl)

	result, err := CompareDirectories(baseDir, prDir)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingDiffs())
	// The path should be relative and include the subdirectory
	require.Len(t, result.Files, 1)
	assert.Contains(t, result.Files[0].Path, "module")
}

func TestDirectoryResult_Summary(t *testing.T) {
	dr := &DirectoryResult{
		Files: []FileResult{
			{Path: "missing.tf", Kind: FileMissing},
			{Path: "extra.tf", Kind: FileExtra},
		},
	}
	summary := dr.Summary()
	assert.Contains(t, summary, "missing.tf")
	assert.Contains(t, summary, "extra.tf")
}

func TestDirectoryResult_HasBreakingDiffs_False(t *testing.T) {
	dr := &DirectoryResult{
		Files: []FileResult{
			{Path: "extra.tf", Kind: FileExtra},
		},
	}
	assert.False(t, dr.HasBreakingDiffs())
}

func TestDirectoryResult_HasBreakingDiffs_MissingFile(t *testing.T) {
	dr := &DirectoryResult{
		Files: []FileResult{
			{Path: "main.tf", Kind: FileMissing},
		},
	}
	assert.True(t, dr.HasBreakingDiffs())
}

func TestDirectoryResult_HasBreakingDiffs_BreakingContentDiff(t *testing.T) {
	dr := &DirectoryResult{
		Files: []FileResult{
			{
				Path: "main.tf",
				Kind: FileCompared,
				Result: &Result{
					Diffs: []Diff{
						{Kind: DiffMissingResource, Resource: "a.b"},
					},
				},
			},
		},
	}
	assert.True(t, dr.HasBreakingDiffs())
}
