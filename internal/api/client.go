package api

import (
	"context"
	"fmt"

	"github.com/pingidentity/pingone-go-client/config"
	"github.com/pingidentity/pingone-go-client/oauth2"
	"github.com/pingidentity/pingone-go-client/pingone"
)

// Client wraps the PingOne API client for DaVinci operations
type Client struct {
	apiClient         *pingone.APIClient
	serviceCfg        *config.Configuration // WORKAROUND: Used for raw HTTP token access in GetFlow()
	AuthEnvironmentID string                // Environment where OAuth client exists
	EnvironmentID     string                // Target environment for DaVinci operations
	Region            string
}

// NewClient creates a new API client for DaVinci operations with OAuth authentication
// authEnvironmentID: Environment where the OAuth client exists
// targetEnvironmentID: Environment to perform DaVinci operations on
func NewClient(ctx context.Context, authEnvironmentID, targetEnvironmentID, region, clientID, clientSecret string) (*Client, error) {
	if authEnvironmentID == "" {
		return nil, fmt.Errorf("auth environment ID is required")
	}
	if targetEnvironmentID == "" {
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

	// Create service configuration with OAuth credentials
	// Use authEnvironmentID for token acquisition
	serviceCfg := config.NewConfiguration().
		WithEnvironmentID(authEnvironmentID).
		WithTopLevelDomain(config.TopLevelDomain(getRegionDomain(region))).
		WithClientID(clientID).
		WithClientSecret(clientSecret).
		WithGrantType(oauth2.GrantTypeClientCredentials).
		WithStorageType(config.StorageTypeNone)

	// Initialize PingOne API client
	cfg := pingone.NewConfiguration(serviceCfg)
	apiClient, err := pingone.NewAPIClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API client: %w", err)
	}

	client := &Client{
		apiClient:         apiClient,
		serviceCfg:        serviceCfg,
		AuthEnvironmentID: authEnvironmentID,
		EnvironmentID:     targetEnvironmentID,
		Region:            region,
	}

	return client, nil
}

// NewClientSingleEnvironment creates a client where auth and target environment are the same
func NewClientSingleEnvironment(ctx context.Context, environmentID, region, clientID, clientSecret string) (*Client, error) {
	return NewClient(ctx, environmentID, environmentID, region, clientID, clientSecret)
}

// APIClient returns the underlying PingOne SDK client for direct SDK usage.
func (c *Client) APIClient() *pingone.APIClient {
	return c.apiClient
}

// getRegionDomain returns the domain suffix for a given region code
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

// ValidRegions returns the list of valid PingOne region codes
func ValidRegions() []string {
	return []string{"NA", "EU", "AP", "CA"}
}

// IsValidRegion checks if the given region code is valid
func IsValidRegion(region string) bool {
	for _, r := range ValidRegions() {
		if r == region {
			return true
		}
	}
	return false
}
