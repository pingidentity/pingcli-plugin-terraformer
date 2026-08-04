package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_resource_scope_pingone_api manages scopes on the built-in
// PINGONE_API resource, which exists exactly once per environment. Shares
// findBuiltInResourceID / listBuiltInResourceScopes with
// pingone_resource_scope_openid (resource_resource_scope_openid.go).
func init() {
	registerResource("pingone_resource_scope_pingone_api", resourceHandler{
		list: listResourceScopesPingOneAPI,
		get:  getResourceScopePingOneAPI,
	})
}

func listResourceScopesPingOneAPI(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	return listBuiltInResourceScopes(ctx, c, management.ENUMRESOURCETYPE_PINGONE_API)
}

// getResourceScopePingOneAPI retrieves a single PingOne API resource scope by
// its scope ID, resolving the built-in PingOne API resource's ID first.
func getResourceScopePingOneAPI(ctx context.Context, c *Client, _ string, scopeID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := findBuiltInResourceID(ctx, c, management.ENUMRESOURCETYPE_PINGONE_API)
	if err != nil {
		return nil, fmt.Errorf("get resource scope: %w", err)
	}
	scope, _, err := mgmt.ResourceScopesApi.ReadOneResourceScope(ctx, c.environmentID.String(), resourceID, scopeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get resource scope: %w", err)
	}
	return scope, nil
}
