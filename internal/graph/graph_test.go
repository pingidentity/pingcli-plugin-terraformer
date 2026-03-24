package graph

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGraph(t *testing.T) {
	g := New()
	assert.Equal(t, 0, g.NodeCount())
	assert.Equal(t, 0, g.EdgeCount())
}

func TestAddResourceAndGet(t *testing.T) {
	g := New()
	g.AddResource("pingone_davinci_flow", "flow-1", "my_flow")

	node, err := g.GetNode("pingone_davinci_flow", "flow-1")
	require.NoError(t, err)
	assert.Equal(t, "my_flow", node.Name)
	assert.Equal(t, "flow-1", node.ID)
	assert.Equal(t, "pingone_davinci_flow", node.ResourceType)
}

func TestGetNodeNotFound(t *testing.T) {
	g := New()
	_, err := g.GetNode("type", "missing")
	require.Error(t, err)
}

func TestHasResource(t *testing.T) {
	g := New()
	assert.False(t, g.HasResource("type", "id"))
	g.AddResource("type", "id", "name")
	assert.True(t, g.HasResource("type", "id"))
}

func TestGetReferenceName(t *testing.T) {
	g := New()
	g.AddResource("pingone_davinci_flow", "f1", "login_flow")

	name, err := g.GetReferenceName("pingone_davinci_flow", "f1")
	require.NoError(t, err)
	assert.Equal(t, "login_flow", name)

	_, err = g.GetReferenceName("type", "missing")
	require.Error(t, err)
}

func TestUniqueNames(t *testing.T) {
	g := New()
	g.AddResource("type", "1", "name")
	g.AddResource("type", "2", "name")
	g.AddResource("type", "3", "name")

	n1, _ := g.GetReferenceName("type", "1")
	n2, _ := g.GetReferenceName("type", "2")
	n3, _ := g.GetReferenceName("type", "3")

	assert.Equal(t, "name", n1)
	assert.Equal(t, "name_2", n2)
	assert.Equal(t, "name_3", n3)
}

func TestAddDependencyAndQuery(t *testing.T) {
	g := New()
	g.AddResource("flow", "f1", "flow1")
	g.AddResource("connector", "c1", "conn1")

	from, _ := g.GetNode("flow", "f1")
	to, _ := g.GetNode("connector", "c1")
	g.AddDependency(from, to, "connectionId", "graphData.nodes[0]")

	deps := g.GetDependencies("flow", "f1")
	require.Len(t, deps, 1)
	assert.Equal(t, "c1", deps[0].To.ID)
	assert.Equal(t, "connectionId", deps[0].Field)

	dependents := g.GetDependents("connector", "c1")
	require.Len(t, dependents, 1)
	assert.Equal(t, "f1", dependents[0].From.ID)

	assert.Empty(t, g.GetDependencies("connector", "c1"))
}

func TestAddEdge(t *testing.T) {
	g := New()
	g.AddResource("flow", "f1", "flow1")
	g.AddResource("connector", "c1", "conn1")

	err := g.AddEdge("flow", "f1", "connector", "c1", "field", "loc")
	require.NoError(t, err)
	assert.Equal(t, 1, g.EdgeCount())

	err = g.AddEdge("missing", "x", "connector", "c1", "", "")
	require.Error(t, err)

	err = g.AddEdge("flow", "f1", "missing", "x", "", "")
	require.Error(t, err)
}

func TestDetectCyclesNoCycle(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "a1")
	g.AddResource("b", "2", "b2")
	g.AddResource("c", "3", "c3")

	a, _ := g.GetNode("a", "1")
	b, _ := g.GetNode("b", "2")
	c, _ := g.GetNode("c", "3")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(b, c, "ref", "")

	cycles := g.DetectCycles()
	assert.Empty(t, cycles)
}

func TestDetectCyclesWithCycle(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "a1")
	g.AddResource("b", "2", "b2")
	g.AddResource("c", "3", "c3")

	a, _ := g.GetNode("a", "1")
	b, _ := g.GetNode("b", "2")
	c, _ := g.GetNode("c", "3")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(b, c, "ref", "")
	g.AddDependency(c, a, "ref", "")

	cycles := g.DetectCycles()
	require.NotEmpty(t, cycles)
}

