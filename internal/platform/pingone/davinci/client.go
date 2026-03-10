// Package davinci provides the unified service package for PingOne DaVinci.
// It implements the clients.APIClient interface and registers custom handlers
// and transforms for all DaVinci resource types.
//
// Adding a new resource requires only a single new resource_*.go file whose
// init() calls registerResource() and optionally registerHandler() /
// registerTransform(). No other files need editing.
package davinci

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
}

// New creates a DaVinci APIClient from a pre-built SDK client and environment ID.
func New(apiClient *pingone.APIClient, environmentID uuid.UUID) *Client {
	return &Client{apiClient: apiClient, environmentID: environmentID}
}

// Platform returns the platform identifier.
func (c *Client) Platform() string { return "pingone" }

// Service returns the service identifier.
func (c *Client) Service() string { return "davinci" }

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
