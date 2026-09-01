package pingone

import (
	"context"
	"fmt"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
)

func init() {
	// API client dispatch.
	registerResource("pingone_group", resourceHandler{
		list: listGroups,
		get:  getGroup,
	})

	// Embedded reference: user_filter is a plain SCIM filter string (not a
	// jsonencode(...) JSON blob) that may embed a population UUID, e.g.
	// `population.id eq "<uuid>"`. reference_with_fallback mirrors the
	// DaVinci embedded-reference precedent (resource_flow.go) — resolve via
	// the graph when possible, otherwise emit a variable so the export never
	// fails outright over a filtered-out or cross-environment population.
	registerEmbeddedReferenceRule(core.EmbeddedReferenceRule{
		ResourceType:       "pingone_group",
		AttributePath:      "user_filter",
		TargetResourceType: "pingone_population",
		ReferenceField:     "id",
		PlainStringPattern: `population\.id eq "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})"`,
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "group_user_filter_population",
	})
}

// listGroups lists all groups in the environment using the management SDK's
// paginated iterator over ReadAllGroups.
func listGroups(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	iterator := mgmt.GroupsApi.ReadAllGroups(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list groups: %w", err)
		}
		if cursor.EntityArray == nil {
			continue
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		groups, ok := embedded.GetGroupsOk()
		if !ok {
			continue
		}
		for i := range groups {
			result = append(result, &groups[i])
		}
	}
	return result, nil
}

// getGroup retrieves a single group by ID using the management SDK.
func getGroup(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	group, _, err := mgmt.GroupsApi.ReadOneGroup(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	return group, nil
}
