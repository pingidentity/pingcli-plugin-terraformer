package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_application_secret is a 1:1 singleton per pingone_application —
// the management SDK's ApplicationSecret struct has no Id field at all
// (Links, Environment, Secret, Previous only), so applicationSecretData
// projects a synthetic ID equal to the owning application's ID, matching
// applicationSecretData source_paths in application_secret.yaml.
func init() {
	registerResource("pingone_application_secret", resourceHandler{
		list: listApplicationSecrets,
		get:  getApplicationSecret,
	})
}

type applicationSecretData struct {
	ID            string
	EnvironmentID string
	ApplicationID string
	Secret        *string
	Previous      *applicationSecretPreviousData
}

type applicationSecretPreviousData struct {
	Secret    *string
	ExpiresAt string
	LastUsed  *string
}

func toApplicationSecretData(applicationID, environmentID string, v *management.ApplicationSecret) *applicationSecretData {
	data := &applicationSecretData{
		ID:            applicationID,
		EnvironmentID: environmentID,
		ApplicationID: applicationID,
		Secret:        v.Secret,
	}
	if prev, ok := v.GetPreviousOk(); ok && prev != nil {
		p := &applicationSecretPreviousData{
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

// listApplicationSecrets lists all applications in the environment, then
// reads each one's secret. Applications that don't support a client secret
// (e.g. external link, SAML) return an error from the API; these are skipped
// with a warning rather than failing the whole export, mirroring the
// per-application skip pattern in listSSOApplications.
func listApplicationSecrets(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	apps, err := listSSOApplications(ctx, c, envID)
	if err != nil {
		return nil, fmt.Errorf("list application secrets: %w", err)
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
		secret, _, err := mgmt.ApplicationSecretApi.ReadApplicationSecret(ctx, c.environmentID.String(), app.Id).Execute()
		if err != nil {
			c.AddWarning(fmt.Sprintf("skipping application secret for application %s: %v", app.Id, err))
			continue
		}
		result = append(result, toApplicationSecretData(app.Id, app.EnvironmentId, secret))
	}
	return result, nil
}

// getApplicationSecret retrieves a single application's secret by
// application ID.
func getApplicationSecret(ctx context.Context, c *Client, _ string, applicationID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	secret, _, err := mgmt.ApplicationSecretApi.ReadApplicationSecret(ctx, c.environmentID.String(), applicationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get application secret: %w", err)
	}
	return toApplicationSecretData(applicationID, c.environmentID.String(), secret), nil
}
