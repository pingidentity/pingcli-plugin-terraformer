package pingone

import (
	"context"
	"fmt"
)

func init() {
	// API client dispatch.
	registerResource("pingone_population", resourceHandler{
		list: listPopulations,
		get:  getPopulation,
	})
}

// listPopulations lists all populations in the environment using the
// management SDK's paginated iterator. Unlike the DaVinci SDK's cursor
// pattern (pageCursor.Data.Embedded), the management SDK exposes the page's
// entity array embedded struct via cursor.EntityArray.GetEmbeddedOk().
func listPopulations(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
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
			result = append(result, &populations[i])
		}
	}
	return result, nil
}

// getPopulation retrieves a single population by ID via the management SDK.
func getPopulation(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	population, _, err := mgmt.PopulationsApi.ReadOnePopulation(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get population: %w", err)
	}
	return population, nil
}
