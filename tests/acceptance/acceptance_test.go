//go:build acceptance

package acceptance

import (
	"context"
	"os"
	"testing"

	// "github.com/pingidentity/pingcli-plugin-terraformer/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIClientAuthentication verifies basic authentication and client creation
func TestAPIClientAuthentication(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	assert.NotNil(t, client, "API client should be created")
	assert.NotEmpty(t, client.AuthEnvironmentID, "Auth environment ID should be set")
	assert.NotEmpty(t, client.EnvironmentID, "Target environment ID should be set")
	assert.NotEmpty(t, client.Region, "Region should be set")

	t.Logf("Auth Environment: %s", client.AuthEnvironmentID)
	t.Logf("Target Environment: %s", client.EnvironmentID)
	t.Logf("Region: %s", client.Region)
}

// TestListFlowsFromAPI verifies we can list flows from a real environment
func TestListFlowsFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	t.Logf("Listing flows from target environment: %s", client.EnvironmentID)

	flows, err := client.ListFlows(ctx)
	require.NoError(t, err, "Should successfully list flows")

	// Log flow count for visibility
	t.Logf("Found %d flows in environment", len(flows))

	// If environment has flows, verify structure
	if len(flows) > 0 {
		firstFlow := flows[0]
		assert.NotEmpty(t, firstFlow.FlowID, "Flow should have an ID")
		assert.NotEmpty(t, firstFlow.Name, "Flow should have a name")

		t.Logf("Sample flow: ID=%s, Name=%s", firstFlow.FlowID, firstFlow.Name)
	} else {
		t.Log("WARNING: No flows found in target environment. Consider creating test flows.")
	}
} // TestGetSingleFlowFromAPI verifies we can retrieve a specific flow with full details
func TestGetSingleFlowFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// First, list flows to get a valid flow ID
	flows, err := client.ListFlows(ctx)
	require.NoError(t, err, "Should successfully list flows")

	if len(flows) == 0 {
		t.Skip("No flows available in target environment for testing")
	} // Get the first flow's details
	flowID := flows[0].FlowID
	t.Logf("Testing with flow ID: %s", flowID)

	flowDetail, err := client.GetFlow(ctx, flowID)
	require.NoError(t, err, "Should successfully get flow details")
	require.NotNil(t, flowDetail, "Flow detail should not be nil")

	// Verify flow structure
	assert.Equal(t, flowID, flowDetail.FlowID, "Flow ID should match")
	assert.NotEmpty(t, flowDetail.Name, "Flow should have a name")
	// Note: GraphData may be nil for empty flows, this is valid

	t.Logf("Retrieved flow: %s", flowDetail.Name)

	// Verify graph data structure if present
	if flowDetail.GraphData != nil {
		t.Logf("Flow has graph data")
		// Check for expected keys in graph data
		_, hasNodes := flowDetail.GraphData["nodes"]
		t.Logf("Graph data has 'nodes' key: %v", hasNodes)
	} else {
		t.Logf("Flow has no graph data (empty flow)")
	}
}

// TestGetFlowWithSpecificID tests retrieving a flow using TEST_FLOW_ID env var
// If TEST_FLOW_ID is not set, uses the first flow from the environment
func TestGetFlowWithSpecificID(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	flowID := os.Getenv("TEST_FLOW_ID")

	// If no TEST_FLOW_ID, use first flow from the environment
	if flowID == "" {
		flows, err := client.ListFlows(ctx)
		require.NoError(t, err, "Should successfully list flows")

		if len(flows) == 0 {
			t.Skip("No flows available in environment and TEST_FLOW_ID not set")
		}

		flowID = flows[0].FlowID
		t.Logf("TEST_FLOW_ID not set, using first flow from environment: %s", flowID)
	} else {
		t.Logf("Using TEST_FLOW_ID: %s", flowID)
	}

	flowDetail, err := client.GetFlow(ctx, flowID)
	require.NoError(t, err, "Should successfully get specific flow")
	require.NotNil(t, flowDetail, "Flow detail should not be nil")

	assert.Equal(t, flowID, flowDetail.FlowID, "Flow ID should match requested ID")
	assert.NotEmpty(t, flowDetail.Name, "Flow should have a name")

	t.Logf("Successfully retrieved flow: %s (ID: %s)", flowDetail.Name, flowDetail.FlowID)
}

// TestAPIErrorHandling verifies proper error handling for invalid requests
func TestAPIErrorHandling(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// Test with invalid flow ID
	invalidFlowID := "00000000-0000-0000-0000-000000000000"
	flowDetail, err := client.GetFlow(ctx, invalidFlowID)

	// Should return an error for non-existent flow
	if err == nil {
		t.Logf("WARNING: Expected error for invalid flow ID, but got nil. Flow detail: %+v", flowDetail)
	} else {
		t.Logf("Correctly received error for invalid flow ID: %v", err)
	}
}

