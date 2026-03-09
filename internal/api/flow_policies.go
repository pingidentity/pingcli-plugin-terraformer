package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pingidentity/pingone-go-client/pingone"
)

// FlowPolicySummary represents a summary of a DaVinci flow policy
type FlowPolicySummary struct {
	PolicyID      string
	Name          string
	Status        string
	ApplicationID string
}

// FlowPolicyDetail represents detailed flow policy data
type FlowPolicyDetail struct {
	PolicyID      string
	Name          string
	Status        string
	ApplicationID string
	// Store the raw SDK response for conversion
	RawResponse pingone.DaVinciFlowPolicyResponse
}

// ListFlowPolicies retrieves all flow policies from all applications in the environment
func (c *Client) ListFlowPolicies(ctx context.Context) ([]FlowPolicySummary, error) {
	envID, err := uuid.Parse(c.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	// First, get all applications in the environment
	appsResp, _, err := c.apiClient.DaVinciApplicationsApi.GetDavinciApplications(ctx, envID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	// Extract applications from embedded response
	embedded, ok := appsResp.GetEmbeddedOk()
	if !ok || embedded == nil {
		return []FlowPolicySummary{}, nil
	}

	applications, ok := embedded.GetDavinciApplicationsOk()
	if !ok || applications == nil {
		return []FlowPolicySummary{}, nil
	}

	// Collect all flow policies from all applications
	var allPolicies []FlowPolicySummary

	for _, app := range applications {
		appID := app.GetId()

		// Get flow policies for this application
		policiesResp, _, err := c.apiClient.DaVinciApplicationsApi.GetFlowPoliciesByDavinciApplicationId(ctx, envID, appID).Execute()
		if err != nil {
			// Skip applications that don't have flow policies or have errors
			continue
		}

		// Extract flow policies from embedded response
		policiesEmbedded, ok := policiesResp.GetEmbeddedOk()
		if !ok || policiesEmbedded == nil {
			continue
		}

		policies, ok := policiesEmbedded.GetFlowPoliciesOk()
		if !ok || policies == nil {
			continue
		}

		// Add policies to the collection
		for _, policy := range policies {
			summary := FlowPolicySummary{
				PolicyID:      policy.GetId(),
				Name:          policy.GetName(),
				Status:        string(policy.GetStatus()),
				ApplicationID: appID,
			}
			allPolicies = append(allPolicies, summary)
		}
	}

	return allPolicies, nil
}

// GetFlowPolicy retrieves detailed flow policy data including distributions and triggers
func (c *Client) GetFlowPolicy(ctx context.Context, applicationID, policyID string) (*FlowPolicyDetail, error) {
	envID, err := uuid.Parse(c.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	// Validate inputs
	if applicationID == "" {
		return nil, fmt.Errorf("application ID cannot be empty")
	}

	if policyID == "" {
		return nil, fmt.Errorf("flow policy ID cannot be empty")
	}

	// Get flow policy details
	resp, _, err := c.apiClient.DaVinciApplicationsApi.GetFlowPolicyByIdUsingDavinciApplicationId(ctx, envID, applicationID, policyID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get flow policy: %w", err)
	}

	// Build detail structure
	detail := &FlowPolicyDetail{
		PolicyID:      resp.GetId(),
		Name:          resp.GetName(),
		Status:        string(resp.GetStatus()),
		ApplicationID: applicationID,
		RawResponse:   *resp,
	}

	return detail, nil
}
