package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_resource_scope_openid manages scopes on the built-in
// OPENID_CONNECT resource, which exists exactly once per environment. The
// handler resolves that resource's ID once via findBuiltInResourceID, then
// lists its scopes — mirroring pingone_resource_scope_pingone_api.
func init() {
	registerResource("pingone_resource_scope_openid", resourceHandler{
		list: listResourceScopesOpenID,
		get:  getResourceScopeOpenID,
	})
}

func listResourceScopesOpenID(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	return listBuiltInResourceScopes(ctx, c, management.ENUMRESOURCETYPE_OPENID_CONNECT)
}

// getResourceScopeOpenID retrieves a single OIDC resource scope by its scope
// ID, resolving the built-in OIDC resource's ID first.
func getResourceScopeOpenID(ctx context.Context, c *Client, _ string, scopeID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := findBuiltInResourceID(ctx, c, management.ENUMRESOURCETYPE_OPENID_CONNECT)
	if err != nil {
		return nil, fmt.Errorf("get resource scope: %w", err)
	}
	scope, _, err := mgmt.ResourceScopesApi.ReadOneResourceScope(ctx, c.environmentID.String(), resourceID, scopeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get resource scope: %w", err)
	}
	return scope, nil
}

// findBuiltInResourceID locates the single built-in resource of the given
// type (OPENID_CONNECT or PINGONE_API) in the environment and returns its ID.
func findBuiltInResourceID(ctx context.Context, c *Client, resourceType management.EnumResourceType) (string, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return "", err
	}

	iterator := mgmt.ResourcesApi.ReadAllResources(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return "", fmt.Errorf("list resources: %w", err)
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
			if resType, ok := res.GetTypeOk(); ok && resType != nil && *resType == resourceType {
				return res.GetId(), nil
			}
		}
	}
	return "", fmt.Errorf("built-in resource of type %s not found in environment", resourceType)
}

// listBuiltInResourceScopes lists all scopes on the single built-in resource
// of the given type, shared by pingone_resource_scope_openid and
// pingone_resource_scope_pingone_api.
func listBuiltInResourceScopes(ctx context.Context, c *Client, resourceType management.EnumResourceType) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	resourceID, err := findBuiltInResourceID(ctx, c, resourceType)
	if err != nil {
		return nil, fmt.Errorf("list resource scopes: %w", err)
	}

	var result []interface{}
	iterator := mgmt.ResourceScopesApi.ReadAllResourceScopes(ctx, c.environmentID.String(), resourceID).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list resource scopes: %w", err)
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
	return result, nil
}
