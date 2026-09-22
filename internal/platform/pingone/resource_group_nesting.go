package pingone

import (
	"context"
	"fmt"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_group_nesting is a child of pingone_group. GroupsApi.ReadGroupNesting
// is scoped by groupID, so listing requires enumerating groups first, then
// listing nestings per group.
//
// The SDK's GroupNesting struct only carries {Id, Type} — Id IS the nested
// group's ID, with no separate nesting-relationship ID and no parent group
// ID field (the parent is only known from the scoping API parameter). This
// projects both onto a flat struct matching group_nesting.yaml source_paths.
func init() {
	registerResource("pingone_group_nesting", resourceHandler{
		list: listGroupNestings,
		get:  getGroupNesting,
	})
}

type groupNestingData struct {
	ID            string
	EnvironmentID string
	GroupID       string
	NestedGroupID string
	Type          string
}

func toGroupNestingData(groupID, environmentID string, v *management.GroupNesting) *groupNestingData {
	return &groupNestingData{
		ID:            v.GetId(),
		EnvironmentID: environmentID,
		GroupID:       groupID,
		NestedGroupID: v.GetId(),
		Type:          v.GetType(),
	}
}

// listGroupNestings implements list-then-scan: lists all groups, then lists
// nestings for each.
func listGroupNestings(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	groupIDs, err := listGroupIDs(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("list group nestings: %w", err)
	}

	var result []interface{}
	for _, groupID := range groupIDs {
		iterator := mgmt.GroupsApi.ReadGroupNesting(ctx, c.environmentID.String(), groupID).Execute()
		for cursor, err := range iterator {
			if err != nil {
				return nil, fmt.Errorf("list group nestings for group %s: %w", groupID, err)
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
				nestedID := groups[i].GetId()
				result = append(result, toGroupNestingData(groupID, c.environmentID.String(), management.NewGroupNesting(nestedID)))
			}
		}
	}
	return result, nil
}

// getGroupNesting retrieves a single group nesting. resourceID is a
// composite "groupID/nestedGroupID" string.
func getGroupNesting(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("get group nesting: resourceID must be groupID/nestedGroupID, got: %s", resourceID)
	}
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	nesting, _, err := mgmt.GroupsApi.ReadOneGroupNesting(ctx, c.environmentID.String(), parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get group nesting: %w", err)
	}
	return toGroupNestingData(parts[0], c.environmentID.String(), nesting), nil
}

// listGroupIDs enumerates the IDs of all groups in the environment.
func listGroupIDs(ctx context.Context, c *Client) ([]string, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
	iterator := mgmt.GroupsApi.ReadAllGroups(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list groups: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		groups, ok := embedded.GetGroupsOk()
		if !ok {
			continue
		}
		for _, grp := range groups {
			ids = append(ids, grp.GetId())
		}
	}
	return ids, nil
}
