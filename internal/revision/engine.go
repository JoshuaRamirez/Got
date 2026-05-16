package revision

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// dpoEngine implements a Double-Pushout (DPO) rewrite over the hypergraph.
//
// Conventions used here:
//   - The Rule supplies three Subgraphs: Left (L), Context (K), and Right (R).
//     Vertex IDs in K are common to L and R (the preserved interface).
//   - The Match supplies the injective mapping m: L → G for the consumed
//     pattern. Context vertices must be present in m's domain too.
//   - Vertices and edges in R that are not in K are added to G with the IDs
//     declared in R (i.e. R supplies fresh, content-addressed IDs).
//   - Vertices and edges in L that are not in K are deleted from G via the
//     match.
//
// This is a literal "delete what's in L\K, keep what's in K, add what's in
// R\K" interpretation. It does not attempt to compute the pushout complement
// from scratch; the Rule pre-declares the consumed and produced subsets.
type dpoEngine struct{}

// NewEngine returns a default DPO rewrite engine.
func NewEngine() Engine {
	return dpoEngine{}
}

func (dpoEngine) Apply(ctx context.Context, g graph.Graph, r Rule, m Match) (graph.Graph, ChangeCapsule, error) {
	if err := ctx.Err(); err != nil {
		return nil, ChangeCapsule{}, err
	}

	mapping := m.Mapping()
	left := r.Left()
	context_ := r.Context()
	right := r.Right()

	contextIDs := vertexSet(context_.VertexIDs())

	for _, pid := range left.VertexIDs() {
		hid, ok := mapping[pid]
		if !ok {
			return nil, ChangeCapsule{}, fmt.Errorf("%w: pattern vertex %v has no mapping",
				ErrNoMatch, pid)
		}
		if _, ok := g.Vertex(hid); !ok {
			return nil, ChangeCapsule{}, fmt.Errorf("%w: mapped vertex %v not in host graph",
				ErrNoMatch, hid)
		}
	}

	for _, p := range r.SideConditions() {
		if err := p.Check(g, m); err != nil {
			return nil, ChangeCapsule{}, fmt.Errorf("%w: %v", ErrSideConditionFailed, err)
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, ChangeCapsule{}, err
	}

	var consumed []identity.VertexID
	deleteSet := make(map[identity.VertexID]bool)
	for _, pid := range left.VertexIDs() {
		if contextIDs[pid] {
			continue
		}
		hid := mapping[pid]
		deleteSet[hid] = true
		consumed = append(consumed, hid)
	}

	leftEdges := edgeIDSet(left.Edges())
	contextEdges := edgeIDSet(context_.Edges())
	deleteEdges := make(map[identity.EdgeID]bool)
	for eid := range leftEdges {
		if !contextEdges[eid] {
			deleteEdges[eid] = true
		}
	}

	newGraph, err := rebuildWithout(g, deleteSet, deleteEdges)
	if err != nil {
		return nil, ChangeCapsule{}, err
	}

	var produced []identity.VertexID
	contextVerts := contextIDs
	for _, v := range right.Vertices() {
		if contextVerts[v.ID] {
			continue
		}
		newGraph, err = newGraph.WithVertex(v)
		if err != nil {
			return nil, ChangeCapsule{}, fmt.Errorf("revision: insert vertex %v failed: %w", v.ID, err)
		}
		produced = append(produced, v.ID)
	}

	rightEdges := right.Edges()
	for _, e := range rightEdges {
		if contextEdges[e.ID] {
			continue
		}
		newGraph, err = newGraph.WithEdge(e)
		if err != nil {
			return nil, ChangeCapsule{}, fmt.Errorf("revision: insert edge %v failed: %w", e.ID, err)
		}
	}

	if err := newGraph.Validate(); err != nil {
		return nil, ChangeCapsule{}, err
	}

	capsule := ChangeCapsule{
		Consumed: consumed,
		Produced: produced,
	}
	return newGraph, capsule, nil
}

func (dpoEngine) Replayable(ctx context.Context, g graph.Graph, c ChangeCapsule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, id := range c.Consumed {
		if _, ok := g.Vertex(id); !ok {
			return fmt.Errorf("%w: consumed vertex %v not in host graph", ErrNoMatch, id)
		}
	}
	for _, id := range c.Produced {
		if _, ok := g.Vertex(id); !ok {
			return fmt.Errorf("%w: produced vertex %v not in host graph", ErrNoMatch, id)
		}
	}
	return nil
}

// rebuildWithout returns a new graph that excludes the given vertices and
// edges. Edges that touch a deleted vertex are also dropped. The result
// uses the same schema as the input via graph.Graph.Empty.
func rebuildWithout(g graph.Graph, deleteVerts map[identity.VertexID]bool, deleteEdges map[identity.EdgeID]bool) (graph.Graph, error) {
	out := g.Empty()

	for _, v := range g.Vertices() {
		if deleteVerts[v.ID] {
			continue
		}
		var err error
		out, err = out.WithVertex(v)
		if err != nil {
			return nil, err
		}
	}
	for _, e := range g.Edges() {
		if deleteEdges[e.ID] {
			continue
		}
		if deleteVerts[e.From] || deleteVerts[e.To] {
			continue
		}
		var err error
		out, err = out.WithEdge(e)
		if err != nil {
			return nil, err
		}
	}
	for _, h := range g.Hyperedges() {
		touched := false
		for _, id := range h.Inputs {
			if deleteVerts[id] {
				touched = true
				break
			}
		}
		if !touched {
			for _, id := range h.Outputs {
				if deleteVerts[id] {
					touched = true
					break
				}
			}
		}
		if touched {
			continue
		}
		var err error
		out, err = out.WithHyperedge(h)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func vertexSet(ids []identity.VertexID) map[identity.VertexID]bool {
	s := make(map[identity.VertexID]bool, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}

func edgeIDSet(edges []graph.Edge) map[identity.EdgeID]bool {
	s := make(map[identity.EdgeID]bool, len(edges))
	for _, e := range edges {
		s[e.ID] = true
	}
	return s
}