func TestTopologicalSortLinear(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "a1")
	g.AddResource("b", "2", "b2")
	g.AddResource("c", "3", "c3")

	a, _ := g.GetNode("a", "1")
	b, _ := g.GetNode("b", "2")
	c, _ := g.GetNode("c", "3")

	g.AddDependency(c, b, "ref", "")
	g.AddDependency(b, a, "ref", "")

	sorted, err := g.TopologicalSort()
	require.NoError(t, err)
	require.Len(t, sorted, 3)

	idxOf := func(id string) int {
		for i, n := range sorted {
			if n.ID == id {
				return i
			}
		}
		return -1
	}
	assert.Less(t, idxOf("1"), idxOf("2"))
	assert.Less(t, idxOf("2"), idxOf("3"))
}

func TestTopologicalSortWithCycle(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "a1")
	g.AddResource("b", "2", "b2")

	a, _ := g.GetNode("a", "1")
	b, _ := g.GetNode("b", "2")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(b, a, "ref", "")

	_, err := g.TopologicalSort()
	require.Error(t, err)
}

func TestTopologicalSortNoEdges(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "a1")
	g.AddResource("b", "2", "b2")

	sorted, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, sorted, 2)
}

func TestValidateClean(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "a1")
	g.AddResource("b", "2", "b2")
	_ = g.AddEdge("a", "1", "b", "2", "ref", "")

	err := g.Validate()
	require.NoError(t, err)
}

func TestValidateWithCycle(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "a1")
	g.AddResource("b", "2", "b2")

	a, _ := g.GetNode("a", "1")
	b, _ := g.GetNode("b", "2")
	g.AddDependency(a, b, "", "")
	g.AddDependency(b, a, "", "")

	err := g.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}

func TestGenerateTerraformReference(t *testing.T) {
	g := New()
	g.AddResource("pingone_davinci_flow", "f1", "login_flow")

	ref, err := g.GenerateTerraformReference("pingone_davinci_flow", "f1", "id")
	require.NoError(t, err)
	assert.Equal(t, "pingone_davinci_flow.login_flow.id", ref)

	_, err = g.GenerateTerraformReference("missing", "x", "id")
	require.Error(t, err)
}

func TestAllNodesAndEdges(t *testing.T) {
	g := New()
	g.AddResource("a", "1", "n1")
	g.AddResource("b", "2", "n2")
	_ = g.AddEdge("a", "1", "b", "2", "ref", "loc")

	nodes := g.AllNodes()
	assert.Len(t, nodes, 2)

	edges := g.AllEdges()
	assert.Len(t, edges, 1)
}

func TestCycleErrorMessage(t *testing.T) {
	err := &CycleError{
		Cycle: []ResourceNode{
			{ResourceType: "a", ID: "1"},
			{ResourceType: "b", ID: "2"},
			{ResourceType: "a", ID: "1"},
		},
	}
	assert.Contains(t, err.Error(), "a:1")
	assert.Contains(t, err.Error(), "b:2")
}

func TestConcurrentAccess(t *testing.T) {
	g := New()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("id-%d", n)
			g.AddResource("type", id, fmt.Sprintf("name_%d", n))
			g.HasResource("type", id)
			_ = g.AllNodes()
			_ = g.DetectCycles()
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 50, g.NodeCount())
}

func TestDiamondDependency(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")
	g.AddResource("t", "c", "c")
	g.AddResource("t", "d", "d")

	a, _ := g.GetNode("t", "a")
	b, _ := g.GetNode("t", "b")
	c, _ := g.GetNode("t", "c")
	d, _ := g.GetNode("t", "d")

	g.AddDependency(a, b, "", "")
	g.AddDependency(a, c, "", "")
	g.AddDependency(b, d, "", "")
	g.AddDependency(c, d, "", "")

	cycles := g.DetectCycles()
	assert.Empty(t, cycles)

	sorted, err := g.TopologicalSort()
	require.NoError(t, err)
	require.Len(t, sorted, 4)

	idxOf := func(id string) int {
		for i, n := range sorted {
			if n.ID == id {
				return i
			}
		}
		return -1
	}
	assert.Less(t, idxOf("d"), idxOf("b"))
	assert.Less(t, idxOf("d"), idxOf("c"))
	assert.Less(t, idxOf("b"), idxOf("a"))
	assert.Less(t, idxOf("c"), idxOf("a"))
}

func TestWalkDependencies_EmptySeeds(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")

	result := g.WalkDependencies([]ResourceNode{})
	assert.Empty(t, result)
}

func TestWalkDependencies_SingleSeedNoDeps(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")

	a, _ := g.GetNode("t", "a")
	result := g.WalkDependencies([]ResourceNode{a})

	require.Len(t, result, 1)
	assert.Equal(t, "a", result[0].ID)
}

