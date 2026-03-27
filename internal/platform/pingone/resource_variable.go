package pingone

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func init() {
	// API client dispatch.
	registerResource("pingone_davinci_variable", resourceHandler{
		list: listVariables,
		get:  getVariable,
	})
}

func listVariables(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	iterator := c.apiClient.DaVinciVariablesApi.GetVariables(ctx, c.environmentID).Execute()

	var result []interface{}
	for pageCursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list variables: %w", err)
		}
		embedded := pageCursor.Data.Embedded
		variables, ok := embedded.GetVariablesOk()
		if ok && variables != nil {
			for i := range variables {
				result = append(result, &variables[i])
			}
		}
	}
	return result, nil
}

func getVariable(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	varUUID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid variable ID: %w", err)
	}
	variable, _, err := c.apiClient.DaVinciVariablesApi.GetVariableById(ctx, c.environmentID, varUUID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get variable: %w", err)
	}
	return variable, nil
}
