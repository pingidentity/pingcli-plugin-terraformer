package davinci

import (
	"context"
	"fmt"
	"strings"
)

func init() {
	// API client dispatch.
	registerResource("pingone_davinci_application_flow_policy", resourceHandler{
		list: listFlowPolicies,
		get:  getFlowPolicy,
	})
}

// listFlowPolicies implements list-then-get: lists all flow policies across
// all applications, then calls get for each to retrieve full details including
// trigger configuration and flow distributions.
func listFlowPolicies(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	// First, get all applications in the environment.
	appsResp, _, err := c.apiClient.DaVinciApplicationsApi.GetDavinciApplications(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list applications for flow policies: %w", err)
	}
	embedded, ok := appsResp.GetEmbeddedOk()
	if !ok || embedded == nil {
		return []interface{}{}, nil
	}
	applications, ok := embedded.GetDavinciApplicationsOk()
	if !ok || applications == nil {
		return []interface{}{}, nil
	}

	// Collect all flow policies from all applications.
	var result []interface{}
	for _, app := range applications {
		appID := app.GetId()
		policiesResp, _, err := c.apiClient.DaVinciApplicationsApi.GetFlowPoliciesByDavinciApplicationId(ctx, c.environmentID, appID).Execute()
		if err != nil {
			// Skip applications that don't have flow policies or have errors.
			continue
		}
		polEmbedded, ok := policiesResp.GetEmbeddedOk()
		if !ok || polEmbedded == nil {
			continue
		}
		policies, ok := polEmbedded.GetFlowPoliciesOk()
		if !ok || policies == nil {
			continue
		}
		for _, pol := range policies {
			detail, _, err := c.apiClient.DaVinciApplicationsApi.GetFlowPolicyByIdUsingDavinciApplicationId(ctx, c.environmentID, appID, pol.GetId()).Execute()
			if err != nil {
				return nil, fmt.Errorf("get flow policy %s/%s: %w", appID, pol.GetId(), err)
			}
			result = append(result, detail)
		}
	}
	return result, nil
}

// getFlowPolicy parses a composite "applicationID/policyID" resource ID and
// returns the SDK DaVinciFlowPolicyResponse.
func getFlowPolicy(ctx context.Context, c *Client, _ string, compositeID string) (interface{}, error) {
	parts := strings.SplitN(compositeID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("flow policy resourceID must be applicationID/policyID, got: %s", compositeID)
	}
	detail, _, err := c.apiClient.DaVinciApplicationsApi.GetFlowPolicyByIdUsingDavinciApplicationId(ctx, c.environmentID, parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get flow policy: %w", err)
	}
	return detail, nil
}