func TestWalkDependencies_LinearChain(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")
	g.AddResource("t", "c", "c")

	a, _ := g.GetNode("t", "a")
	b, _ := g.GetNode("t", "b")
	c, _ := g.GetNode("t", "c")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(b, c, "ref", "")

	result := g.WalkDependencies([]ResourceNode{a})

	require.Len(t, result, 3)
	resultKeys := nodeKeys(result)
	expectedKeys := []string{makeKey("t", "a"), makeKey("t", "b"), makeKey("t", "c")}
	assert.Equal(t, expectedKeys, resultKeys)
}

func TestWalkDependencies_DiamondGraph(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")
	g.AddResource("t", "c", "c")
	g.AddResource("t", "d", "d")

	a, _ := g.GetNode("t", "a")
	b, _ := g.GetNode("t", "b")
	c, _ := g.GetNode("t", "c")
	d, _ := g.GetNode("t", "d")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(a, c, "ref", "")
	g.AddDependency(b, d, "ref", "")
	g.AddDependency(c, d, "ref", "")

	result := g.WalkDependencies([]ResourceNode{a})

	require.Len(t, result, 4)
	resultKeys := nodeKeys(result)
	expectedKeys := []string{
		makeKey("t", "a"),
		makeKey("t", "b"),
		makeKey("t", "c"),
		makeKey("t", "d"),
	}
	assert.Equal(t, expectedKeys, resultKeys)
}

func TestWalkDependencies_CircularDeps(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")
	g.AddResource("t", "c", "c")

	a, _ := g.GetNode("t", "a")
	b, _ := g.GetNode("t", "b")
	c, _ := g.GetNode("t", "c")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(b, c, "ref", "")
	g.AddDependency(c, a, "ref", "")

	result := g.WalkDependencies([]ResourceNode{a})

	require.Len(t, result, 3)
	resultKeys := nodeKeys(result)
	expectedKeys := []string{
		makeKey("t", "a"),
		makeKey("t", "b"),
		makeKey("t", "c"),
	}
	assert.Equal(t, expectedKeys, resultKeys)
}

func TestWalkDependencies_MultipleSeeds(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")
	g.AddResource("t", "c", "c")
	g.AddResource("t", "d", "d")

	a, _ := g.GetNode("t", "a")
	b, _ := g.GetNode("t", "b")
	c, _ := g.GetNode("t", "c")
	d, _ := g.GetNode("t", "d")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(c, d, "ref", "")

	result := g.WalkDependencies([]ResourceNode{a, c})

	require.Len(t, result, 4)
	resultKeys := nodeKeys(result)
	expectedKeys := []string{
		makeKey("t", "a"),
		makeKey("t", "b"),
		makeKey("t", "c"),
		makeKey("t", "d"),
	}
	assert.Equal(t, expectedKeys, resultKeys)
}

func TestWalkDependencies_SeedNotInGraph(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")

	a, _ := g.GetNode("t", "a")
	b, _ := g.GetNode("t", "b")
	missingNode := ResourceNode{
		ResourceType: "t",
		ID:           "missing",
		Name:         "missing",
	}

	g.AddDependency(a, b, "ref", "")

	result := g.WalkDependencies([]ResourceNode{a, missingNode})

	require.Len(t, result, 2)
	resultKeys := nodeKeys(result)
	expectedKeys := []string{
		makeKey("t", "a"),
		makeKey("t", "b"),
	}
	assert.Equal(t, expectedKeys, resultKeys)
}

func TestWalkDependencies_PartialOverlap(t *testing.T) {
	g := New()
	g.AddResource("t", "a", "a")
	g.AddResource("t", "b", "b")
	g.AddResource("t", "c", "c")

	a, _ := g.GetNode("t", "a")
	b, _ := g.GetNode("t", "b")
	c, _ := g.GetNode("t", "c")

	g.AddDependency(a, b, "ref", "")
	g.AddDependency(c, b, "ref", "")

	result := g.WalkDependencies([]ResourceNode{a, c})

	require.Len(t, result, 3)
	resultKeys := nodeKeys(result)
	expectedKeys := []string{
		makeKey("t", "a"),
		makeKey("t", "b"),
		makeKey("t", "c"),
	}
	assert.Equal(t, expectedKeys, resultKeys)
}

// nodeKeys is a helper that extracts and sorts the composite keys from a slice of ResourceNodes.
func nodeKeys(nodes []ResourceNode) []string {
	keys := make([]string, len(nodes))
	for i, n := range nodes {
		keys[i] = makeKey(n.ResourceType, n.ID)
	}
	sort.Strings(keys)
	return keys
}