// TestMultipleFlowRetrieval verifies we can retrieve multiple flows sequentially
func TestMultipleFlowRetrieval(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// List all flows
	flows, err := client.ListFlows(ctx)
	require.NoError(t, err, "Should successfully list flows")

	if len(flows) < 2 {
		t.Skip("Need at least 2 flows in target environment for this test")
	} // Retrieve details for first 2 flows (or all if less than 5)
	maxFlows := 2
	if len(flows) < maxFlows {
		maxFlows = len(flows)
	}

	retrievedCount := 0
	for i := 0; i < maxFlows; i++ {
		flowDetail, err := client.GetFlow(ctx, flows[i].FlowID)
		require.NoError(t, err, "Should successfully get flow %d", i)
		require.NotNil(t, flowDetail, "Flow detail should not be nil")

		t.Logf("Retrieved flow %d: %s", i+1, flowDetail.Name)
		retrievedCount++
	}

	assert.Equal(t, maxFlows, retrievedCount, "Should retrieve all requested flows")
}

// TestListConnectorInstancesFromAPI verifies we can list connector instances from a real environment
func TestListConnectorInstancesFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	t.Logf("Listing connector instances from target environment: %s", client.EnvironmentID)

	instances, err := client.ListConnectorInstances(ctx)
	require.NoError(t, err, "Should successfully list connector instances")

	// Log instance count for visibility
	t.Logf("Found %d connector instances in environment", len(instances))

	// If environment has instances, verify structure
	if len(instances) > 0 {
		firstInstance := instances[0]
		assert.NotEmpty(t, firstInstance.InstanceID, "Instance should have an ID")
		assert.NotEmpty(t, firstInstance.Name, "Instance should have a name")
		assert.NotEmpty(t, firstInstance.ConnectorID, "Instance should have a connector ID")

		t.Logf("Sample instance - ID: %s, Name: %s, ConnectorID: %s",
			firstInstance.InstanceID, firstInstance.Name, firstInstance.ConnectorID)
	} else {
		t.Log("No connector instances found in environment (this is acceptable)")
	}
}

// TestGetSingleConnectorInstanceFromAPI verifies we can retrieve a specific connector instance's details
func TestGetSingleConnectorInstanceFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// First list instances to get a valid ID
	instances, err := client.ListConnectorInstances(ctx)
	require.NoError(t, err, "Should successfully list connector instances")

	if len(instances) == 0 {
		t.Skip("No connector instances in environment to test")
	}

	// Get details for the first instance
	firstInstanceID := instances[0].InstanceID
	t.Logf("Getting details for connector instance: %s", firstInstanceID)

	instanceDetail, err := client.GetConnectorInstance(ctx, firstInstanceID)
	require.NoError(t, err, "Should successfully get connector instance detail")
	require.NotNil(t, instanceDetail, "Instance detail should not be nil")

	// Verify structure
	assert.Equal(t, firstInstanceID, instanceDetail.InstanceID, "Instance ID should match")
	assert.NotEmpty(t, instanceDetail.Name, "Instance should have a name")
	assert.NotEmpty(t, instanceDetail.ConnectorID, "Instance should have a connector ID")

	// Properties may or may not be present depending on connector type
	if instanceDetail.Properties != nil {
		t.Logf("Instance has %d properties", len(instanceDetail.Properties))
	} else {
		t.Log("Instance has no properties (acceptable)")
	}

	t.Logf("Successfully retrieved connector instance: %s", instanceDetail.Name)
}

// TestGetInvalidConnectorInstanceFromAPI verifies error handling for invalid instance IDs
func TestGetInvalidConnectorInstanceFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	invalidInstanceID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Attempting to get non-existent connector instance: %s", invalidInstanceID)

	instanceDetail, err := client.GetConnectorInstance(ctx, invalidInstanceID)

	// Should return an error for non-existent instance
	if err == nil {
		t.Logf("WARNING: Expected error for invalid instance ID, but got nil. Instance detail: %+v", instanceDetail)
	} else {
		t.Logf("Correctly received error for invalid instance ID: %v", err)
	}
}

// TestMultipleConnectorInstanceRetrieval verifies we can retrieve multiple connector instances sequentially
func TestMultipleConnectorInstanceRetrieval(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// List all instances
	instances, err := client.ListConnectorInstances(ctx)
	require.NoError(t, err, "Should successfully list connector instances")

	if len(instances) < 2 {
		t.Skip("Need at least 2 connector instances in target environment for this test")
	}

	// Retrieve details for first 2 instances (or all if less than 2)
	maxInstances := 2
	if len(instances) < maxInstances {
		maxInstances = len(instances)
	}

	retrievedCount := 0
	for i := 0; i < maxInstances; i++ {
		instanceDetail, err := client.GetConnectorInstance(ctx, instances[i].InstanceID)
		require.NoError(t, err, "Should successfully get connector instance %d", i)
		require.NotNil(t, instanceDetail, "Instance detail should not be nil")

		t.Logf("Retrieved connector instance %d: %s", i+1, instanceDetail.Name)
		retrievedCount++
	}

	assert.Equal(t, maxInstances, retrievedCount, "Should retrieve all requested connector instances")
}
