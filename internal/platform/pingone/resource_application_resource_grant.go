package pingone

import (
	"context"
	"fmt"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_application_resource_grant is a child of pingone_application.
// ApplicationResourceGrantsApi is scoped by applicationID, so listing
// requires enumerating applications first, then grants per application.
//
// The SDK's ApplicationResourceGrant.Resource only carries {Id} — no
// resource_type discriminator — but the provider schema requires
// resource_type (CUSTOM/OPENID_CONNECT/PINGONE_API) and, only for CUSTOM,
// custom_resource_id. Each grant requires a supplemental
// ResourcesApi.ReadOneResource lookup to resolve the resource's Type.
func init() {
	registerResource("pingone_application_resource_grant", resourceHandler{
		list: listApplicationResourceGrants,
		get:  getApplicationResourceGrant,
	})
}

// applicationResourceGrantData projects the SDK's ApplicationResourceGrant
// plus a supplemental resource-type lookup into a flat struct matching
// application_resource_grant.yaml source_paths.
type applicationResourceGrantData struct {
	ID               string
	EnvironmentID    string
	ApplicationID    string
	ResourceType     string
	CustomResourceID string
	ResourceID       string
	Scopes           []string
}

func toApplicationResourceGrantData(applicationID, environmentID string, v *management.ApplicationResourceGrant, resourceType management.EnumResourceType) *applicationResourceGrantData {
	scopes := make([]string, 0, len(v.GetScopes()))
	for _, s := range v.GetScopes() {
		scopes = append(scopes, s.GetId())
	}

	data := &applicationResourceGrantData{
		ID:            v.GetId(),
		EnvironmentID: environmentID,
		ApplicationID: applicationID,
		ResourceType:  string(resourceType),
		ResourceID:    v.Resource.Id,
		Scopes:        scopes,
	}
	if resourceType == management.ENUMRESOURCETYPE_CUSTOM {
		data.CustomResourceID = v.Resource.Id
	}
	return data
}

// listApplicationResourceGrants implements list-then-scan: lists all
// applications, then lists resource grants for each, resolving each grant's
// resource type via a per-resource cache to avoid redundant lookups.
func listApplicationResourceGrants(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	apps, err := listSSOApplications(ctx, c, envID)
	if err != nil {
		return nil, fmt.Errorf("list application resource grants: %w", err)
	}

	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	resourceTypeCache := make(map[string]management.EnumResourceType)

	var result []interface{}
	for _, item := range apps {
		app, ok := item.(*applicationData)
		if !ok || app == nil {
			continue
		}
		iterator := mgmt.ApplicationResourceGrantsApi.ReadAllApplicationGrants(ctx, c.environmentID.String(), app.Id).Execute()
		for cursor, err := range iterator {
			if err != nil {
				return nil, fmt.Errorf("list application resource grants for application %s: %w", app.Id, err)
			}
			embedded, ok := cursor.EntityArray.GetEmbeddedOk()
			if !ok || embedded == nil {
				continue
			}
			grants, ok := embedded.GetGrantsOk()
			if !ok {
				continue
			}
			for i := range grants {
				grant := &grants[i]
				resourceID := grant.Resource.Id
				resourceType, err := resolveResourceType(ctx, c, mgmt, resourceTypeCache, resourceID)
				if err != nil {
					c.AddWarning(fmt.Sprintf("skipping application resource grant %s: %v", grant.GetId(), err))
					continue
				}
				result = append(result, toApplicationResourceGrantData(app.Id, app.EnvironmentId, grant, resourceType))
			}
		}
	}
	return result, nil
}

// resolveResourceType looks up a resource's Type, caching by resource ID to
// avoid redundant API calls across grants sharing the same resource (e.g.
// the built-in OPENID_CONNECT/PINGONE_API resources, referenced by nearly
// every grant).
func resolveResourceType(ctx context.Context, c *Client, mgmt *management.APIClient, cache map[string]management.EnumResourceType, resourceID string) (management.EnumResourceType, error) {
	if cached, ok := cache[resourceID]; ok {
		return cached, nil
	}
	res, _, err := mgmt.ResourcesApi.ReadOneResource(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return "", fmt.Errorf("resolve resource type for resource %s: %w", resourceID, err)
	}
	resType, ok := res.GetTypeOk()
	if !ok || resType == nil {
		return "", fmt.Errorf("resource %s has no type", resourceID)
	}
	cache[resourceID] = *resType
	return *resType, nil
}

// getApplicationResourceGrant retrieves a single resource grant.
// resourceID is a composite "applicationID/grantID" string.
func getApplicationResourceGrant(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("get application resource grant: resourceID must be applicationID/grantID, got: %s", resourceID)
	}
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	grant, _, err := mgmt.ApplicationResourceGrantsApi.ReadOneApplicationGrant(ctx, c.environmentID.String(), parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get application resource grant: %w", err)
	}
	resourceType, err := resolveResourceType(ctx, c, mgmt, make(map[string]management.EnumResourceType), grant.Resource.Id)
	if err != nil {
		return nil, fmt.Errorf("get application resource grant: %w", err)
	}
	return toApplicationResourceGrantData(parts[0], c.environmentID.String(), grant, resourceType), nil
}
