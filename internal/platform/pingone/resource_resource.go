package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

func init() {
	registerResource("pingone_resource", resourceHandler{
		list: listResources,
		get:  getResource,
	})
}

// listResources lists all OAuth 2.0 custom resources in the environment.
// The API also returns the two built-in resources (OPENID_CONNECT and
// PINGONE_API) that exist in every environment by default — these are not
// manageable via the pingone_resource Terraform resource (only CUSTOM
// resources can be created), so they are filtered out here, mirroring the
// precedent set by listConnectorInstances skipping the built-in User Pool
// connector.
func listResources(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	iterator := mgmt.ResourcesApi.ReadAllResources(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list resources: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		resources, ok := embedded.GetResourcesOk()
		if !ok {
			continue
		}
		for i := range resources {
			inner := resources[i]
			res, ok := inner.GetActualInstance().(*management.Resource)
			if !ok || res == nil {
				// Skip the ResourceApplicationResource union variant — that
				// shape represents a different embedded resource type, not
				// a top-level pingone_resource.
				continue
			}
			if resType, ok := res.GetTypeOk(); !ok || resType == nil || *resType != management.ENUMRESOURCETYPE_CUSTOM {
				// Skip built-in resources (OPENID_CONNECT, PINGONE_API) —
				// not manageable via the pingone_resource Terraform resource.
				continue
			}
			result = append(result, res)
		}
	}
	return result, nil
}

// getResource retrieves a single OAuth 2.0 custom resource by ID.
func getResource(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	res, _, err := mgmt.ResourcesApi.ReadOneResource(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get resource: %w", err)
	}
	return res, nil
}
