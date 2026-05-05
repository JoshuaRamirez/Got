package provenance

import (
	"context"

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
		for _, in := range h.Inputs {
			for _, out := range h.Outputs {
				adj[in] = append(adj[in], out)
				adj[out] = append(adj[out], in)
			}
		}
	}

	return adj
}

func (e *bfsEngine) Causes(ctx context.Context, g graph.Graph, from, to identity.VertexID) (bool, error) {
	if from == to {
		return true, nil
	}
	adj := e.buildAdj(g)
	visited := map[identity.VertexID]bool{from: true}
	queue := []identity.VertexID{from}

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return false, err
		}
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

// Axiom: provCone(G, v) = provClose(G, {v}).
func (e *bfsEngine) Cone(ctx context.Context, g graph.Graph, seed identity.VertexID) ([]identity.VertexID, error) {
	return e.Close(ctx, g, []identity.VertexID{seed})
}

func (e *bfsEngine) Close(ctx context.Context, g graph.Graph, seed []identity.VertexID) ([]identity.VertexID, error) {
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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

func (e *bfsEngine) TraceSet(ctx context.Context, g graph.Graph, from, to identity.VertexID) ([]Trace, error) {
	adj := e.buildAdj(g)
	var traces []Trace
	visited := map[identity.VertexID]bool{from: true}
	path := []identity.VertexID{from}

	var dfs func(cur identity.VertexID) error
	dfs = func(cur identity.VertexID) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if cur == to {
			p := make([]identity.VertexID, len(path))
			copy(p, path)
			traces = append(traces, Trace{Vertices: p})
			return nil
		}
		for _, nb := range adj[cur] {
			if !visited[nb] {
				visited[nb] = true
				path = append(path, nb)
				if err := dfs(nb); err != nil {
					return err
				}
				path = path[:len(path)-1]
				visited[nb] = false
			}
		}
		return nil
	}
	if err := dfs(from); err != nil {
		return nil, err
	}

	return traces, nil
}
