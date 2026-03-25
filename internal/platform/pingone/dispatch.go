package pingone

import (
	"context"
	"fmt"
	"sort"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
)

// ── API client dispatch ──────────────────────────────────────────

// resourceHandler provides list and get operations for a single resource type.
type resourceHandler struct {
	list func(ctx context.Context, c *Client, envID string) ([]interface{}, error)
	get  func(ctx context.Context, c *Client, envID string, resourceID string) (interface{}, error)
}

// resourceHandlers is the dispatch table populated by init() in resource_*.go files.
var resourceHandlers = map[string]resourceHandler{}

// registerResource adds a resource type handler to the API dispatch table.
// Called from init() in each resource_*.go file.
func registerResource(resourceType string, h resourceHandler) {
	if _, exists := resourceHandlers[resourceType]; exists {
		panic(fmt.Sprintf("duplicate resource handler registration: %s", resourceType))
	}
	resourceHandlers[resourceType] = h
}

// SupportedResourceTypes returns the sorted list of registered resource type names.
func SupportedResourceTypes() []string {
	types := make([]string, 0, len(resourceHandlers))
	for rt := range resourceHandlers {
		types = append(types, rt)
	}
	sort.Strings(types)
	return types
}

// isSupported returns true if the resource type has a registered handler.
func isSupported(resourceType string) bool {
	_, ok := resourceHandlers[resourceType]
	return ok
}

// ── Custom handler / transform dispatch ─────────────────────────
//
// Delegates to core.CustomHandlerQueue. The package-level functions below
// preserve the simple registerHandler() / registerTransform() call sites in
// resource_*.go init() functions.

// customHandlerQueue collects registrations made during init().
var customHandlerQueue = core.NewCustomHandlerQueue()

// registerHandler queues a custom handler for bulk registration.
// Called from init() in resource_*.go files.
func registerHandler(name string, fn core.CustomHandlerFunc) {
	customHandlerQueue.AddHandler(name, fn)
}

// registerTransform queues a custom transform for bulk registration.
// Called from init() in resource_*.go files.
func registerTransform(name string, fn core.CustomTransformFunc) {
	customHandlerQueue.AddTransform(name, fn)
}

// RegisterCustomHandlers loads all DaVinci custom handlers and transforms
// into the given core registry. Call once during application startup.
func RegisterCustomHandlers(reg *core.CustomHandlerRegistry) {
	customHandlerQueue.LoadInto(reg)
}

// ── Embedded reference rule dispatch ────────────────────────────

// embeddedRefRules collects rules registered during init().
var embeddedRefRules []core.EmbeddedReferenceRule

// registerEmbeddedReferenceRule queues an embedded reference rule for
// bulk registration. Called from init() in resource_*.go files.
func registerEmbeddedReferenceRule(rule core.EmbeddedReferenceRule) {
	embeddedRefRules = append(embeddedRefRules, rule)
}

// NewEmbeddedReferenceRegistry creates an EmbeddedReferenceRegistry populated
// with all rules registered during init(). Call once during application startup.
func NewEmbeddedReferenceRegistry() *core.EmbeddedReferenceRegistry {
	reg := core.NewEmbeddedReferenceRegistry()
	for _, rule := range embeddedRefRules {
		reg.Register(rule)
	}
	return reg
}

// RegisteredHandlerNames returns sorted names of all queued handlers.
func RegisteredHandlerNames() []string {
	return customHandlerQueue.HandlerNames()
}

// RegisteredTransformNames returns sorted names of all queued transforms.
func RegisteredTransformNames() []string {
	return customHandlerQueue.TransformNames()
}

// ── Shared stub helpers ─────────────────────────────────────────

// stubHandler delegates to core.StubHandler.
func stubHandler(handlerName string) core.CustomHandlerFunc {
	return core.StubHandler(handlerName)
}

// stubTransform delegates to core.StubTransform.
func stubTransform() core.CustomTransformFunc {
	return core.StubTransform()
}
