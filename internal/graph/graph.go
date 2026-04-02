package graph

// Package graph provides a dependency graph for tracking relationships between
// Terraform resources. It supports cycle detection, topological ordering, and
// cross-resource reference resolution.

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ResourceNode represents a resource in the dependency graph.
type ResourceNode struct {
	ResourceType string // Full Terraform resource type (e.g., "pingone_davinci_flow")
	ID           string // Original API resource ID
	Name         string // Sanitized Terraform resource label
}

// Edge represents a dependency from one resource to another.
type Edge struct {
	From     ResourceNode
	To       ResourceNode
	Field    string // Field name containing the reference
	Location string // Location in the data structure (e.g., "graphData.nodes[5].connectionId")
}

// DependencyGraph tracks resources and their dependency relationships.
// It is concurrency-safe.
type DependencyGraph struct {
	nodes     map[string]ResourceNode // composite key -> node
	edges     []Edge
	nameUsage map[string]int
	mu        sync.RWMutex
}

// New creates an empty dependency graph.
func New() *DependencyGraph {
	return &DependencyGraph{
		nodes:     make(map[string]ResourceNode),
		edges:     make([]Edge, 0),
		nameUsage: make(map[string]int),
	}
}

// AddResource registers a resource node. The name should already be sanitized.
// Duplicate names receive a numeric suffix (_2, _3, ...).
func (g *DependencyGraph) AddResource(resourceType, id, name string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	uniqueName := g.ensureUniqueName(name)
	key := makeKey(resourceType, id)
	g.nodes[key] = ResourceNode{
		ResourceType: resourceType,
		ID:           id,
		Name:         uniqueName,
	}
}

// AddDependency records that `from` depends on `to`.
func (g *DependencyGraph) AddDependency(from, to ResourceNode, field, location string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.edges = append(g.edges, Edge{
		From:     from,
		To:       to,
		Field:    field,
		Location: location,
	})
}

// AddEdge is a convenience wrapper that creates an edge between two resources
// identified by type and ID.
func (g *DependencyGraph) AddEdge(fromType, fromID, toType, toID, field, location string) error {
	g.mu.RLock()
	fromNode, fromOK := g.nodes[makeKey(fromType, fromID)]
	toNode, toOK := g.nodes[makeKey(toType, toID)]
	g.mu.RUnlock()

	if !fromOK {
		return fmt.Errorf("source resource not found: %s:%s", fromType, fromID)
	}
	if !toOK {
		return fmt.Errorf("target resource not found: %s:%s", toType, toID)
	}

	g.AddDependency(fromNode, toNode, field, location)
	return nil
}

// GetNode returns the node for a given resource type and ID.
func (g *DependencyGraph) GetNode(resourceType, id string) (ResourceNode, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, ok := g.nodes[makeKey(resourceType, id)]
	if !ok {
		return ResourceNode{}, fmt.Errorf("resource not found: %s:%s", resourceType, id)
	}
	return node, nil
}

// HasResource reports whether a resource exists in the graph.
func (g *DependencyGraph) HasResource(resourceType, id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	_, ok := g.nodes[makeKey(resourceType, id)]
	return ok
}

// GetReferenceName returns the Terraform resource label for a resource.
func (g *DependencyGraph) GetReferenceName(resourceType, id string) (string, error) {
	node, err := g.GetNode(resourceType, id)
	if err != nil {
		return "", err
	}
	return node.Name, nil
}

// GetDependencies returns all edges where the given resource is the source.
func (g *DependencyGraph) GetDependencies(resourceType, id string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := makeKey(resourceType, id)
	var result []Edge
	for _, edge := range g.edges {
		if makeKey(edge.From.ResourceType, edge.From.ID) == key {
			result = append(result, edge)
		}
	}
	return result
}

