package pingone

import (
	"context"
	"fmt"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_application_resource is a child of pingone_resource, NOT
// pingone_application — the provider's own import format confirms this
// (<environment_id>/<resource_id>/<application_resource_id>). Enumerates
// CUSTOM resources (same filter as listResources/listCustomResourceIDs),
// then lists application resources per resource. ApplicationResourcesApi
// shares the discriminated-union embedded "resources" key with ResourcesApi,
// distinguishing *management.Resource from *management.ResourceApplicationResource.
func init() {
	registerResource("pingone_application_resource", resourceHandler{
		list: listApplicationResources,
		get:  getApplicationResource,
	})
}

// listApplicationResources implements list-then-scan: lists all CUSTOM
// resources, then lists application resources for each.
func listApplicationResources(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	customResourceIDs, err := listCustomResourceIDs(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("list application resources: %w", err)
	}

	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	for _, resourceID := range customResourceIDs {
		iterator := mgmt.ApplicationResourcesApi.ReadAllApplicationResources(ctx, c.environmentID.String(), resourceID).Execute()
		for cursor, err := range iterator {
			if err != nil {
				return nil, fmt.Errorf("list application resources for resource %s: %w", resourceID, err)
			}
			embedded, ok := cursor.EntityArray.GetEmbeddedOk()
			if !ok || embedded == nil {
				continue
			}
			resources, ok := embedded.GetResourcesOk()
			if !ok {
				continue
			}
			for _, inner := range resources {
				appRes, ok := inner.GetActualInstance().(*management.ResourceApplicationResource)
				if !ok || appRes == nil {
					continue
				}
				result = append(result, appRes)
			}
		}
	}
	return result, nil
}

// getApplicationResource retrieves a single application resource.
// resourceID is a composite "resourceID/applicationResourceID" string.
func getApplicationResource(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("get application resource: resourceID must be resourceID/applicationResourceID, got: %s", resourceID)
	}
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	appRes, _, err := mgmt.ApplicationResourcesApi.ReadOneApplicationResource(ctx, c.environmentID.String(), parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get application resource: %w", err)
	}
	return appRes, nil
}
