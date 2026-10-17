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
	"slices"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	pingonesdkv2 "github.com/patrickcping/pingone-go-sdk-v2/pingone"
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
	apiClient        *pingone.APIClient
	managementCfg    *pingonesdkv2.Config
	managementClient *management.APIClient
	environmentID    uuid.UUID
	warnings         []string
}

// New creates a DaVinci APIClient from a pre-built SDK client and environment ID.
func New(apiClient *pingone.APIClient, environmentID uuid.UUID) *Client {
	return &Client{apiClient: apiClient, environmentID: environmentID}
}

// NewWithManagementClient creates a Client from a pre-built DaVinci SDK
// client and a pre-built management SDK client. Used by tests that need to
// inject a fake management.APIClient without performing a real OAuth
// exchange.
func NewWithManagementClient(apiClient *pingone.APIClient, managementClient *management.APIClient, environmentID uuid.UUID) *Client {
	return &Client{apiClient: apiClient, managementClient: managementClient, environmentID: environmentID}
}

// management lazily builds and caches the management SDK API client. The
// management SDK performs an OAuth token exchange as soon as the client is
// built (unlike the DaVinci SDK, which defers auth to the first request), so
// construction is deferred until a management resource handler actually
// needs it.
func (c *Client) management(ctx context.Context) (*management.APIClient, error) {
	if c.managementClient != nil {
		return c.managementClient, nil
	}
	if c.managementCfg == nil {
		return nil, fmt.Errorf("management API client not configured")
	}
	client, err := c.managementCfg.ManagementAPIClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize management API client: %w", err)
	}
	c.managementClient = client
	return client, nil
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

// AddWarning records a non-fatal warning message for later retrieval,
// skipping exact duplicates. Several resource handlers independently
// enumerate the same underlying entities (e.g. every application-child
// handler calls listSSOApplications), so without dedup the same message
// (e.g. "skipping application <id>: ...") would otherwise be emitted once
// per handler that happens to touch it.
func (c *Client) AddWarning(msg string) {
	if slices.Contains(c.warnings, msg) {
		return
	}
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
	apiClient, err := pingone.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API client: %w", err)
	}

	regionCode := management.EnumRegionCode(region)
	managementCfg := &pingonesdkv2.Config{
		ClientID:      &clientID,
		ClientSecret:  &clientSecret,
		EnvironmentID: &workerEnvID,
		RegionCode:    &regionCode,
	}

	envUUID, err := uuid.Parse(exportEnvID)
	if err != nil {
		return nil, fmt.Errorf("invalid export environment ID format: %w", err)
	}

	return &Client{apiClient: apiClient, managementCfg: managementCfg, environmentID: envUUID}, nil
}

// ValidRegions returns the list of valid PingOne region codes.
func ValidRegions() []string {
	return []string{"NA", "EU", "AP", "CA", "AU", "SG"}
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
	case "AU":
		return "com.au"
	case "SG":
		return "sg"
	default:
		return "com"
	}
}
