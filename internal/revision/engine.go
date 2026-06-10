package revision

import (
	"context"
	"fmt"
	"reflect"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// Strictness controls whether revision.Apply refuses rewrites that
// would silently drop dangling edges. Mirrors the composition bridge's
// Strictness flag.
type Strictness int

const (
	// Lenient is the historical behavior: edges incident to deleted
	// vertices are silently dropped (rebuildWithout skips them).
	Lenient Strictness = iota

	// Strict refuses any rewrite that would leave a host-graph edge
	// orphaned by deletion. Categorically: the pushout complement
	// does not exist when the rule deletes a vertex whose remaining
	// incident edges aren't also in L\K. Returns ErrDanglingEdge.
	Strict
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
//
// Strict mode adds two faithfulness audits the Lenient impl skips:
//   - delete side: dangling-edge detection (the pushout complement must
//     exist) — see ErrDanglingEdge.
//   - produce side: content-addressing check — a produced element's
//     caller-declared ID may not collide with structurally different host
//     content — see ErrIdentityCollision.
type dpoEngine struct {
	strictness Strictness
}

// NewEngine returns a default DPO rewrite engine in Lenient mode.
func NewEngine() Engine {
	return dpoEngine{strictness: Lenient}
}

// NewEngineStrict returns a DPO rewrite engine in Strict mode. Strict
// refuses rewrites that would leave dangling edges, returning
// ErrDanglingEdge. Use this when silent edge-drop is a correctness
// concern.
func NewEngineStrict() Engine {
	return dpoEngine{strictness: Strict}
}

// Strictness returns the configured strictness mode.
func (e dpoEngine) Strictness() Strictness { return e.strictness }

func (e dpoEngine) Apply(ctx context.Context, g graph.Graph, r Rule, m Match) (graph.Graph, ChangeCapsule, error) {
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

	if e.strictness == Strict {
		if danglers := danglingEdges(g, deleteSet, deleteEdges); len(danglers) > 0 {
			return nil, ChangeCapsule{}, fmt.Errorf("%w: %d edge(s)/hyperedge(s) would be orphaned by deletion",
				ErrDanglingEdge, len(danglers))
		}
	}

	newGraph, err := rebuildWithout(g, deleteSet, deleteEdges)
	if err != nil {
		return nil, ChangeCapsule{}, err
	}

	if e.strictness == Strict {
		if cols := contentCollisions(newGraph, right, contextIDs, contextEdges); len(cols) > 0 {
			return nil, ChangeCapsule{}, fmt.Errorf("%w: %d produced element(s) collide with different host content",
				ErrIdentityCollision, len(cols))
		}
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

// Dangler identifies one orphaned edge or hyperedge that would be
// produced by a rewrite. EdgeID is non-zero for edge orphans;
// HyperedgeID is non-zero for hyperedge orphans. Exactly one is set.
type Dangler struct {
	EdgeID      identity.EdgeID
	HyperedgeID identity.HyperedgeID
}

// danglingEdges returns IDs in g that touch a deleted vertex but are
// not themselves slated for deletion. These are the edges/hyperedges
// that Lenient mode would silently drop; Strict mode refuses the
// rewrite when any such item exists. Categorically, the pushout
// complement of l along m does not exist when this set is non-empty.
func danglingEdges(g graph.Graph, deleteVerts map[identity.VertexID]bool, deleteEdges map[identity.EdgeID]bool) []Dangler {
	if len(deleteVerts) == 0 {
		return nil
	}
	var danglers []Dangler
	for _, e := range g.Edges() {
		if deleteEdges[e.ID] {
			continue
		}
		if deleteVerts[e.From] || deleteVerts[e.To] {
			danglers = append(danglers, Dangler{EdgeID: e.ID})
		}
	}
	for _, h := range g.Hyperedges() {
		orphaned := false
		for _, vid := range h.Inputs {
			if deleteVerts[vid] {
				orphaned = true
				break
			}
		}
		if !orphaned {
			for _, vid := range h.Outputs {
				if deleteVerts[vid] {
					orphaned = true
					break
				}
			}
		}
		if orphaned {
			danglers = append(danglers, Dangler{HyperedgeID: h.ID})
		}
	}
	return danglers
}

// Collision identifies one produced (R-side) element whose caller-declared
// ID already binds structurally different content in the post-deletion
// graph. VertexID is non-zero for a vertex collision; EdgeID is non-zero
// for an edge collision. Exactly one is set.
type Collision struct {
	VertexID identity.VertexID
	EdgeID   identity.EdgeID
}

// contentCollisions returns the produced (non-context) vertices and edges in
// `right` whose IDs already exist in `post` (the host graph after deletions)
// bound to structurally different content. These are exactly the insertions
// where graph.WithVertex / graph.WithEdge would silently overwrite an
// existing element with a different value — a content-addressing violation,
// because equal IDs must imply equal content.
//
// Context (K-side) elements are excluded: they are the preserved interface
// and are expected to already exist with matching content. Produced elements
// that re-state existing identical content are not collisions (the rewrite is
// idempotent on them). Genuinely new IDs are not collisions.
//
// Hyperedges are not audited because the current Apply does not insert R-side
// hyperedges (see the insertion loops in Apply); there is no overwrite to
// guard against.
func contentCollisions(post graph.Graph, right graph.Subgraph, contextVerts map[identity.VertexID]bool, contextEdges map[identity.EdgeID]bool) []Collision {
	var collisions []Collision

	for _, v := range right.Vertices() {
		if contextVerts[v.ID] {
			continue
		}
		if existing, ok := post.Vertex(v.ID); ok && !vertexContentEqual(existing, v) {
			collisions = append(collisions, Collision{VertexID: v.ID})
		}
	}

	for _, e := range right.Edges() {
		if contextEdges[e.ID] {
			continue
		}
		if existing, ok := post.Edge(e.ID); ok && !edgeContentEqual(existing, e) {
			collisions = append(collisions, Collision{EdgeID: e.ID})
		}
	}

	return collisions
}

// vertexContentEqual reports whether two vertices with the same ID carry the
// same content: type, temporal triple, trust annotation, and attributes.
func vertexContentEqual(a, b graph.Vertex) bool {
	return a.Type == b.Type &&
		a.Time == b.Time &&
		a.Trust == b.Trust &&
		attrMapEqual(a.Attrs, b.Attrs)
}

// edgeContentEqual reports whether two edges with the same ID carry the same
// content: type, endpoints, and attributes.
func edgeContentEqual(a, b graph.Edge) bool {
	return a.Type == b.Type &&
		a.From == b.From &&
		a.To == b.To &&
		attrMapEqual(a.Attrs, b.Attrs)
}

// attrMapEqual compares two attribute maps by deep equality, treating a nil
// map and an empty map as equal.
func attrMapEqual(a, b graph.AttrMap) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !reflect.DeepEqual(av, bv) {
			return false
		}
	}
	return true
}
