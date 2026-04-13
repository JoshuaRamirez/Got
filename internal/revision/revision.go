// Package revision implements the RevisionCalculus specification.
//
// Revision is graph rewriting. A Rule is a decorated double-pushout (DPO)
// rewrite rule with left-hand side (consumed pattern), context (preserved),
// right-hand side (produced pattern), and side conditions. A ChangeCapsule
// records the audit trail of each applied rewrite.
//
// Categorically, a change capsule is:
//   p = (L <-l- K -r-> R, lambda)
// and application is the standard DPO construction when the pushout complement
// exists and side conditions hold.
//
// Imports: internal/graph, internal/identity.
// Must not import: composition, realization, repo.
package revision

import (
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// TransformKind classifies the nature of a graph rewrite.
type TransformKind string

// Match is a morphism from a rule's left-hand side into the host graph
// (categorically, a mono m : L -> G). It provides the injective vertex map
// that the revision Engine uses to locate the consumed pattern inside the
// host graph before applying the DPO rewrite.
type Match interface {
	// Mapping returns the injective vertex map: each key is a vertex ID from
	// the rule's left-hand side (L), and the corresponding value is its image
	// in the host graph (G).
	Mapping() map[identity.VertexID]identity.VertexID
}

// Rule is a DPO rewrite rule with side conditions.
//
// Left() is the consumed pattern (L), Context() is the preserved interface (K),
// and Right() is the produced pattern (R).
type Rule interface {
	Left() graph.Subgraph
	Context() graph.Subgraph
	Right() graph.Subgraph
	SideConditions() []Predicate
}

// Predicate is a side condition that must hold for a rule application to proceed.
type Predicate interface {
	Check(g graph.Graph, m Match) error
}

// ChangeCapsule is the audit record of a single rule application.
// It captures the consumed and produced frontiers along with provenance metadata.
//
// Note: in the algebraic spec, applyRule returns Res[Graph, Error] and the
// capsule is a separate construct. Here we bundle them as co-outputs of Apply,
// making the capsule the canonical replay input for internal/replay.
type ChangeCapsule struct {
	Consumed    []identity.VertexID
	Produced    []identity.VertexID
	Kind        TransformKind
	Actor       identity.VertexID
	Environment identity.VertexID
	Policies    []identity.VertexID
	Metadata    graph.AttrMap
}

// Engine applies rewrite rules to a graph and checks replay feasibility.
//
// Axiom: Apply(G, r, m) = ok(G') => extends(G, G') and sideOK(G, r, m).
// Axiom: Replayable(G, c) => consumed(c) subset vertexIDs(G) and
//
//	produced(c) subset vertexIDs(G).
type Engine interface {
	// Apply executes a DPO rewrite of rule r at match m in graph g.
	// Returns the rewritten graph and the change capsule recording the rewrite.
	Apply(g graph.Graph, r Rule, m Match) (graph.Graph, ChangeCapsule, error)

	// Replayable checks whether the change capsule c can be replayed on graph g
	// (i.e., consumed vertices exist and produced vertices are present).
	Replayable(g graph.Graph, c ChangeCapsule) error
}
