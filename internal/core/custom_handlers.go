// Package core provides the generic resource processing engine.
package core

import (
	"fmt"
	"sync"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// CustomHandlerFunc processes resource data using logic too complex to express
// in YAML attribute definitions alone. It receives the full API data and the
// resource definition, and returns a map of additional attribute key/value pairs
// to merge into the processed resource.
type CustomHandlerFunc func(data interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error)

// CustomTransformFunc processes a single attribute value using resource-specific
// logic that goes beyond standard transforms. It receives the raw attribute
// value, the full API data for context, the attribute definition, and the
// parent resource definition (for accessing resource-level config like
// variable_prefix).
type CustomTransformFunc func(value interface{}, apiData interface{}, attr *schema.AttributeDefinition, def *schema.ResourceDefinition) (interface{}, error)

// CustomHandlerRegistry maps handler names declared in YAML to their Go
// implementations. It is concurrency-safe.
type CustomHandlerRegistry struct {
	handlers   map[string]CustomHandlerFunc
	transforms map[string]CustomTransformFunc
	mu         sync.RWMutex
}

// NewCustomHandlerRegistry creates an empty handler registry.
func NewCustomHandlerRegistry() *CustomHandlerRegistry {
	return &CustomHandlerRegistry{
		handlers:   make(map[string]CustomHandlerFunc),
		transforms: make(map[string]CustomTransformFunc),
	}
}

// RegisterHandler adds or replaces a named HCL-generator handler.
func (r *CustomHandlerRegistry) RegisterHandler(name string, fn CustomHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = fn
}

// RegisterTransform adds or replaces a named custom transform.
func (r *CustomHandlerRegistry) RegisterTransform(name string, fn CustomTransformFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transforms[name] = fn
}

// GetHandler retrieves a named handler. Returns an error if not found.
func (r *CustomHandlerRegistry) GetHandler(name string) (CustomHandlerFunc, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.handlers[name]
	if !ok {
		return nil, fmt.Errorf("custom handler %q not registered", name)
	}
	return fn, nil
}

// GetTransform retrieves a named custom transform. Returns an error if not found.
func (r *CustomHandlerRegistry) GetTransform(name string) (CustomTransformFunc, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.transforms[name]
	if !ok {
		return nil, fmt.Errorf("custom transform %q not registered", name)
	}
	return fn, nil
}

// HasHandler reports whether the named handler exists.
func (r *CustomHandlerRegistry) HasHandler(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[name]
	return ok
}

// HasTransform reports whether the named custom transform exists.
func (r *CustomHandlerRegistry) HasTransform(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.transforms[name]
	return ok
}

// ListHandlers returns all registered handler names.
func (r *CustomHandlerRegistry) ListHandlers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// ListTransforms returns all registered custom transform names.
func (r *CustomHandlerRegistry) ListTransforms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.transforms))
	for name := range r.transforms {
		names = append(names, name)
	}
	return names
}
