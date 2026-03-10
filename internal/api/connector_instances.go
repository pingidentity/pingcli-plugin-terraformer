package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ConnectorInstanceSummary represents a summary of a DaVinci connector instance from the API
type ConnectorInstanceSummary struct {
	InstanceID  string
	Name        string
	ConnectorID string
}

// ConnectorInstanceDetail represents detailed connector instance data including properties
type ConnectorInstanceDetail struct {
	InstanceID  string
	Name        string
	ConnectorID string
	Properties  map[string]interface{} // The connector configuration properties
}

// ListConnectorInstances retrieves all connector instances from the environment using the SDK
func (c *Client) ListConnectorInstances(ctx context.Context) ([]ConnectorInstanceSummary, error) {
	envID, err := uuid.Parse(c.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	resp, _, err := c.apiClient.DaVinciConnectorsApi.GetConnectorInstances(ctx, envID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list connector instances: %w", err)
	}

	// Extract connector instances from embedded response
	embedded, ok := resp.GetEmbeddedOk()
	if !ok || embedded == nil {
		return []ConnectorInstanceSummary{}, nil
	}

	instancesResp, ok := embedded.GetConnectorInstancesOk()
	if !ok || instancesResp == nil {
		return []ConnectorInstanceSummary{}, nil
	}

	// Convert to summary structure
	instances := make([]ConnectorInstanceSummary, 0, len(instancesResp))
	for _, instance := range instancesResp {
		summary := ConnectorInstanceSummary{
			InstanceID: instance.GetId(),
			Name:       instance.GetName(),
		}

		// Extract connector ID from relationship
		if connector, ok := instance.GetConnectorOk(); ok && connector != nil {
			summary.ConnectorID = connector.GetId()
		}

		instances = append(instances, summary)
	}

	return instances, nil
}

// GetConnectorInstance retrieves detailed connector instance data including properties using the SDK
func (c *Client) GetConnectorInstance(ctx context.Context, instanceID string) (*ConnectorInstanceDetail, error) {
	envID, err := uuid.Parse(c.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	// Validate instance ID is not empty
	// Note: Some connector instances use non-UUID identifiers (e.g., "defaultUserPool")
	if instanceID == "" {
		return nil, fmt.Errorf("connector instance ID cannot be empty")
	}

	resp, _, err := c.apiClient.DaVinciConnectorsApi.GetConnectorInstanceById(ctx, envID, instanceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get connector instance: %w", err)
	}

	// Build detail structure
	detail := &ConnectorInstanceDetail{
		InstanceID: resp.GetId(),
		Name:       resp.GetName(),
	}

	// Extract connector ID
	if connector, ok := resp.GetConnectorOk(); ok && connector != nil {
		detail.ConnectorID = connector.GetId()
	}

	// Extract properties
	if properties, ok := resp.GetPropertiesOk(); ok && properties != nil {
		detail.Properties = properties
	}

	return detail, nil
}
