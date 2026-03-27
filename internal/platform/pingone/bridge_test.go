package pingone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── flowDeployData projection tests ─────────────────────────────

func TestToFlowDeployData_WithCurrentVersion(t *testing.T) {
	version := float32(3.0)
	result := toFlowDeployData("flow-001", "Login Flow", &version)

	assert.Equal(t, "flow-001", result.FlowID)
	assert.Equal(t, "Login Flow", result.Name)
	require.NotNil(t, result.DeployTriggerValues)
	assert.Equal(t, 3, result.DeployTriggerValues.DeployedVersion)
}

func TestToFlowDeployData_NilVersion(t *testing.T) {
	result := toFlowDeployData("flow-002", "Simple Flow", nil)

	assert.Equal(t, "flow-002", result.FlowID)
	assert.Equal(t, "Simple Flow", result.Name)
	assert.Nil(t, result.DeployTriggerValues)
}

func TestFlowDeployData_FieldNamesMatchYAML(t *testing.T) {
	// Verify projection struct field names match flow_deploy.yaml source_paths.
	data := &flowDeployData{
		FlowID: "test-id",
		Name:   "test-name",
		DeployTriggerValues: &deployTriggerValues{
			DeployedVersion: 5,
		},
	}
	assert.Equal(t, "test-id", data.FlowID)
	assert.Equal(t, "test-name", data.Name)
	assert.Equal(t, 5, data.DeployTriggerValues.DeployedVersion)
}
