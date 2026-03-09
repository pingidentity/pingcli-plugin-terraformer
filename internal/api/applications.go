package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pingidentity/pingone-go-client/pingone"
)

// ListApplications retrieves all DaVinci applications for an environment
func (c *Client) ListApplications(ctx context.Context, environmentID string) ([]pingone.DaVinciApplicationResponse, error) {
	if environmentID == "" {
		return nil, fmt.Errorf("environment ID is required")
	}

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID format: %w", err)
	}

	response, _, err := c.apiClient.DaVinciApplicationsApi.GetDavinciApplications(ctx, envUUID).Execute()
	if err != nil {
		return nil, fmt.Errorf("error fetching applications: %w", err)
	}

	embedded := response.Embedded
	applications, ok := embedded.GetDavinciApplicationsOk()
	if !ok || applications == nil {
		return []pingone.DaVinciApplicationResponse{}, nil
	}

	return applications, nil
}

// GetApplication retrieves a specific DaVinci application by ID
func (c *Client) GetApplication(ctx context.Context, environmentID, applicationID string) (*pingone.DaVinciApplicationResponse, error) {
	if environmentID == "" {
		return nil, fmt.Errorf("environment ID is required")
	}
	if applicationID == "" {
		return nil, fmt.Errorf("application ID is required")
	}

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID format: %w", err)
	}

	application, _, err := c.apiClient.DaVinciApplicationsApi.GetDavinciApplicationById(ctx, envUUID, applicationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("error fetching application %s: %w", applicationID, err)
	}

	return application, nil
}
