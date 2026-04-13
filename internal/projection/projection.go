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
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

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
	Select(g graph.Graph, s Selector) (Frontier, error)

	// Project applies a full projection Spec to the graph and returns a View.
	Project(g graph.Graph, s Spec) (View, error)
}
