// Package realization implements the RealizationSystem specification.
//
// Realization materializes a projected view of the graph into a concrete
// bundle for a target format. Each artifact path in the bundle carries a
// provenance witness linking it back to the projected subgraph.
//
// Categorically, for each target t, materialization is a lax functor:
//   Mat_t : Repo_Sigma -> Bund_t
// with a provenance witness map:
//   pi_B : Paths(B) -> Sub(U(R))
// landing inside the projected subgraph.
//
// Imports: internal/projection, internal/graph, internal/identity.
// Must not import: repo.
package realization

import (
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/projection"
)

// Target names the output format or destination for materialization.
type Target string

// FidelityContract describes the guarantees a bundle provides about
// faithfulness to the source projection.
type FidelityContract interface {
	Name() string
}

// Bundle is the materialized output of a projection for a given target.
//
// Axiom: materialize(G, P, T) = ok(B) and path in Paths(B) =>
//
//	Provenance(path) subset vertexIDs(project(G, P)).
type Bundle interface {
	Target() Target
	Paths() []string
	Provenance(path string) []identity.VertexID
	Fidelity() FidelityContract
}

// Engine materializes projected views into target bundles.
type Engine interface {
	Materialize(v projection.View, target Target) (Bundle, error)
}
