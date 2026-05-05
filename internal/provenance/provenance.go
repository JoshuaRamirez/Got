// Package provenance implements the ProvenanceCore specification.
//
// Provenance is a closure operator over causal structure in the graph. It
// computes the transitive causal cone of any vertex and provides trace paths
// between causally related vertices.
//
// Categorically, provenance induces a closure operator on the subobject poset:
//
//	cl^prov_G : Sub(G) -> Sub(G)
//
// satisfying extensivity, monotonicity, and idempotence.
//
// Imports: internal/graph, internal/identity.
// Must not import: revision or any higher orchestration package.
package provenance

import (
	"context"
	"errors"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// ErrUnknownVertex is returned when a referenced vertex is not in the graph.
var ErrUnknownVertex = errors.New("provenance: unknown vertex")

// Trace is a causal path between two vertices in the graph. Per
// docs/design-rules.md it is a struct because it is a pure data holder with no
// alternative implementations.
type Trace struct {
	Vertices []identity.VertexID
}

// Engine computes provenance relationships over a graph.
//
// Axiom: S subset Close(G, S)                             — extensive
// Axiom: S1 subset S2 => Close(G, S1) subset Close(G, S2) — monotone
// Axiom: Close(G, Close(G, S)) = Close(G, S)              — idempotent
// Axiom: Cone(G, v) = Close(G, {v})
type Engine interface {
	// Causes returns true if there is a causal path from 'from' to 'to' in g.
	Causes(ctx context.Context, g graph.Graph, from, to identity.VertexID) (bool, error)

	// Cone returns the provenance cone of the seed vertex: all vertices
	// reachable via causal edges. Equivalent to Close with a singleton seed.
	Cone(ctx context.Context, g graph.Graph, seed identity.VertexID) ([]identity.VertexID, error)

	// Close computes the provenance closure of a set of seed vertices.
	// The result is a superset of seed that is closed under causal reachability.
	Close(ctx context.Context, g graph.Graph, seed []identity.VertexID) ([]identity.VertexID, error)

	// TraceSet returns all distinct causal traces from 'from' to 'to'.
	TraceSet(ctx context.Context, g graph.Graph, from, to identity.VertexID) ([]Trace, error)
}
