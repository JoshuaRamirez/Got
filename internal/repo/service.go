package repo

import (
	"context"
	"fmt"

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

// DefaultState is the canonical State: an immutable graph + a mutable
// namespace store.
type DefaultState struct {
	graph graph.Graph
	ns    namespace.Store
}

// NewState bundles a graph and namespace into a State.
func NewState(g graph.Graph, ns namespace.Store) *DefaultState {
	return &DefaultState{graph: g, ns: ns}
}

// Graph satisfies State.
func (s *DefaultState) Graph() graph.Graph { return s.graph }

// Namespace satisfies State.
func (s *DefaultState) Namespace() namespace.Store { return s.ns }

// withGraph returns a new DefaultState with the same namespace but the
// supplied graph value. The namespace is shared (it's the mutable shell).
func (s *DefaultState) withGraph(g graph.Graph) *DefaultState {
	return &DefaultState{graph: g, ns: s.ns}
}

// DefaultService orchestrates every Engine and Service exposed by
// internal/. Construct one with NewService, then call its methods to
// drive end-to-end operations on a State.
type DefaultService struct {
	composition  composition.Engine
	governance   governance.Engine
	projection   projection.Engine
	realization  realization.Engine
	revision     revision.Engine
	verification verification.Engine
}

// NewService wires the supplied engines into a DefaultService. All seven
// engines are required.
func NewService(
	comp composition.Engine,
	gov governance.Engine,
	proj projection.Engine,
	real realization.Engine,
	rev revision.Engine,
	ver verification.Engine,
) *DefaultService {
	return &DefaultService{
		composition:  comp,
		governance:   gov,
		projection:   proj,
		realization:  real,
		revision:     rev,
		verification: ver,
	}
}

// --- Payloads ---

// VertexPayload adds vertices to the graph.
type VertexPayload struct {
	Vertices []graph.Vertex
}

// PayloadKind identifies VertexPayload.
func (VertexPayload) PayloadKind() string { return "vertex" }

// EdgePayload adds edges to the graph.
type EdgePayload struct {
	Edges []graph.Edge
}

// PayloadKind identifies EdgePayload.
func (EdgePayload) PayloadKind() string { return "edge" }

// --- Service methods ---

// Ingest dispatches by payload kind and extends the graph.
func (s *DefaultService) Ingest(ctx context.Context, state State, p Payload) (State, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("%w: nil payload", ErrIngestRejected)
	}
	g := state.Graph()

	switch v := p.(type) {
	case VertexPayload:
		for _, vert := range v.Vertices {
			var err error
			g, err = g.WithVertex(vert)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrIngestRejected, err)
			}
		}
	case EdgePayload:
		for _, e := range v.Edges {
			var err error
			g, err = g.WithEdge(e)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrIngestRejected, err)
			}
		}
	default:
		return nil, fmt.Errorf("%w: unknown payload kind %q", ErrIngestRejected, p.PayloadKind())
	}

	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIngestRejected, err)
	}
	return defaultStateWith(state, g), nil
}

// Revise applies a DPO rewrite to the graph and discards the audit
// capsule. Callers that need the ChangeCapsule should use
// ReviseWithCapsule instead.
func (s *DefaultService) Revise(ctx context.Context, state State, r revision.Rule, m revision.Match) (State, error) {
	newState, _, err := s.ReviseWithCapsule(ctx, state, r, m)
	return newState, err
}

// ReviseWithCapsule applies a DPO rewrite and returns the audit
// ChangeCapsule alongside the new state. Per UC-U02 step 6, every
// revision emits a capsule recording consumed and produced vertex IDs
// plus actor/environment/policies metadata. Replay (UC-U14) consumes
// the capsule.
//
// Revise (which discards the capsule) remains for callers that don't
// need the audit trail; it delegates to this method.
func (s *DefaultService) ReviseWithCapsule(ctx context.Context, state State, r revision.Rule, m revision.Match) (State, revision.ChangeCapsule, error) {
	g := state.Graph()
	newG, capsule, err := s.revision.Apply(ctx, g, r, m)
	if err != nil {
		return nil, revision.ChangeCapsule{}, err
	}
	return defaultStateWith(state, newG), capsule, nil
}

// Branch binds a name to a vertex.
func (s *DefaultService) Branch(ctx context.Context, state State, name namespace.RefName, target identity.VertexID) (State, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, ok := state.Graph().Vertex(target); !ok {
		return nil, fmt.Errorf("%w: %v", graph.ErrVertexNotFound, target)
	}
	if err := state.Namespace().BindRef(ctx, name, target); err != nil {
		return nil, err
	}
	return state, nil
}

// Merge computes the guarded pushout.
func (s *DefaultService) Merge(ctx context.Context, state State, left, right projection.Frontier, ps []governance.Policy) (State, composition.MergeResult, error) {
	mr, err := s.composition.Merge(ctx, state.Graph(), left, right, ps)
	if err != nil {
		return nil, composition.MergeResult{}, err
	}
	return state, mr, nil
}

// MergeThreeWay reconciles two divergent frontiers against a common
// ancestor by delegating to the composition engine's three-way capability
// (UC-U18). The wired composition.Engine must satisfy
// composition.ThreeWayMerger — the default DefaultEngine does. Returns
// ErrThreeWayUnsupported otherwise.
func (s *DefaultService) MergeThreeWay(ctx context.Context, state State, ancestor, left, right projection.Frontier, ps []governance.Policy) (State, composition.MergeResult, error) {
	tw, ok := s.composition.(composition.ThreeWayMerger)
	if !ok {
		return nil, composition.MergeResult{}, ErrThreeWayUnsupported
	}
	mr, err := tw.MergeThreeWay(ctx, state.Graph(), ancestor, left, right, ps)
	if err != nil {
		return nil, composition.MergeResult{}, err
	}
	return state, mr, nil
}

// Evaluate runs an evaluation against a frontier.
func (s *DefaultService) Evaluate(ctx context.Context, state State, f projection.Frontier, env verification.EnvironmentBinding) (State, verification.Evaluation, error) {
	eval, err := s.verification.Evaluate(ctx, state.Graph(), f, env)
	if err != nil {
		return nil, nil, err
	}
	return state, eval, nil
}

// Materialize projects and then materializes for a target.
func (s *DefaultService) Materialize(ctx context.Context, state State, spec projection.Spec, target realization.Target) (realization.Bundle, error) {
	view, err := s.projection.Project(ctx, state.Graph(), spec)
	if err != nil {
		return nil, err
	}
	return s.realization.Materialize(ctx, view, target)
}

// Release gates a frontier for release.
func (s *DefaultService) Release(ctx context.Context, state State, f projection.Frontier, ps []governance.Policy) (State, error) {
	ok, obligations, err := s.governance.GateRelease(ctx, state.Graph(), f, ps)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("repo: release blocked: %d obligation(s) unmet", len(obligations))
	}
	return state, nil
}

// defaultStateWith returns a State whose graph is replaced with g and
// whose namespace is preserved from the input state. If state is the
// concrete *DefaultState, the helper preserves that type; otherwise it
// wraps the namespace from state with the new graph in a fresh
// *DefaultState.
func defaultStateWith(state State, g graph.Graph) State {
	if ds, ok := state.(*DefaultState); ok {
		return ds.withGraph(g)
	}
	return &DefaultState{graph: g, ns: state.Namespace()}
}
