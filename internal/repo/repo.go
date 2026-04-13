// Package repo implements the RepositoryFacade specification.
//
// It is the single orchestration layer that composes all root modules. A
// repository object R = (G, N, Pi, Gamma) bundles the immutable graph with
// the mutable namespace shell.
//
// Categorically:
//   Repo_Sigma ~= integral_{G in Valid_Sigma} State(G)
// where the fiber State(G) contains namespace bindings, active projections,
// release aliases, and activation state.
//
// Axiom: wellFormed(graphOf(R)) for all R.
// Axiom: revise(R, rule, m) = ok(R') => extends(graphOf(R), graphOf(R')).
//
// Imports: all root internal modules.
package repo

import (
	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/realization"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

// State is the composite repository state: an immutable graph plus a mutable
// namespace shell.
type State interface {
	Graph() graph.Graph
	Namespace() namespace.Store
}

// Service is the top-level repository API. Every operation that mutates the
// repository returns a new State value; the graph component is append-only
// while only the namespace component is mutated in place.
type Service interface {
	// Ingest adds raw payload data to the graph as new vertices.
	Ingest(State, any) (State, error)

	// Revise applies a DPO rewrite rule to the graph.
	Revise(State, revision.Rule, revision.Match) (State, error)

	// Branch creates or updates a named branch pointing at the given vertex.
	Branch(State, namespace.RefName, identity.VertexID) (State, error)

	// Merge computes the guarded pushout of two frontiers under governance.
	Merge(State, projection.Frontier, projection.Frontier, []governance.Policy) (State, composition.MergeResult, error)

	// Evaluate runs an evaluation of a frontier in a given environment.
	Evaluate(State, projection.Frontier, verification.EnvironmentBinding) (State, verification.Evaluation, error)

	// Materialize produces a target bundle from a projected view of the graph.
	Materialize(State, projection.Spec, realization.Target) (realization.Bundle, error)

	// Release gates a frontier for release under the given policies.
	Release(State, projection.Frontier, []governance.Policy) (State, error)
}
