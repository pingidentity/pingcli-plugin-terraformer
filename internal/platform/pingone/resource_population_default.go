package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_population_default is a 1:1 singleton per environment — it manages
// whichever pingone_population has Default == true. There is no dedicated
// "read the default population" API, so the handler lists all populations
// and filters for the default one, reusing the same *management.Population
// struct as pingone_population.
func init() {
	registerResource("pingone_population_default", resourceHandler{
		list: listPopulationDefault,
		get:  getPopulationDefault,
	})
}

// listPopulationDefault lists all populations and returns the single one
// with Default == true, if any.
func listPopulationDefault(ctx context.Context, c *Client, envID string) ([]interface{}, error) {
	population, err := findDefaultPopulation(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("list default population: %w", err)
	}
	if population == nil {
		return nil, nil
	}
	return []interface{}{population}, nil
}

// getPopulationDefault ignores resourceID (this is a singleton keyed only by
// environment) and returns the default population.
func getPopulationDefault(ctx context.Context, c *Client, _ string, _ string) (interface{}, error) {
	population, err := findDefaultPopulation(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("get default population: %w", err)
	}
	if population == nil {
		return nil, fmt.Errorf("get default population: no default population found in environment")
	}
	return population, nil
}

// findDefaultPopulation scans all populations in the environment for the one
// with Default == true. Returns nil, nil if none is found.
func findDefaultPopulation(ctx context.Context, c *Client) (*management.Population, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

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
		for i := range populations {
			if isDefault, ok := populations[i].GetDefaultOk(); ok && isDefault != nil && *isDefault {
				return &populations[i], nil
			}
		}
	}
	return nil, nil
}
