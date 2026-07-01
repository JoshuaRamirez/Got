// Package repo implements the RepositoryFacade specification.
//
// It is the single orchestration layer that composes all root modules. A
// repository object R = (G, N, Pi, Gamma) bundles the immutable graph with
// the mutable namespace shell. It exposes a Service (not an Engine) because
// it orchestrates all subsystem Engines rather than implementing a single
// domain operation.
//
// Categorically:
//
//	Repo_Sigma ~= integral_{G in Valid_Sigma} State(G)
//
// where the fiber State(G) contains namespace bindings, active projections,
// release aliases, and activation state.
//
// Axiom: wellFormed(graphOf(R)) for all R.
// Axiom: revise(R, rule, m) = ok(R') => extends(graphOf(R), graphOf(R')).
//
// Imports: all root internal modules.
package repo

import (
	"context"
	"errors"

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

// ErrIngestRejected indicates Ingest could not accept the supplied payload.
var ErrIngestRejected = errors.New("repo: ingest rejected")

// ErrThreeWayUnsupported indicates the wired composition engine does not
// implement composition.ThreeWayMerger, so MergeThreeWay cannot run.
var ErrThreeWayUnsupported = errors.New("repo: three-way merge unsupported by composition engine")

// Payload is the typed input to Ingest. Concrete payload types (e.g.
// VertexPayload, EdgePayload, BulkPayload) implement this interface and
// supply their own typed fields. The PayloadKind discriminator mirrors the
// pattern used by graph.Query.
type Payload interface {
	PayloadKind() string
}

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
	Ingest(ctx context.Context, s State, p Payload) (State, error)

	// Revise applies a DPO rewrite rule to the graph.
	Revise(ctx context.Context, s State, r revision.Rule, m revision.Match) (State, error)

	// Branch creates or updates a named branch pointing at the given vertex.
	Branch(ctx context.Context, s State, name namespace.RefName, target identity.VertexID) (State, error)

	// Merge computes the guarded pushout of two frontiers under governance.
	Merge(ctx context.Context, s State, left, right projection.Frontier, ps []governance.Policy) (State, composition.MergeResult, error)

	// MergeThreeWay reconciles two divergent frontiers against a common
	// ancestor (UC-U18). It requires the wired composition engine to
	// satisfy composition.ThreeWayMerger; if it does not, ErrThreeWayUnsupported
	// is returned.
	MergeThreeWay(ctx context.Context, s State, ancestor, left, right projection.Frontier, ps []governance.Policy) (State, composition.MergeResult, error)

	// Evaluate runs an evaluation of a frontier in a given environment.
	Evaluate(ctx context.Context, s State, f projection.Frontier, env verification.EnvironmentBinding) (State, verification.Evaluation, error)

	// Materialize produces a target bundle from a projected view of the graph.
	Materialize(ctx context.Context, s State, spec projection.Spec, target realization.Target) (realization.Bundle, error)

	// Release gates a frontier for release under the given policies.
	Release(ctx context.Context, s State, f projection.Frontier, ps []governance.Policy) (State, error)
}
