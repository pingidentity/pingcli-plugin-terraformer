package pingone

import (
	"context"
	"fmt"
)

func init() {
	registerResource("pingone_password_policy", resourceHandler{
		list: listPasswordPolicies,
		get:  getPasswordPolicy,
	})
}

// listPasswordPolicies lists all password policies in the environment using
// the management SDK's paginated iterator.
func listPasswordPolicies(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	iterator := mgmt.PasswordPoliciesApi.ReadAllPasswordPolicies(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list password policies: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		policies, ok := embedded.GetPasswordPoliciesOk()
		if !ok {
			continue
		}
		for i := range policies {
			result = append(result, &policies[i])
		}
	}
	return result, nil
}

// getPasswordPolicy retrieves a single password policy by ID via the
// management SDK.
func getPasswordPolicy(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	policy, _, err := mgmt.PasswordPoliciesApi.ReadOnePasswordPolicy(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get password policy: %w", err)
	}
	return policy, nil
}
