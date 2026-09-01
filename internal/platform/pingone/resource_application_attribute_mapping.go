package pingone

import (
	"context"
	"fmt"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_application_attribute_mapping is a child of pingone_application.
// ApplicationAttributeMappingApi is scoped by applicationID, so listing
// requires enumerating applications first, then attribute mappings per
// application.
func init() {
	registerResource("pingone_application_attribute_mapping", resourceHandler{
		list: listApplicationAttributeMappings,
		get:  getApplicationAttributeMapping,
	})
}

// listApplicationAttributeMappings implements list-then-scan: lists all
// applications, then lists attribute mappings for each.
func listApplicationAttributeMappings(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	apps, err := listSSOApplications(ctx, c, envID)
	if err != nil {
		return nil, fmt.Errorf("list application attribute mappings: %w", err)
	}

	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	for _, item := range apps {
		app, ok := item.(*applicationData)
		if !ok || app == nil {
			continue
		}
		iterator := mgmt.ApplicationAttributeMappingApi.ReadAllApplicationAttributeMappings(ctx, c.environmentID.String(), app.Id).Execute()
		for cursor, err := range iterator {
			if err != nil {
				return nil, fmt.Errorf("list application attribute mappings for application %s: %w", app.Id, err)
			}
			embedded, ok := cursor.EntityArray.GetEmbeddedOk()
			if !ok || embedded == nil {
				continue
			}
			mappings, ok := embedded.GetAttributesOk()
			if !ok {
				continue
			}
			for i := range mappings {
				mapping, ok := mappings[i].GetActualInstance().(*management.ApplicationAttributeMapping)
				if !ok || mapping == nil {
					continue
				}
				result = append(result, mapping)
			}
		}
	}
	return result, nil
}

// getApplicationAttributeMapping retrieves a single attribute mapping.
// resourceID is a composite "applicationID/mappingID" string.
func getApplicationAttributeMapping(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("get application attribute mapping: resourceID must be applicationID/mappingID, got: %s", resourceID)
	}
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	mapping, _, err := mgmt.ApplicationAttributeMappingApi.ReadOneApplicationAttributeMapping(ctx, c.environmentID.String(), parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get application attribute mapping: %w", err)
	}
	return mapping, nil
}
