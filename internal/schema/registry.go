package schema

import (
	"fmt"
	"io/fs"
	"sync"
)

// Registry is a thread-safe registry of resource definitions
type Registry struct {
	definitions map[string]*ResourceDefinition
	mu          sync.RWMutex
	loader      *Loader
}

// NewRegistry creates a new resource registry
func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[string]*ResourceDefinition),
		loader:      NewLoader(),
	}
}

// Register registers a resource definition
func (r *Registry) Register(def *ResourceDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.definitions[def.Metadata.ResourceType]; exists {
		return fmt.Errorf("resource type %s is already registered", def.Metadata.ResourceType)
	}

	r.definitions[def.Metadata.ResourceType] = def
	return nil
}

// Get retrieves a resource definition by resource type
func (r *Registry) Get(resourceType string) (*ResourceDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.definitions[resourceType]
	if !exists {
		return nil, fmt.Errorf("resource type %s not found in registry", resourceType)
	}

	return def, nil
}

// ListAll returns all registered resource definitions
func (r *Registry) ListAll() []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]*ResourceDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		defs = append(defs, def)
	}

	return defs
}

// ListByPlatform returns all resource definitions for a specific platform
func (r *Registry) ListByPlatform(platform string) []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]*ResourceDefinition, 0)
	for _, def := range r.definitions {
		if def.Metadata.Platform == platform {
			defs = append(defs, def)
		}
	}

	return defs
}

// ListByService returns all resource definitions for a specific platform and service
func (r *Registry) ListByService(platform, service string) []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]*ResourceDefinition, 0)
	for _, def := range r.definitions {
		if def.Metadata.Platform == platform && def.Metadata.Service == service {
			defs = append(defs, def)
		}
	}

	return defs
}

// LoadFromDirectory loads all definitions from a directory and registers them
func (r *Registry) LoadFromDirectory(dir string) error {
	definitions, err := r.loader.LoadFromDirectory(dir)
	if err != nil {
		return err
	}

	for _, def := range definitions {
		if err := r.Register(def); err != nil {
			return fmt.Errorf("failed to register %s: %w", def.Metadata.ResourceType, err)
		}
	}

	return nil
}

// LoadPlatform loads all definitions for a specific platform
func (r *Registry) LoadPlatform(baseDir, platform string) error {
	definitions, err := r.loader.LoadPlatformDefinitions(baseDir, platform)
	if err != nil {
		return err
	}

	for _, def := range definitions {
		if err := r.Register(def); err != nil {
			return fmt.Errorf("failed to register %s: %w", def.Metadata.ResourceType, err)
		}
	}

	return nil
}

// LoadFromFS loads all definitions from an fs.FS (e.g., embed.FS) and registers them.
// The dir parameter is the root path within the FS to walk.
func (r *Registry) LoadFromFS(fsys fs.FS, dir string) error {
	definitions, err := r.loader.LoadFromFS(fsys, dir)
	if err != nil {
		return err
	}

	for _, def := range definitions {
		if err := r.Register(def); err != nil {
			return fmt.Errorf("failed to register %s: %w", def.Metadata.ResourceType, err)
		}
	}

	return nil
}

// Count returns the number of registered definitions
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.definitions)
}

// Clear removes all registered definitions (for testing)
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.definitions = make(map[string]*ResourceDefinition)
}
