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
