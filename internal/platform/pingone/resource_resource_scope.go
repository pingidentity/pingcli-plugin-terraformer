package pingone

import (
	"context"
	"fmt"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_resource_scope manages scopes on CUSTOM resources only — the
// provider docs state this resource "cannot manage PingOne API or OpenID
// Connect scopes" (those are pingone_resource_scope_pingone_api /
// pingone_resource_scope_openid, handled by their own resource_*.go files).
// ResourceScopesApi is scoped by resourceID, so listing requires first
// enumerating CUSTOM resources (mirroring the filtering in listResources)
// then listing scopes per resource.
func init() {
	registerResource("pingone_resource_scope", resourceHandler{
		list: listResourceScopes,
		get:  getResourceScope,
	})
}

// listResourceScopes implements list-then-scan: lists all CUSTOM resources,
// then lists scopes for each.
func listResourceScopes(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	customResourceIDs, err := listCustomResourceIDs(ctx, c)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	for _, resourceID := range customResourceIDs {
		iterator := mgmt.ResourceScopesApi.ReadAllResourceScopes(ctx, c.environmentID.String(), resourceID).Execute()
		for cursor, err := range iterator {
			if err != nil {
				return nil, fmt.Errorf("list resource scopes for resource %s: %w", resourceID, err)
			}
			embedded, ok := cursor.EntityArray.GetEmbeddedOk()
			if !ok || embedded == nil {
				continue
			}
			scopes, ok := embedded.GetScopesOk()
			if !ok {
				continue
			}
			for i := range scopes {
				result = append(result, &scopes[i])
			}
		}
	}
	return result, nil
}

// getResourceScope retrieves a single resource scope. resourceID is a
// composite "resourceID/scopeID" string.
func getResourceScope(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("get resource scope: resourceID must be resourceID/scopeID, got: %s", resourceID)
	}
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	scope, _, err := mgmt.ResourceScopesApi.ReadOneResourceScope(ctx, c.environmentID.String(), parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get resource scope: %w", err)
	}
	return scope, nil
}

// listCustomResourceIDs enumerates the IDs of all CUSTOM resources in the
// environment, filtering out the built-in OPENID_CONNECT and PINGONE_API
// resources — the same filter listResources (resource_resource.go) applies.
func listCustomResourceIDs(ctx context.Context, c *Client) ([]string, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
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
		for _, inner := range resources {
			res, ok := inner.GetActualInstance().(*management.Resource)
			if !ok || res == nil {
				continue
			}
			if resType, ok := res.GetTypeOk(); !ok || resType == nil || *resType != management.ENUMRESOURCETYPE_CUSTOM {
				continue
			}
			ids = append(ids, res.GetId())
		}
	}
	return ids, nil
}