// GetDependents returns all edges where the given resource is the target.
func (g *DependencyGraph) GetDependents(resourceType, id string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := makeKey(resourceType, id)
	var result []Edge
	for _, edge := range g.edges {
		if makeKey(edge.To.ResourceType, edge.To.ID) == key {
			result = append(result, edge)
		}
	}
	return result
}

// WalkDependencies performs a BFS from the given seed nodes, following outgoing
// edges (dependencies), and returns all transitively reachable nodes including
// the seeds themselves. Seeds not present in the graph are silently skipped.
// The result is deterministic (sorted by composite key).
func (g *DependencyGraph) WalkDependencies(seeds []ResourceNode) []ResourceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(seeds) == 0 {
		return []ResourceNode{}
	}

	visited := make(map[string]bool)
	var queue []string // composite keys for BFS queue
	var result []ResourceNode

	// Add seeds to queue, but only if they exist in the graph.
	// Silently skip seeds not in the graph.
	for _, seed := range seeds {
		key := makeKey(seed.ResourceType, seed.ID)
		if _, ok := g.nodes[key]; ok {
			if !visited[key] {
				visited[key] = true
				queue = append(queue, key)
				result = append(result, g.nodes[key])
			}
		}
	}

	// BFS traversal following outgoing edges.
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find all outgoing edges from current node.
		for _, edge := range g.edges {
			fromKey := makeKey(edge.From.ResourceType, edge.From.ID)
			if fromKey == current {
				toKey := makeKey(edge.To.ResourceType, edge.To.ID)
				if !visited[toKey] {
					visited[toKey] = true
					queue = append(queue, toKey)
					result = append(result, g.nodes[toKey])
				}
			}
		}
	}

	// Sort by composite key for determinism.
	sort.Slice(result, func(i, j int) bool {
		keyI := makeKey(result[i].ResourceType, result[i].ID)
		keyJ := makeKey(result[j].ResourceType, result[j].ID)
		return keyI < keyJ
	})

	return result
}

// AllNodes returns all nodes in the graph.
func (g *DependencyGraph) AllNodes() []ResourceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]ResourceNode, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// AllEdges returns all edges in the graph.
func (g *DependencyGraph) AllEdges() []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edges := make([]Edge, len(g.edges))
	copy(edges, g.edges)
	return edges
}

// NodeCount returns the number of nodes.
func (g *DependencyGraph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.nodes)
}

// EdgeCount returns the number of edges.
func (g *DependencyGraph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.edges)
}

// --- Cycle detection ---

// CycleError represents a circular dependency.
type CycleError struct {
	Cycle []ResourceNode
}

// Error returns a human-readable cycle description.
func (e *CycleError) Error() string {
	parts := make([]string, len(e.Cycle))
	for i, n := range e.Cycle {
		parts[i] = fmt.Sprintf("%s:%s", n.ResourceType, n.ID)
	}
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(parts, " -> "))
}

// DetectCycles finds circular dependencies using DFS.
func (g *DependencyGraph) DetectCycles() [][]ResourceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.detectCyclesLocked()
}

// detectCyclesLocked is the internal unlocked variant for use within
// already-locked methods.
func (g *DependencyGraph) detectCyclesLocked() [][]ResourceNode {
	var cycles [][]ResourceNode
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for key := range g.nodes {
		if !visited[key] {
			if cycle := g.detectCycleDFS(key, visited, recStack, nil); cycle != nil {
				cycles = append(cycles, cycle)
			}
		}
	}
	return cycles
}

func (g *DependencyGraph) detectCycleDFS(key string, visited, recStack map[string]bool, path []ResourceNode) []ResourceNode {
	visited[key] = true
	recStack[key] = true
	node := g.nodes[key]
	path = append(path, node)

	for _, edge := range g.edges {
		fromKey := makeKey(edge.From.ResourceType, edge.From.ID)
		if fromKey != key {
			continue
		}
		toKey := makeKey(edge.To.ResourceType, edge.To.ID)

		if recStack[toKey] {
			// Found cycle - extract the cycle portion
			start := -1
			for i, n := range path {
				if makeKey(n.ResourceType, n.ID) == toKey {
					start = i
					break
				}
			}
			if start >= 0 {
				cycle := make([]ResourceNode, len(path[start:]))
				copy(cycle, path[start:])
				cycle = append(cycle, g.nodes[toKey])
				return cycle
			}
		}

		if !visited[toKey] {
			if cycle := g.detectCycleDFS(toKey, visited, recStack, path); cycle != nil {
				return cycle
			}
		}
	}

	recStack[key] = false
	return nil
}

