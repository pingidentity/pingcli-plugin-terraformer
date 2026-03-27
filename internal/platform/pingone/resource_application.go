package pingone

import (
	"context"
	"fmt"
)

func init() {
	// API client dispatch.
	registerResource("pingone_davinci_application", resourceHandler{
		list: listApplications,
		get:  getApplication,
	})
}

func listApplications(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	resp, _, err := c.apiClient.DaVinciApplicationsApi.GetDavinciApplications(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	embedded := resp.GetEmbedded()
	apps := embedded.GetDavinciApplications()
	result := make([]interface{}, len(apps))
	for i := range apps {
		result[i] = &apps[i]
	}
	return result, nil
}

func getApplication(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	app, _, err := c.apiClient.DaVinciApplicationsApi.GetDavinciApplicationById(ctx, c.environmentID, resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	return app, nil
}
