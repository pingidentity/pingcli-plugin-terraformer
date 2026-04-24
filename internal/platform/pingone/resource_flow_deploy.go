package pingone

import (
	"context"
	"fmt"
)

// flowDeployData is a projection struct that maps DaVinciFlowResponse fields
// into the nested structure required by flow_deploy.yaml source_paths.
// The Terraform resource needs deploy_trigger_values { deployed_version = N }
// but the SDK has CurrentVersion as a flat top-level *float32 field.
//
// Field names must match flow_deploy.yaml source_paths:
//
//	FlowID              -> source_path: FlowID
//	Name                -> source_path: Name
//	DeployTriggerValues -> source_path: DeployTriggerValues
type flowDeployData struct {
	FlowID              string
	Name                string
	DeployTriggerValues *deployTriggerValues
}

// deployTriggerValues holds the version that triggers a deployment.
// When CurrentVersion changes (flow is updated), it differs from the
// previously deployed version, causing Terraform to plan an update.
//
// Field names must match flow_deploy.yaml nested_attributes source_paths:
//
//	DeployedVersion -> source_path: DeployedVersion
type deployTriggerValues struct {
	DeployedVersion int
}

func init() {
	registerResource("pingone_davinci_flow_deploy", resourceHandler{
		list: listFlowDeploys,
		get:  getFlowDeploy,
	})
}

// toFlowDeployData projects a DaVinciFlowResponse into the nested struct
// matching flow_deploy.yaml source_paths.
func toFlowDeployData(flowID, name string, currentVersion *float32) *flowDeployData {
	data := &flowDeployData{
		FlowID: flowID,
		Name:   name,
	}
	if currentVersion != nil {
		data.DeployTriggerValues = &deployTriggerValues{
			DeployedVersion: int(*currentVersion),
		}
	}
	return data
}

// listFlowDeploys implements list-then-get: lists all flows, then calls get
// for each to retrieve full details and projects them into flowDeployData.
func listFlowDeploys(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	resp, _, err := c.apiClient.DaVinciFlowsApi.GetFlows(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list flow deploys: %w", err)
	}
	embedded := resp.GetEmbedded()
	flows := embedded.GetFlows()
	result := make([]interface{}, 0, len(flows))
	for _, flow := range flows {
		detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, flow.GetId()).Execute()
		if err != nil {
			return nil, fmt.Errorf("get flow deploy %s: %w", flow.GetId(), err)
		}
		result = append(result, toFlowDeployData(detail.GetId(), detail.GetName(), detail.CurrentVersion))
	}
	return result, nil
}

// getFlowDeploy retrieves a single flow and projects it into flowDeployData.
func getFlowDeploy(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get flow deploy: %w", err)
	}
	return toFlowDeployData(detail.GetId(), detail.GetName(), detail.CurrentVersion), nil
}