// --- Topological sort ---

// TopologicalSort returns nodes in dependency order (dependencies appear before
// dependents). Returns an error if the graph contains cycles.
func (g *DependencyGraph) TopologicalSort() ([]ResourceNode, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cycles := g.detectCyclesLocked()
	if len(cycles) > 0 {
		return nil, &CycleError{Cycle: cycles[0]}
	}

	// Build adjacency list and in-degree map.
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)
	for key := range g.nodes {
		inDegree[key] = 0
		adjList[key] = nil
	}
	for _, edge := range g.edges {
		fromKey := makeKey(edge.From.ResourceType, edge.From.ID)
		toKey := makeKey(edge.To.ResourceType, edge.To.ID)
		adjList[toKey] = append(adjList[toKey], fromKey)
		inDegree[fromKey]++
	}

	// Deterministic queue ordering by key so output is stable.
	var queue []string
	for key, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, key)
		}
	}
	sort.Strings(queue)

	var sorted []ResourceNode
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, g.nodes[current])

		neighbors := adjList[current]
		sort.Strings(neighbors)
		for _, nb := range neighbors {
			inDegree[nb]--
			if inDegree[nb] == 0 {
				queue = append(queue, nb)
			}
		}
	}

	if len(sorted) != len(g.nodes) {
		return nil, fmt.Errorf("topological sort failed: possible cycle")
	}
	return sorted, nil
}

// --- Validation ---

// Validate checks the graph for cycles and dangling references.
func (g *DependencyGraph) Validate() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cycles := g.detectCyclesLocked()
	if len(cycles) > 0 {
		var msgs []string
		for _, cycle := range cycles {
			parts := make([]string, len(cycle))
			for i, n := range cycle {
				parts[i] = fmt.Sprintf("%s:%s", n.ResourceType, n.ID)
			}
			msgs = append(msgs, strings.Join(parts, " -> "))
		}
		return fmt.Errorf("circular dependencies:\n  %s", strings.Join(msgs, "\n  "))
	}

	// Check for dangling edge references.
	for _, edge := range g.edges {
		fromKey := makeKey(edge.From.ResourceType, edge.From.ID)
		toKey := makeKey(edge.To.ResourceType, edge.To.ID)
		if _, ok := g.nodes[fromKey]; !ok {
			return fmt.Errorf("edge references non-existent source: %s %s", edge.From.ResourceType, edge.From.ID)
		}
		if _, ok := g.nodes[toKey]; !ok {
			return fmt.Errorf("edge references non-existent target: %s %s", edge.To.ResourceType, edge.To.ID)
		}
	}

	return nil
}

// GenerateTerraformReference returns the Terraform resource reference string
// for a resource (e.g., "pingone_davinci_flow.my_flow.id").
func (g *DependencyGraph) GenerateTerraformReference(resourceType, resourceID, attribute string) (string, error) {
	name, err := g.GetReferenceName(resourceType, resourceID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s.%s", resourceType, name, attribute), nil
}

// --- Helpers ---

func makeKey(resourceType, id string) string {
	return resourceType + ":" + id
}

func (g *DependencyGraph) ensureUniqueName(name string) string {
	count, exists := g.nameUsage[name]
	if !exists {
		g.nameUsage[name] = 1
		return name
	}
	g.nameUsage[name] = count + 1
	return fmt.Sprintf("%s_%d", name, count+1)
}
