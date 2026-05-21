// Package projection implements the ProjectionSystem specification.
//
// A projection selects a frontier (a subset of vertex IDs) from the graph and
// derives a closed view (subgraph) from it. Projections are idempotent:
// applying the same projection twice yields the same result.
//
// Categorically, a projection query q defines an idempotent endofunctor:
//
//	P_q : Repo_Sigma -> Repo_Sigma    with    P_q . P_q ~= P_q
//
// Imports: internal/graph, internal/identity.
// Must not import: governance, verification, composition, repo.
package projection

import (
	"context"
	"errors"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// ErrInvalidSelector is returned when a Selector cannot be evaluated against
// the given graph.
var ErrInvalidSelector = errors.New("projection: invalid selector")

// Selector chooses a frontier from a graph. A branch selector is a particular
// kind of Selector — not a pointer, but a chosen frontier.
// Categorically, it is a section of the subobject fibration over repositories.
type Selector interface {
	Frontier(g graph.Graph) ([]identity.VertexID, error)
}

// Spec describes a complete projection: selection plus scope boundary.
type Spec interface {
	Apply(g graph.Graph) (graph.Subgraph, error)
}

// Frontier is a selected set of vertex IDs within a graph.
//
// Axiom: frontier(select(G, s)) subset vertexIDs(G).
type Frontier interface {
	VertexIDs() []identity.VertexID
}

// Edited is an optional capability a Frontier may satisfy to carry the
// per-side vertex and edge data that produced it. Used by
// composition.DefaultEngine in Strict mode to detect content-level
// disagreement between left and right frontiers — Textual (Attrs),
// Schema (VertexType), Trust (TrustAnnotation), and Temporal (TimeTriple).
//
// Frontiers that do not satisfy Edited (the default IDsSelector-derived
// frontier among them) still merge correctly under Strict; the per-side
// audits simply do not run.
//
// VertexEdits maps a vertex ID to the Vertex value this frontier "thinks"
// it has, including type, attrs, time, and trust. EdgeEdits maps an edge
// ID to its Edge value. Both maps may be empty; presence is per-id.
type Edited interface {
	Frontier
	VertexEdits() map[identity.VertexID]graph.Vertex
	EdgeEdits() map[identity.EdgeID]graph.Edge
}

// EditedFrontier is a concrete Frontier that carries per-side vertex and
// edge data alongside the vertex ID list. Construct one via
// NewEditedFrontier; the resulting value satisfies both Frontier and
// Edited.
type EditedFrontier struct {
	IDs      []identity.VertexID
	Vertices map[identity.VertexID]graph.Vertex
	Edges    map[identity.EdgeID]graph.Edge
}

// NewEditedFrontier builds an EditedFrontier with empty maps. Callers
// populate Vertices and Edges before passing the frontier to Merge.
func NewEditedFrontier(ids []identity.VertexID) *EditedFrontier {
	return &EditedFrontier{
		IDs:      append([]identity.VertexID(nil), ids...),
		Vertices: make(map[identity.VertexID]graph.Vertex),
		Edges:    make(map[identity.EdgeID]graph.Edge),
	}
}

// VertexIDs satisfies Frontier.
func (f *EditedFrontier) VertexIDs() []identity.VertexID { return f.IDs }

// VertexEdits satisfies Edited.
func (f *EditedFrontier) VertexEdits() map[identity.VertexID]graph.Vertex {
	return f.Vertices
}

// EdgeEdits satisfies Edited.
func (f *EditedFrontier) EdgeEdits() map[identity.EdgeID]graph.Edge {
	return f.Edges
}

// View is the result of applying a projection: a subgraph derived from the
// graph according to a Spec.
type View interface {
	Subgraph() graph.Subgraph
}

// Engine executes selections and projections against a graph.
//
// Axiom: project(graph(project(G, p)), p) = project(G, p) — idempotent.
// Axiom: closedView(G, p) subset provClose(G, vertexIDs(project(G, p))).
type Engine interface {
	// Select evaluates a Selector against the graph and returns the frontier.
	Select(ctx context.Context, g graph.Graph, s Selector) (Frontier, error)

	// Project applies a full projection Spec to the graph and returns a View.
	Project(ctx context.Context, g graph.Graph, s Spec) (View, error)
}
