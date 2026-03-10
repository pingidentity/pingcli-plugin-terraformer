package clients

import (
	"context"
)

// APIClient defines the generic interface for platform API interactions.
// Each platform/service combination implements this interface to provide
// resource listing and retrieval capabilities to the core processor.
type APIClient interface {
	// ListResources retrieves all resources of the given type from the environment.
	// Returns a slice of raw data objects suitable for schema-driven field extraction.
	ListResources(ctx context.Context, resourceType string, envID string) ([]interface{}, error)

	// GetResource retrieves a single resource by type and ID.
	GetResource(ctx context.Context, resourceType string, envID string, resourceID string) (interface{}, error)

	// Platform returns the platform identifier (e.g., "pingone").
	Platform() string

	// Service returns the service identifier within the platform (e.g., "davinci").
	Service() string
}
