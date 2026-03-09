package core

import (
	"fmt"
	"sort"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// handlerQueueEntry pairs a handler name with its implementation.
type handlerQueueEntry struct {
	name string
	fn   CustomHandlerFunc
}

// transformQueueEntry pairs a transform name with its implementation.
type transformQueueEntry struct {
	name string
	fn   CustomTransformFunc
}

// CustomHandlerQueue collects custom handlers and transforms registered by
// resource init() functions. It is not concurrency-safe because all init()
// calls run sequentially before main(). Call LoadInto() once at startup to
// transfer entries into the concurrency-safe CustomHandlerRegistry.
type CustomHandlerQueue struct {
	handlers   []handlerQueueEntry
	transforms []transformQueueEntry
}

// NewCustomHandlerQueue creates an empty queue.
func NewCustomHandlerQueue() *CustomHandlerQueue {
	return &CustomHandlerQueue{}
}

// AddHandler queues a custom handler for later bulk registration.
func (q *CustomHandlerQueue) AddHandler(name string, fn CustomHandlerFunc) {
	q.handlers = append(q.handlers, handlerQueueEntry{name: name, fn: fn})
}

// AddTransform queues a custom transform for later bulk registration.
func (q *CustomHandlerQueue) AddTransform(name string, fn CustomTransformFunc) {
	q.transforms = append(q.transforms, transformQueueEntry{name: name, fn: fn})
}

// LoadInto transfers all queued handlers and transforms into the given registry.
func (q *CustomHandlerQueue) LoadInto(reg *CustomHandlerRegistry) {
	for _, h := range q.handlers {
		reg.RegisterHandler(h.name, h.fn)
	}
	for _, t := range q.transforms {
		reg.RegisterTransform(t.name, t.fn)
	}
}

// HandlerNames returns sorted names of all queued handlers.
func (q *CustomHandlerQueue) HandlerNames() []string {
	names := make([]string, len(q.handlers))
	for i, h := range q.handlers {
		names[i] = h.name
	}
	sort.Strings(names)
	return names
}

// TransformNames returns sorted names of all queued transforms.
func (q *CustomHandlerQueue) TransformNames() []string {
	names := make([]string, len(q.transforms))
	for i, t := range q.transforms {
		names[i] = t.name
	}
	sort.Strings(names)
	return names
}

// ── Stub helpers ─────────────────────────────────────────────────

// StubHandler returns a CustomHandlerFunc stub that returns a
// not-yet-implemented error. Use during development to register a handler
// before its real implementation exists.
func StubHandler(handlerName string) CustomHandlerFunc {
	return func(_ interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error) {
		return nil, fmt.Errorf("%s: not yet implemented for %s", handlerName, def.Metadata.ResourceType)
	}
}

// StubTransform returns a CustomTransformFunc stub that passes through the
// value unmodified. Use during development to register a transform before
// its real implementation exists.
func StubTransform() CustomTransformFunc {
	return func(value interface{}, _ interface{}, _ *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
		return value, nil
	}
}
