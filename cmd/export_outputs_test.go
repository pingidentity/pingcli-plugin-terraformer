// Copyright © 2025 Ping Identity Corporation

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- parseOutputPath tests ----

func TestParseOutputPath(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantOk       bool
		wantResType  string
		wantLabel    string
		wantAttrPath string
	}{
		{
			name:         "valid 3-segment path",
			raw:          "pingone_davinci_flow.my_flow.id",
			wantOk:       true,
			wantResType:  "pingone_davinci_flow",
			wantLabel:    "my_flow",
			wantAttrPath: "id",
		},
		{
			name:         "valid 4-segment nested path",
			raw:          "pingone_davinci_flow.my_flow.settings.csp",
			wantOk:       true,
			wantResType:  "pingone_davinci_flow",
			wantLabel:    "my_flow",
			wantAttrPath: "settings.csp",
		},
		{
			name:         "glob pattern preserved in label",
			raw:          "pingone_davinci_flow.*.id",
			wantOk:       true,
			wantResType:  "pingone_davinci_flow",
			wantLabel:    "*",
			wantAttrPath: "id",
		},
		{
			name:   "only 2 segments — invalid",
			raw:    "pingone_davinci_flow.id",
			wantOk: false,
		},
		{
			name:   "empty string — invalid",
			raw:    "",
			wantOk: false,
		},
		{
			name:   "single segment — invalid",
			raw:    "pingone_davinci_flow",
			wantOk: false,
		},
		{
			name:   "missing attr after second dot — invalid",
			raw:    "pingone_davinci_flow.my_flow.",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseOutputPath(tt.raw)
			assert.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				assert.Equal(t, tt.wantResType, got.resourceType)
				assert.Equal(t, tt.wantLabel, got.labelPattern)
				assert.Equal(t, tt.wantAttrPath, got.attrPath)
			}
		})
	}
}

// ---- collectOutputPaths tests ----

func TestCollectOutputPaths_FlagsOnly(t *testing.T) {
	paths, err := collectOutputPaths([]string{
		"pingone_davinci_flow.my_flow.id",
		"pingone_davinci_flow.my_flow.id", // duplicate
		"pingone_davinci_flow.other_flow.name",
	}, "")
	require.NoError(t, err)
	assert.Equal(t, []string{
		"pingone_davinci_flow.my_flow.id",
		"pingone_davinci_flow.other_flow.name",
	}, paths)
}

func TestCollectOutputPaths_File(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "outputs.txt")
	content := "pingone_davinci_flow.flow_a.id\n# comment line\n\npingone_davinci_flow.flow_b.name\n"
	require.NoError(t, os.WriteFile(tmp, []byte(content), 0600))

	paths, err := collectOutputPaths(nil, tmp)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"pingone_davinci_flow.flow_a.id",
		"pingone_davinci_flow.flow_b.name",
	}, paths)
}

func TestCollectOutputPaths_MergeAndDedup(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "outputs.txt")
	require.NoError(t, os.WriteFile(tmp, []byte("pingone_davinci_flow.flow_a.id\n"), 0600))

	paths, err := collectOutputPaths([]string{
		"pingone_davinci_flow.flow_a.id", // already in file — should dedup
		"pingone_davinci_flow.flow_b.name",
	}, tmp)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"pingone_davinci_flow.flow_a.id",
		"pingone_davinci_flow.flow_b.name",
	}, paths)
}

func TestCollectOutputPaths_MissingFile(t *testing.T) {
	_, err := collectOutputPaths(nil, "/nonexistent/path/outputs.txt")
	assert.Error(t, err)
}

// ---- buildOutputs tests ----

func makeResult(resourceType string, labels ...string) *core.ExportResult {
	resources := make([]*core.ResourceData, len(labels))
	for i, l := range labels {
		resources[i] = &core.ResourceData{Label: l}
	}
	return &core.ExportResult{
		ResourcesByType: []*core.ExportedResourceData{
			{ResourceType: resourceType, Resources: resources},
		},
	}
}

func TestBuildOutputs_ExactMatch(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__my_flow")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{"pingone_davinci_flow.pingcli__my_flow.id"}, result, logger)
	require.Len(t, outputs, 1)
	assert.Equal(t, "pingone_davinci_flow__pingcli__my_flow__id", outputs[0].Name)
	assert.Equal(t, "pingone_davinci_flow.pingcli__my_flow.id", outputs[0].Value)
	assert.Equal(t, "The id of pingone_davinci_flow pingcli__my_flow", outputs[0].Description)
	assert.Empty(t, logger.warnings)
}

func TestBuildOutputs_GlobMatchesAll(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__flow_a", "pingcli__flow_b")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{"pingone_davinci_flow.*.id"}, result, logger)
	require.Len(t, outputs, 2)
	// Sorted by name
	assert.Equal(t, "pingone_davinci_flow__pingcli__flow_a__id", outputs[0].Name)
	assert.Equal(t, "pingone_davinci_flow__pingcli__flow_b__id", outputs[1].Name)
	assert.Empty(t, logger.warnings)
}

func TestBuildOutputs_NoMatchWarns(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__flow_a")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{"pingone_davinci_flow.no_such_label.id"}, result, logger)
	assert.Empty(t, outputs)
	require.Len(t, logger.warnings, 1)
	assert.Contains(t, logger.warnings[0], "matched no")
}

func TestBuildOutputs_UnknownResourceTypeWarns(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__flow_a")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{"pingone_davinci_variable.*.id"}, result, logger)
	assert.Empty(t, outputs)
	require.Len(t, logger.warnings, 1)
	assert.Contains(t, logger.warnings[0], "not found in export results")
}

func TestBuildOutputs_MalformedPathWarns(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__flow_a")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{"bad_path"}, result, logger)
	assert.Empty(t, outputs)
	require.Len(t, logger.warnings, 1)
	assert.Contains(t, logger.warnings[0], "malformed")
}

func TestBuildOutputs_NestedAttrPath(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__my_flow")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{"pingone_davinci_flow.pingcli__my_flow.settings.csp"}, result, logger)
	require.Len(t, outputs, 1)
	assert.Equal(t, "pingone_davinci_flow__pingcli__my_flow__settings__csp", outputs[0].Name)
	assert.Equal(t, "pingone_davinci_flow.pingcli__my_flow.settings.csp", outputs[0].Value)
}

func TestBuildOutputs_SortedOutput(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__z_flow", "pingcli__a_flow")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{"pingone_davinci_flow.*.id"}, result, logger)
	require.Len(t, outputs, 2)
	assert.True(t, outputs[0].Name < outputs[1].Name, "outputs should be sorted by name")
}

func TestBuildOutputs_EmptyWhenNoPaths(t *testing.T) {
	result := makeResult("pingone_davinci_flow", "pingcli__flow_a")
	logger := &mockLogger{}

	outputs := buildOutputs([]string{}, result, logger)
	assert.Empty(t, outputs)
}
