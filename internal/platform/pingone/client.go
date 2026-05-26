// Package pingone provides the unified platform package for PingOne.
// It implements the clients.APIClient interface and registers custom handlers
// and transforms for all PingOne resource types.
//
// Adding a new resource requires only a single new resource_*.go file whose
// init() calls registerResource() and optionally registerHandler() /
// registerTransform(). No other files need editing.
package pingone

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pingidentity/pingone-go-client/config"
	"github.com/pingidentity/pingone-go-client/oauth2"
	"github.com/pingidentity/pingone-go-client/pingone"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/clients"
)

// Compile-time interface check.
var _ clients.APIClient = (*Client)(nil)

// Client wraps the PingOne SDK APIClient to satisfy the clients.APIClient
// interface for the PingOne DaVinci service.
//
// Resource-specific list/get logic and custom handlers live in resource_*.go
// files. Each file registers everything for its resource via init().
type Client struct {
	apiClient     *pingone.APIClient
	environmentID uuid.UUID
	warnings      []string
}

// New creates a DaVinci APIClient from a pre-built SDK client and environment ID.
func New(apiClient *pingone.APIClient, environmentID uuid.UUID) *Client {
	return &Client{apiClient: apiClient, environmentID: environmentID}
}

// Platform returns the platform identifier.
func (c *Client) Platform() string { return "pingone" }

// ListResources retrieves all resources of the given type from the environment.
func (c *Client) ListResources(ctx context.Context, resourceType string, envID string) ([]interface{}, error) {
	h, ok := resourceHandlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	return h.list(ctx, c, envID)
}

// GetResource retrieves a single resource by type and ID.
func (c *Client) GetResource(ctx context.Context, resourceType string, envID string, resourceID string) (interface{}, error) {
	h, ok := resourceHandlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	return h.get(ctx, c, envID, resourceID)
}

// AddWarning records a non-fatal warning message for later retrieval.
func (c *Client) AddWarning(msg string) {
	c.warnings = append(c.warnings, msg)
}

// Warnings returns all warnings collected during resource operations.
func (c *Client) Warnings() []string {
	return c.warnings
}

// NewFromCredentials creates a fully initialized Client from OAuth credentials.
// workerEnvID is the environment where the OAuth client lives (used for token acquisition).
// exportEnvID is the target environment whose resources will be exported.
func NewFromCredentials(ctx context.Context, workerEnvID, exportEnvID, region, clientID, clientSecret string) (*Client, error) {
	if workerEnvID == "" {
		return nil, fmt.Errorf("auth environment ID is required")
	}
	if exportEnvID == "" {
		return nil, fmt.Errorf("target environment ID is required")
	}
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}
	if !IsValidRegion(region) {
		return nil, fmt.Errorf("invalid region: %s (valid regions: %v)", region, ValidRegions())
	}
	if clientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	serviceCfg := config.NewConfiguration().
		WithEnvironmentID(workerEnvID).
		WithTopLevelDomain(config.TopLevelDomain(getRegionDomain(region))).
		WithClientID(clientID).
		WithClientSecret(clientSecret).
		WithGrantType(oauth2.GrantTypeClientCredentials).
		WithStorageType(config.StorageTypeNone)

	cfg := pingone.NewConfiguration(serviceCfg)
	apiClient, err := pingone.NewAPIClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API client: %w", err)
	}

	envUUID, err := uuid.Parse(exportEnvID)
	if err != nil {
		return nil, fmt.Errorf("invalid export environment ID format: %w", err)
	}

	return &Client{apiClient: apiClient, environmentID: envUUID}, nil
}

// ValidRegions returns the list of valid PingOne region codes.
func ValidRegions() []string {
	return []string{"NA", "EU", "AP", "CA"}
}

// IsValidRegion reports whether the given region code is valid.
func IsValidRegion(region string) bool {
	for _, r := range ValidRegions() {
		if r == region {
			return true
		}
	}
	return false
}

// getRegionDomain returns the domain suffix for a given region code.
func getRegionDomain(region string) string {
	switch region {
	case "NA":
		return "com"
	case "EU":
		return "eu"
	case "AP":
		return "asia"
	case "CA":
		return "ca"
	default:
		return "com"
	}
}
