package repo_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/realization"
	"github.com/joshuaramirez/got/internal/repo"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func eid(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

func TestErrIngestRejectedSentinel(t *testing.T) {
	if !errors.Is(repo.ErrIngestRejected, repo.ErrIngestRejected) {
		t.Fatal("sentinel must match itself")
	}
}

// --- helpers ---

func newService(t *testing.T) *repo.DefaultService {
	t.Helper()
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, nil)
	return repo.NewService(
		composition.NewEngine(gov, ver),
		gov,
		projection.NewEngine(),
		realization.NewEngine(),
		revision.NewEngine(),
		ver,
	)
}

func newState() repo.State {
	g := graph.NewGraph(ontology.NewDefaultSchema())
	ns := namespace.NewStore()
	return repo.NewState(g, ns)
}

type stubPayload struct{ kind string }

func (s stubPayload) PayloadKind() string { return s.kind }

// --- Payload interface smoke test (preserved from interface-only era) ---

func TestPayloadInterface(t *testing.T) {
	var p repo.Payload = stubPayload{kind: "vertex-batch"}
	if p.PayloadKind() != "vertex-batch" {
		t.Fatalf("PayloadKind = %q, want vertex-batch", p.PayloadKind())
	}
}

// --- Ingest ---

func TestIngestVertexPayload(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	v := graph.Vertex{ID: vid("a"), Type: ontology.Artifact}
	out, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{v}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Graph().Vertex(v.ID); !ok {
		t.Fatal("expected ingested vertex to be present in new state")
	}
}

func TestIngestEdgePayload(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	a := graph.Vertex{ID: vid("agent"), Type: ontology.Agent}
	b := graph.Vertex{ID: vid("artifact"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	e := graph.Edge{ID: eid("e1"), Type: ontology.AuthoredBy, From: a.ID, To: b.ID}
	out, err := svc.Ingest(ctx, state, repo.EdgePayload{Edges: []graph.Edge{e}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Graph().Edge(e.ID); !ok {
		t.Fatal("expected ingested edge to be present in new state")
	}
}

func TestIngestNilPayload(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	_, err := svc.Ingest(ctx, state, nil)
	if !errors.Is(err, repo.ErrIngestRejected) {
		t.Fatalf("expected ErrIngestRejected, got %v", err)
	}
}

func TestIngestUnknownPayloadKind(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	_, err := svc.Ingest(ctx, state, stubPayload{kind: "alien"})
	if !errors.Is(err, repo.ErrIngestRejected) {
		t.Fatalf("expected ErrIngestRejected, got %v", err)
	}
}

func TestIngestMissingEndpoint(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	// Edge whose endpoints aren't in the graph.
	e := graph.Edge{ID: eid("e"), Type: ontology.AuthoredBy, From: vid("a"), To: vid("b")}
	_, err := svc.Ingest(ctx, state, repo.EdgePayload{Edges: []graph.Edge{e}})
	if !errors.Is(err, repo.ErrIngestRejected) {
		t.Fatalf("expected ErrIngestRejected, got %v", err)
	}
}

// --- Branch ---

func TestBranchHappyPath(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	v := graph.Vertex{ID: vid("a"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{v}})
	if err != nil {
		t.Fatal(err)
	}
	state, err = svc.Branch(ctx, state, "main", v.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := state.Namespace().ResolveRef(ctx, "main")
	if !ok || got != v.ID {
		t.Fatalf("ResolveRef(main) = (%v, %v), want (%v, true)", got, ok, v.ID)
	}
}

func TestBranchTargetNotFound(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	_, err := svc.Branch(ctx, state, "main", vid("ghost"))
	if !errors.Is(err, graph.ErrVertexNotFound) {
		t.Fatalf("expected graph.ErrVertexNotFound, got %v", err)
	}
}

// --- Revise (smoke through revision engine) ---

type inlineSubgraph struct {
	ids   []identity.VertexID
	verts []graph.Vertex
}

func (s *inlineSubgraph) VertexIDs() []identity.VertexID { return s.ids }
func (s *inlineSubgraph) Vertices() []graph.Vertex       { return s.verts }
func (s *inlineSubgraph) Edges() []graph.Edge            { return nil }
func (s *inlineSubgraph) Hyperedges() []graph.Hyperedge  { return nil }

type addVertexRule struct{ v graph.Vertex }

func (r addVertexRule) Left() graph.Subgraph    { return &inlineSubgraph{} }
func (r addVertexRule) Context() graph.Subgraph { return &inlineSubgraph{} }
func (r addVertexRule) Right() graph.Subgraph {
	return &inlineSubgraph{ids: []identity.VertexID{r.v.ID}, verts: []graph.Vertex{r.v}}
}
func (r addVertexRule) SideConditions() []revision.Predicate { return nil }

type emptyMatch struct{}

func (emptyMatch) Mapping() map[identity.VertexID]identity.VertexID { return nil }

func TestReviseAddVertex(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	v := graph.Vertex{ID: vid("rev"), Type: ontology.Revision}
	out, err := svc.Revise(ctx, state, addVertexRule{v: v}, emptyMatch{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Graph().Vertex(v.ID); !ok {
		t.Fatal("expected revised graph to contain new vertex")
	}
}

// --- Merge ---

func TestMergeHappyPath(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	a := graph.Vertex{ID: vid("a"), Type: ontology.Artifact}
	b := graph.Vertex{ID: vid("b"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a, b}})
	if err != nil {
		t.Fatal(err)
	}

	pe := projection.NewEngine()
	left, err := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{a.ID}})
	if err != nil {
		t.Fatal(err)
	}
	right, err := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{b.ID}})
	if err != nil {
		t.Fatal(err)
	}

	_, mr, err := svc.Merge(ctx, state, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier == nil || len(mr.Conflicts) != 0 {
		t.Fatalf("expected clean merge, got %+v", mr)
	}
}

// --- Evaluate ---

func TestEvaluateHappyPath(t *testing.T) {
	ctx := context.Background()
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, verification.EvaluatorFunc(func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
		return verification.ScalarResult(0.9), nil
	}))
	svc := repo.NewService(
		composition.NewEngine(gov, ver),
		gov,
		projection.NewEngine(),
		realization.NewEngine(),
		revision.NewEngine(),
		ver,
	)
	state := newState()

	pe := projection.NewEngine()
	f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{})

	_, eval, err := svc.Evaluate(ctx, state, f, verification.EnvironmentBinding{ID: vid("env")})
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := eval.Result().(verification.ScalarResult); !ok || s != 0.9 {
		t.Fatalf("Result = %v, want 0.9", eval.Result())
	}
}

