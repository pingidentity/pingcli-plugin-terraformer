package pingone

import (
	"context"
	"fmt"
	"strings"
)

// pingone_application_sign_on_policy_assignment is a child of
// pingone_application. ApplicationSignOnPolicyAssignmentsApi is scoped by
// applicationID, so listing requires enumerating applications first, then
// sign-on policy assignments per application.
func init() {
	registerResource("pingone_application_sign_on_policy_assignment", resourceHandler{
		list: listApplicationSignOnPolicyAssignments,
		get:  getApplicationSignOnPolicyAssignment,
	})
}

// listApplicationSignOnPolicyAssignments implements list-then-scan: lists
// all applications, then lists sign-on policy assignments for each.
func listApplicationSignOnPolicyAssignments(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	apps, err := listSSOApplications(ctx, c, envID)
	if err != nil {
		return nil, fmt.Errorf("list application sign-on policy assignments: %w", err)
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
		iterator := mgmt.ApplicationSignOnPolicyAssignmentsApi.ReadAllSignOnPolicyAssignments(ctx, c.environmentID.String(), app.Id).Execute()
		for cursor, err := range iterator {
			if err != nil {
				return nil, fmt.Errorf("list application sign-on policy assignments for application %s: %w", app.Id, err)
			}
			embedded, ok := cursor.EntityArray.GetEmbeddedOk()
			if !ok || embedded == nil {
				continue
			}
			assignments, ok := embedded.GetSignOnPolicyAssignmentsOk()
			if !ok {
				continue
			}
			for i := range assignments {
				result = append(result, &assignments[i])
			}
		}
	}
	return result, nil
}

// getApplicationSignOnPolicyAssignment retrieves a single sign-on policy
// assignment. resourceID is a composite "applicationID/assignmentID" string.
func getApplicationSignOnPolicyAssignment(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("get application sign-on policy assignment: resourceID must be applicationID/assignmentID, got: %s", resourceID)
	}
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	assignment, _, err := mgmt.ApplicationSignOnPolicyAssignmentsApi.ReadOneSignOnPolicyAssignment(ctx, c.environmentID.String(), parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get application sign-on policy assignment: %w", err)
	}
	return assignment, nil
}
