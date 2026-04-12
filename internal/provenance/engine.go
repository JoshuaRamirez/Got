package provenance

import (
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// bfsEngine computes provenance via BFS over the causal edge subgraph.
//
// Causal edges are treated as undirected for closure computation: the
// admissibility table mixes edge directions (derived_from points effect→cause,
// materializes points cause→effect), so undirected reachability is the
// conservative, correct interpretation of causal connectedness.
type bfsEngine struct {
	causal map[ontology.EdgeType]bool
}

// NewEngine creates a provenance engine that traverses the given set of
// causal edge types. Typically called with ontology.CausalEdges.
func NewEngine(causalEdges map[ontology.EdgeType]bool) Engine {
	c := make(map[ontology.EdgeType]bool, len(causalEdges))
	for k, v := range causalEdges {
		c[k] = v
	}
	return &bfsEngine{causal: c}
}

// buildAdj constructs an undirected adjacency list from all causal edges
// and causal hyperedges in the graph.
func (e *bfsEngine) buildAdj(g graph.Graph) map[identity.VertexID][]identity.VertexID {
	adj := make(map[identity.VertexID][]identity.VertexID)

	for _, edge := range g.Edges() {
		if !e.causal[edge.Type] {
			continue
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
		adj[edge.To] = append(adj[edge.To], edge.From)
	}

	for _, h := range g.Hyperedges() {
		if !e.causal[h.Type] {
			continue
		}
		// Connect every input to every output (undirected).
		for _, in := range h.Inputs {
			for _, out := range h.Outputs {
				adj[in] = append(adj[in], out)
				adj[out] = append(adj[out], in)
			}
		}
	}

	return adj
}

// Causes returns true if from and to are connected via causal edges.
func (e *bfsEngine) Causes(g graph.Graph, from, to identity.VertexID) (bool, error) {
	if from == to {
		return true, nil
	}
	adj := e.buildAdj(g)
	visited := map[identity.VertexID]bool{from: true}
	queue := []identity.VertexID{from}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nb := range adj[cur] {
			if nb == to {
				return true, nil
			}
			if !visited[nb] {
				visited[nb] = true
				queue = append(queue, nb)
			}
		}
	}
	return false, nil
}

// Cone returns the provenance cone of seed: all vertices reachable via causal
// edges. Equivalent to Close with a singleton seed set.
//
// Axiom: provCone(G, v) = provClose(G, {v}).
func (e *bfsEngine) Cone(g graph.Graph, seed identity.VertexID) ([]identity.VertexID, error) {
	return e.Close(g, []identity.VertexID{seed})
}

// Close computes the provenance closure of the seed set.
//
// Axiom: S subset Close(G, S)                               — extensive
// Axiom: S1 subset S2 => Close(G, S1) subset Close(G, S2)   — monotone
// Axiom: Close(G, Close(G, S)) = Close(G, S)                — idempotent
func (e *bfsEngine) Close(g graph.Graph, seed []identity.VertexID) ([]identity.VertexID, error) {
	adj := e.buildAdj(g)
	visited := make(map[identity.VertexID]bool, len(seed))
	queue := make([]identity.VertexID, 0, len(seed))

	for _, s := range seed {
		if !visited[s] {
			visited[s] = true
			queue = append(queue, s)
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nb := range adj[cur] {
			if !visited[nb] {
				visited[nb] = true
				queue = append(queue, nb)
			}
		}
	}

	result := make([]identity.VertexID, 0, len(visited))
	for id := range visited {
		result = append(result, id)
	}
	return result, nil
}

// TraceSet returns all simple paths between from and to through causal edges.
func (e *bfsEngine) TraceSet(g graph.Graph, from, to identity.VertexID) ([]Trace, error) {
	adj := e.buildAdj(g)
	var traces []Trace
	visited := map[identity.VertexID]bool{from: true}
	path := []identity.VertexID{from}

	var dfs func(cur identity.VertexID)
	dfs = func(cur identity.VertexID) {
		if cur == to {
			p := make([]identity.VertexID, len(path))
			copy(p, path)
			traces = append(traces, &simpleTrace{vertices: p})
			return
		}
		for _, nb := range adj[cur] {
			if !visited[nb] {
				visited[nb] = true
				path = append(path, nb)
				dfs(nb)
				path = path[:len(path)-1]
				visited[nb] = false
			}
		}
	}
	dfs(from)

	return traces, nil
}

// simpleTrace is a concrete Trace: an ordered sequence of vertex IDs.
type simpleTrace struct {
	vertices []identity.VertexID
}

func (t *simpleTrace) Vertices() []identity.VertexID {
	return t.vertices
}
