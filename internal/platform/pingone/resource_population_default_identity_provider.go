package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_population_default_identity_provider is a child of
// pingone_population. PopulationsApi.ReadOnePopulationDefaultIdp is a
// per-population singleton read (no list variant) — the SDK's
// PopulationDefaultIdp struct only carries {Id, Type}, where Id is the
// *identity provider's* ID, not a resource ID of its own. The handler
// enumerates all populations, then reads each one's default IdP, projecting
// the population ID onto a flat struct since it's the scoping parameter of
// the API call, not part of the response body.
func init() {
	registerResource("pingone_population_default_identity_provider", resourceHandler{
		list: listPopulationDefaultIdentityProviders,
		get:  getPopulationDefaultIdentityProvider,
	})
}

type populationDefaultIdentityProviderData struct {
	EnvironmentID      string
	PopulationID       string
	IdentityProviderID string
	Type               string
}

func toPopulationDefaultIdentityProviderData(populationID, environmentID string, v *management.PopulationDefaultIdp) *populationDefaultIdentityProviderData {
	data := &populationDefaultIdentityProviderData{
		EnvironmentID: environmentID,
		PopulationID:  populationID,
	}
	if id, ok := v.GetIdOk(); ok && id != nil {
		data.IdentityProviderID = *id
	}
	if t, ok := v.GetTypeOk(); ok && t != nil {
		data.Type = *t
	}
	return data
}

// listPopulationDefaultIdentityProviders implements list-then-scan: lists
// all populations, then reads each one's default identity provider.
func listPopulationDefaultIdentityProviders(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	populationIDs, err := listPopulationIDs(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("list population default identity providers: %w", err)
	}

	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	for _, populationID := range populationIDs {
		idp, _, err := mgmt.PopulationsApi.ReadOnePopulationDefaultIdp(ctx, c.environmentID.String(), populationID).Execute()
		if err != nil {
			c.AddWarning(fmt.Sprintf("skipping default identity provider for population %s: %v", populationID, err))
			continue
		}
		result = append(result, toPopulationDefaultIdentityProviderData(populationID, c.environmentID.String(), idp))
	}
	return result, nil
}

// getPopulationDefaultIdentityProvider retrieves a single population's
// default identity provider by population ID.
func getPopulationDefaultIdentityProvider(ctx context.Context, c *Client, _ string, populationID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	idp, _, err := mgmt.PopulationsApi.ReadOnePopulationDefaultIdp(ctx, c.environmentID.String(), populationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get population default identity provider: %w", err)
	}
	return toPopulationDefaultIdentityProviderData(populationID, c.environmentID.String(), idp), nil
}

// listPopulationIDs enumerates the IDs of all populations in the environment.
func listPopulationIDs(ctx context.Context, c *Client) ([]string, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
	iterator := mgmt.PopulationsApi.ReadAllPopulations(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list populations: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		populations, ok := embedded.GetPopulationsOk()
		if !ok {
			continue
		}
		for _, pop := range populations {
			ids = append(ids, pop.GetId())
		}
	}
	return ids, nil
}
