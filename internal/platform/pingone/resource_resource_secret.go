package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_resource_secret is a 1:1 singleton per CUSTOM pingone_resource —
// the management SDK's ResourceSecret struct has no Id field at all
// (Environment, Secret, Previous only), so resourceSecretData projects a
// synthetic ID equal to the owning resource's ID, mirroring
// application_secret.go.
func init() {
	registerResource("pingone_resource_secret", resourceHandler{
		list: listResourceSecrets,
		get:  getResourceSecret,
	})
}

type resourceSecretData struct {
	ID            string
	EnvironmentID string
	ResourceID    string
	Secret        *string
	Previous      *resourceSecretPreviousData
}

type resourceSecretPreviousData struct {
	Secret    *string
	ExpiresAt string
	LastUsed  *string
}

func toResourceSecretData(resourceID, environmentID string, v *management.ResourceSecret) *resourceSecretData {
	data := &resourceSecretData{
		ID:            resourceID,
		EnvironmentID: environmentID,
		ResourceID:    resourceID,
		Secret:        v.Secret,
	}
	if prev, ok := v.GetPreviousOk(); ok && prev != nil {
		p := &resourceSecretPreviousData{
			Secret:    prev.Secret,
			ExpiresAt: prev.GetExpiresAt().Format("2006-01-02T15:04:05Z07:00"),
		}
		if lastUsed, ok := prev.GetLastUsedOk(); ok && lastUsed != nil {
			s := lastUsed.Format("2006-01-02T15:04:05Z07:00")
			p.LastUsed = &s
		}
		data.Previous = p
	}
	return data
}

// listResourceSecrets lists all CUSTOM resources in the environment, then
// reads each one's secret. Resources with no client secret configured
// return an error from the API; these are skipped with a warning rather than
// failing the whole export, mirroring listApplicationSecrets.
func listResourceSecrets(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	customResourceIDs, err := listCustomResourceIDs(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("list resource secrets: %w", err)
	}

	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	for _, resourceID := range customResourceIDs {
		secret, _, err := mgmt.ResourceClientSecretApi.ReadResourceSecret(ctx, c.environmentID.String(), resourceID).Execute()
		if err != nil {
			c.AddWarning(fmt.Sprintf("skipping resource secret for resource %s: %v", resourceID, err))
			continue
		}
		result = append(result, toResourceSecretData(resourceID, c.environmentID.String(), secret))
	}
	return result, nil
}

// getResourceSecret retrieves a single resource's secret by resource ID.
func getResourceSecret(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	secret, _, err := mgmt.ResourceClientSecretApi.ReadResourceSecret(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get resource secret: %w", err)
	}
	return toResourceSecretData(resourceID, c.environmentID.String(), secret), nil
}
