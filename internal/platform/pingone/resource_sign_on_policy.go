package pingone

import (
	"context"
	"fmt"
)

func init() {
	registerResource("pingone_sign_on_policy", resourceHandler{
		list: listSignOnPolicies,
		get:  getSignOnPolicy,
	})
}

// listSignOnPolicies lists all sign-on policies in the target environment.
func listSignOnPolicies(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	iterator := mgmt.SignOnPoliciesApi.ReadAllSignOnPolicies(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list sign-on policies: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		policies, ok := embedded.GetSignOnPoliciesOk()
		if !ok {
			continue
		}
		for i := range policies {
			result = append(result, &policies[i])
		}
	}
	return result, nil
}

// getSignOnPolicy retrieves a single sign-on policy by ID.
func getSignOnPolicy(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	policy, _, err := mgmt.SignOnPoliciesApi.ReadOneSignOnPolicy(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get sign-on policy: %w", err)
	}
	return policy, nil
}