// --- Materialize ---

func TestMaterializeHappyPath(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	v := graph.Vertex{ID: vid("a"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{v}})
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := svc.Materialize(ctx, state, projection.InduceSpec{IDs: []identity.VertexID{v.ID}}, realization.ManifestTarget)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Paths()) != 1 {
		t.Fatalf("expected 1 path in bundle, got %d", len(bundle.Paths()))
	}
}

// --- Release ---

type fixedPolicy struct{ d governance.Decision }

func (fixedPolicy) Name() string { return "fixed" }
func (p fixedPolicy) Check(graph.Graph, projection.Frontier) (governance.Decision, []governance.Obligation, error) {
	return p.d, nil, nil
}

func TestReleaseHappyPath(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	pe := projection.NewEngine()
	f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{})
	_, err := svc.Release(ctx, state, f, []governance.Policy{fixedPolicy{d: governance.Sat}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReleaseBlocked(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	pe := projection.NewEngine()
	f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{})
	_, err := svc.Release(ctx, state, f, []governance.Policy{fixedPolicy{d: governance.Unsat}})
	if err == nil {
		t.Fatal("expected error when policy gate fails")
	}
}

// --- ctx cancellation across methods ---

func TestIngestContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := newService(t)
	state := newState()

	_, err := svc.Ingest(ctx, state, repo.VertexPayload{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// ReviseWithCapsule returns the ChangeCapsule alongside the new state.
// UC-U02 step 6 requires the system to emit a capsule recording
// consumed and produced vertex IDs.
func TestReviseWithCapsule(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	v := graph.Vertex{ID: vid("capsule-v"), Type: ontology.Revision}
	out, capsule, err := svc.ReviseWithCapsule(ctx, state, addVertexRule{v: v}, emptyMatch{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Graph().Vertex(v.ID); !ok {
		t.Fatal("revised graph should contain new vertex")
	}
	if len(capsule.Produced) != 1 || capsule.Produced[0] != v.ID {
		t.Fatalf("capsule.Produced = %v, want [%v]", capsule.Produced, v.ID)
	}
	if len(capsule.Consumed) != 0 {
		t.Fatalf("capsule.Consumed should be empty for pure addition, got %v", capsule.Consumed)
	}
}

// Revise (without capsule) still works — delegates to ReviseWithCapsule
// and discards the capsule.
func TestReviseDelegatesToReviseWithCapsule(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	v := graph.Vertex{ID: vid("delegate-v"), Type: ontology.Revision}
	out, err := svc.Revise(ctx, state, addVertexRule{v: v}, emptyMatch{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Graph().Vertex(v.ID); !ok {
		t.Fatal("revised graph should contain new vertex via Revise delegation")
	}
}

// --- UC-U18 facade passthrough ---

// MergeThreeWay routes through the facade to composition's three-way merger:
// a one-sided change is taken automatically and no conflict is raised.
func TestMergeThreeWayFacadeOneSidedChange(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	id := vid("v")
	ancestor := projection.NewEditedFrontier([]identity.VertexID{id})
	ancestor.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"x": "1"}}
	left := projection.NewEditedFrontier([]identity.VertexID{id})
	left.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"x": "2"}} // changed
	right := projection.NewEditedFrontier([]identity.VertexID{id})
	right.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"x": "1"}} // unchanged

	_, mr, err := svc.MergeThreeWay(ctx, state, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(mr.Conflicts))
	}
	if mr.Frontier == nil {
		t.Fatal("expected a merged frontier through the facade")
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if got := ed.Vertices[id].Attrs["x"]; got != "2" {
		t.Fatalf("expected left change x=2 to win, got %v", got)
	}
}

// MergeThreeWay surfaces conflicts (merged-xor-conflicted) through the facade.
func TestMergeThreeWayFacadeConflict(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	id := vid("v")
	mk := func(val string) *projection.EditedFrontier {
		f := projection.NewEditedFrontier([]identity.VertexID{id})
		f.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"x": val}}
		return f
	}
	_, mr, err := svc.MergeThreeWay(ctx, state, mk("1"), mk("2"), mk("3"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("modify/modify conflict must not yield a merged frontier")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Textual {
		t.Fatalf("expected one Textual conflict, got %+v", mr.Conflicts)
	}
}

func TestErrThreeWayUnsupportedSentinel(t *testing.T) {
	if !errors.Is(repo.ErrThreeWayUnsupported, repo.ErrThreeWayUnsupported) {
		t.Fatal("sentinel must match itself")
	}
}
