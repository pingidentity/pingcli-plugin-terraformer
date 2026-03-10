package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pingidentity/pingone-go-client/pingone"
)

// ListVariables retrieves all variables for an environment
func (c *Client) ListVariables(ctx context.Context, environmentID string) ([]pingone.DaVinciVariableResponse, error) {
	if environmentID == "" {
		return nil, fmt.Errorf("environment ID is required")
	}

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID format: %w", err)
	}

	iterator := c.apiClient.DaVinciVariablesApi.GetVariables(ctx, envUUID).Execute()

	var allVariables []pingone.DaVinciVariableResponse
	for pageCursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("error fetching variables page: %w", err)
		}

		embedded := pageCursor.Data.Embedded
		variables, ok := embedded.GetVariablesOk()
		if ok && variables != nil {
			allVariables = append(allVariables, variables...)
		}
	}

	return allVariables, nil
}

// GetVariable retrieves a specific variable by ID
func (c *Client) GetVariable(ctx context.Context, environmentID, variableID string) (*pingone.DaVinciVariableResponse, error) {
	if environmentID == "" {
		return nil, fmt.Errorf("environment ID is required")
	}
	if variableID == "" {
		return nil, fmt.Errorf("variable ID is required")
	}

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID format: %w", err)
	}

	varUUID, err := uuid.Parse(variableID)
	if err != nil {
		return nil, fmt.Errorf("invalid variable ID format: %w", err)
	}

	variable, _, err := c.apiClient.DaVinciVariablesApi.GetVariableById(ctx, envUUID, varUUID).Execute()
	if err != nil {
		return nil, fmt.Errorf("error fetching variable %s: %w", variableID, err)
	}

	return variable, nil
}
