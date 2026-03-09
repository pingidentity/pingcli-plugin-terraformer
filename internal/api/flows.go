package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// FlowSummary represents a summary of a DaVinci flow from the API
type FlowSummary struct {
	FlowID      string
	Name        string
	Description string
	// Add other relevant fields as needed
}

// FlowDetail represents detailed flow data including graph structure
type FlowDetail struct {
	FlowID              string
	Name                string
	Description         string
	GraphData           map[string]interface{} // The full flow graph structure
	Settings            map[string]interface{} // Flow settings
	Color               string                 // Flow color (API field)
	InputSchema         []interface{}          // Top-level input schema array
	InputSchemaCompiled map[string]interface{} // Compiled input schema
	Trigger             map[string]interface{} // Flow trigger configuration
	// Provider-managed fields useful for auxiliary resources
	Enabled           bool        // Flow enabled status (API responses)
	PublishedVersion  *int        // Published version number (API responses)
	// Add other relevant fields as needed
}

// ListFlows retrieves all flows from the environment
//
// WORKAROUND: Uses raw HTTP request to bypass SDK's strict validation of optional fields.
// The SDK requires the Version field in flow responses, but the API returns flows where
// this field may be absent. This causes unmarshaling to fail.
//
// TODO: Revert to SDK's GetFlows() once the Version field is fixed in the SDK.
// See WORKAROUND_RAW_HTTP.md for detailed reversion instructions.
//
// Related SDK Issue: Version field in DaVinciFlowResponse lacks omitempty tag.
func (c *Client) ListFlows(ctx context.Context) ([]FlowSummary, error) {
	envID, err := uuid.Parse(c.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	// Build authenticated HTTP client from SDK configuration
	base := http.DefaultClient
	httpClient, err := c.serviceCfg.Client(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("failed to build authenticated HTTP client: %w", err)
	}

	// Make raw HTTP request
	// Use the correct path structure matching the SDK
	baseURL := fmt.Sprintf("https://api.pingone.%s/v1/environments/%s/flows",
		getRegionDomain(c.Region), envID.String())

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", httpResp.StatusCode, string(body))
	}

	// Parse response as raw JSON
	var rawResponse map[string]interface{}
	if err := json.NewDecoder(httpResp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract flows from _embedded structure
	summaries := []FlowSummary{}

	if embedded, ok := rawResponse["_embedded"].(map[string]interface{}); ok {
		if flowsData, ok := embedded["flows"].([]interface{}); ok {
			for _, flowItem := range flowsData {
				if flowMap, ok := flowItem.(map[string]interface{}); ok {
					summary := FlowSummary{}

					if id, ok := flowMap["id"].(string); ok {
						summary.FlowID = id
					}
					if name, ok := flowMap["name"].(string); ok {
						summary.Name = name
					}
					if desc, ok := flowMap["description"].(string); ok {
						summary.Description = desc
					}

					summaries = append(summaries, summary)
				}
			}
		}
	}

	return summaries, nil
}

// GetFlow retrieves detailed flow data including graph structure
//
// WORKAROUND: Uses raw HTTP request to bypass SDK's strict validation of optional fields.
// The SDK requires the Position field in flow graph nodes/edges, but the API returns flows
// where these fields may be absent. This causes unmarshaling to fail.
//
// TODO: Revert to SDK's GetFlowById() once the Position field is fixed in the SDK.
// See WORKAROUND_RAW_HTTP.md for detailed reversion instructions.
//
// Related SDK Issue: Position field in DaVinciFlowGraphDataResponseElementsNode and
// DaVinciFlowGraphDataResponseElementsEdge lacks omitempty tag and is not a pointer.
func (c *Client) GetFlow(ctx context.Context, flowID string) (*FlowDetail, error) {
	envID, err := uuid.Parse(c.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	// Build authenticated HTTP client from SDK configuration
	base := http.DefaultClient
	httpClient, err := c.serviceCfg.Client(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("failed to build authenticated HTTP client: %w", err)
	}

	// Make raw HTTP request
	// Use the correct path structure matching the SDK
	baseURL := fmt.Sprintf("https://api.pingone.%s/v1/environments/%s/flows/%s",
		getRegionDomain(c.Region), envID.String(), flowID)

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", httpResp.StatusCode, string(body))
	}

	// Parse response as raw JSON
	var rawResponse map[string]interface{}
	if err := json.NewDecoder(httpResp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract flow details
	detail := &FlowDetail{
		FlowID: flowID,
	}

	if name, ok := rawResponse["name"].(string); ok {
		detail.Name = name
	}

	if desc, ok := rawResponse["description"].(string); ok {
		detail.Description = desc
	}

	if graphData, ok := rawResponse["graphData"].(map[string]interface{}); ok {
		detail.GraphData = graphData
	}

	if settings, ok := rawResponse["settings"].(map[string]interface{}); ok {
		detail.Settings = settings
	}

	if color, ok := rawResponse["color"].(string); ok {
		detail.Color = color
	}

	// Top-level inputSchema array
	if is, ok := rawResponse["inputSchema"].([]interface{}); ok {
		detail.InputSchema = is
	}

	if isc, ok := rawResponse["inputSchemaCompiled"].(map[string]interface{}); ok {
		detail.InputSchemaCompiled = isc
	}

	// Trigger object
	if trg, ok := rawResponse["trigger"].(map[string]interface{}); ok {
		detail.Trigger = trg
	}

	// Enabled flag (API field)
	if en, ok := rawResponse["enabled"].(bool); ok {
		detail.Enabled = en
	}

	// Published version (API field). JSON numbers decode as float64; coerce to int.
	if pv, ok := rawResponse["publishedVersion"].(float64); ok {
		iv := int(pv)
		detail.PublishedVersion = &iv
	}

	return detail, nil
}

// stringValue safely extracts string value from pointer
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
