package davinci

import (
	"context"
	"fmt"
)

func init() {
	// API client dispatch.
	registerResource("pingone_davinci_flow", resourceHandler{
		list: listFlows,
		get:  getFlow,
	})
}

// listFlows implements list-then-get: lists all flows to collect IDs,
// then calls get for each to retrieve full details including graph data,
// settings, input schema, etc. (which the list endpoint may omit).
func listFlows(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	resp, _, err := c.apiClient.DaVinciFlowsApi.GetFlows(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list flows: %w", err)
	}
	embedded := resp.GetEmbedded()
	flows := embedded.GetFlows()
	result := make([]interface{}, 0, len(flows))
	for _, flow := range flows {
		detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, flow.GetId()).Execute()
		if err != nil {
			return nil, fmt.Errorf("get flow %s: %w", flow.GetId(), err)
		}
		result = append(result, detail)
	}
	return result, nil
}

func getFlow(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get flow: %w", err)
	}
	return detail, nil
}
