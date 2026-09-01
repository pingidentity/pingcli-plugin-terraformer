package pingone

import (
	"context"
	"fmt"
	"regexp"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_resource_scope_pingone_api manages scopes on the built-in
// PINGONE_API resource, which exists exactly once per environment. Shares
// findBuiltInResourceID / listBuiltInResourceScopes with
// pingone_resource_scope_openid (resource_resource_scope_openid.go).
//
// The PINGONE_API resource carries many built-in scopes (p1:read:device,
// p1:create:pairingKey, p1:verify:user, etc.), but the provider's own name
// validator — ported verbatim from terraform-provider-pingone's
// resource_resource_scope_pingone_api.go Schema() — only accepts
// p1:read:user / p1:update:user or those two suffixed
// (p1:read:user:{suffix} / p1:update:user:{suffix}). Every other built-in
// scope is structurally unmanageable via this resource type; exporting one
// produces HCL that fails terraform validate. Filter them out at list time.
// These built-ins are identical across every PingOne environment, so this is
// silent rather than a per-export warning — see RESOURCE_COVERAGE.md and
// README.md for the documented behavior.
var pingOneAPIScopeNamePattern = regexp.MustCompile(`^p1:(read|update):user(:[!#-\[\]-~]+)*$`)

func init() {
	registerResource("pingone_resource_scope_pingone_api", resourceHandler{
		list: listResourceScopesPingOneAPI,
		get:  getResourceScopePingOneAPI,
	})
}

func listResourceScopesPingOneAPI(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	all, err := listBuiltInResourceScopes(ctx, c, management.ENUMRESOURCETYPE_PINGONE_API)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	for _, item := range all {
		scope, ok := item.(*management.ResourceScope)
		if !ok || scope == nil {
			continue
		}
		if !pingOneAPIScopeNamePattern.MatchString(scope.GetName()) {
			continue
		}
		result = append(result, scope)
	}
	return result, nil
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
