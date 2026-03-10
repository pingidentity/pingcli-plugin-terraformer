package davinci

import (
	"context"
	"fmt"
)

func init() {
	registerResource("pingone_davinci_flow_enable", resourceHandler{
		list: listFlowEnables,
		get:  getFlowEnable,
	})
}

// listFlowEnables implements list-then-get: lists all flows, then calls get
// for each to retrieve full details including enabled status.
func listFlowEnables(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	resp, _, err := c.apiClient.DaVinciFlowsApi.GetFlows(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list flow enables: %w", err)
	}
	embedded := resp.GetEmbedded()
	flows := embedded.GetFlows()
	result := make([]interface{}, 0, len(flows))
	for _, flow := range flows {
		detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, flow.GetId()).Execute()
		if err != nil {
			return nil, fmt.Errorf("get flow enable %s: %w", flow.GetId(), err)
		}
		result = append(result, detail)
	}
	return result, nil
}

// getFlowEnable retrieves a single flow. The processor extracts Id and
// Enabled from the returned DaVinciFlowResponse.
func getFlowEnable(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get flow enable: %w", err)
	}
	return detail, nil
}
