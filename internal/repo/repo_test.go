package repo_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
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

// --- UC-S21 / Strict-on-Release ---

// ReleaseStrict blocks a frontier with a malformed TimeTriple that plain
// Release would let through.
func TestReleaseStrictBlocksOnTemporal(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	bad := graph.Vertex{ID: vid("rs-bad"), Type: ontology.Artifact, Time: graph.TimeTriple{ValidFrom: 500, ValidTo: 100}}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{bad}})
	if err != nil {
		t.Fatal(err)
	}
	f := projection.NewEditedFrontier([]identity.VertexID{bad.ID})

	// Plain Release lets it through (no audit).
	if _, err := svc.Release(ctx, state, f, nil); err != nil {
		t.Fatalf("plain Release should not audit; got %v", err)
	}
	// ReleaseStrict blocks it.
	if _, err := svc.ReleaseStrict(ctx, state, f, nil); !errors.Is(err, repo.ErrReleaseAudit) {
		t.Fatalf("expected ErrReleaseAudit, got %v", err)
	}
}

// ReleaseStrict allows a clean frontier through the audit and the gate.
func TestReleaseStrictAllowsClean(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	ok := graph.Vertex{ID: vid("rs-ok"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{ok}})
	if err != nil {
		t.Fatal(err)
	}
	f := projection.NewEditedFrontier([]identity.VertexID{ok.ID})
	if _, err := svc.ReleaseStrict(ctx, state, f, nil); err != nil {
		t.Fatalf("clean frontier should pass ReleaseStrict, got %v", err)
	}
}

func TestReleaseAuditSentinels(t *testing.T) {
	for _, e := range []error{repo.ErrReleaseAudit, repo.ErrAuditUnsupported} {
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}

// --- UC-U20: repository directory persistence ---

// SaveState then LoadState round-trips the graph and namespace across a
// simulated restart, driven through the facade.
func TestSaveLoadStateRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	schema := ontology.NewDefaultSchema()

	// Fresh repo: ingest two vertices + an admissible edge, bind a ref.
	svc := newService(t)
	state, err := repo.LoadState(dir, schema)
	if err != nil {
		t.Fatal(err)
	}
	exec := graph.Vertex{ID: vid("p-exec"), Type: ontology.Execution}
	art := graph.Vertex{ID: vid("p-art"), Type: ontology.Artifact}
	state, err = svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{exec, art}})
	if err != nil {
		t.Fatal(err)
	}
	state, err = svc.Ingest(ctx, state, repo.EdgePayload{Edges: []graph.Edge{
		{ID: eid("p-e"), Type: ontology.Materializes, From: exec.ID, To: art.ID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// Bind a ref through the facade (namespace flushes on its own).
	if _, err := svc.Branch(ctx, state, "main", art.ID); err != nil {
		t.Fatal(err)
	}
	// Persist the graph value.
	if err := repo.SaveState(dir, state); err != nil {
		t.Fatal(err)
	}

	// "Restart": load a fresh State from the same directory.
	reloaded, err := repo.LoadState(dir, schema)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reloaded.Graph().Vertex(exec.ID); !ok {
		t.Fatal("exec vertex did not survive save/load")
	}
	if _, ok := reloaded.Graph().Edge(eid("p-e")); !ok {
		t.Fatal("edge did not survive save/load")
	}
	if got, ok := reloaded.Namespace().ResolveRef(ctx, "main"); !ok || got != art.ID {
		t.Fatalf("ref 'main' did not survive save/load: got %v ok=%v", got, ok)
	}
}

// LoadState on an empty directory yields an empty, usable repository.
func TestLoadStateEmptyDir(t *testing.T) {
	state, err := repo.LoadState(t.TempDir(), ontology.NewDefaultSchema())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Graph().VertexIDs()) != 0 {
		t.Fatal("empty dir should load an empty graph")
	}
	if _, ok := state.Namespace().ResolveRef(context.Background(), "x"); ok {
		t.Fatal("empty dir should have no bindings")
	}
}

// A later SaveState overwrites the graph file so LoadState sees the newest
// graph value.
func TestSaveStateOverwritesGraph(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	schema := ontology.NewDefaultSchema()
	svc := newService(t)

	state, _ := repo.LoadState(dir, schema)
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{{ID: vid("v1"), Type: ontology.Artifact}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveState(dir, state); err != nil {
		t.Fatal(err)
	}
	state, err = svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{{ID: vid("v2"), Type: ontology.Artifact}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveState(dir, state); err != nil {
		t.Fatal(err)
	}

	reloaded, _ := repo.LoadState(dir, schema)
	if len(reloaded.Graph().VertexIDs()) != 2 {
		t.Fatalf("expected 2 vertices after second save, got %d", len(reloaded.Graph().VertexIDs()))
	}
}

// A corrupt graph file is rejected on load.
func TestLoadStateCorruptGraph(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "graph.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.LoadState(dir, ontology.NewDefaultSchema()); err == nil {
		t.Fatal("expected error loading corrupt graph file")
	}
}

// --- UC-U21: first-class branches ---

func TestCreateBranchAndList(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	state, main, err := svc.CreateBranch(ctx, state, "main", "", identity.VertexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if main.Name != "main" || main.Parent != "" {
		t.Fatalf("unexpected main branch: %+v", main)
	}
	state, feat, err := svc.CreateBranch(ctx, state, "feature", "main", identity.VertexID{}, map[string]string{"desc": "new work"})
	if err != nil {
		t.Fatal(err)
	}
	if feat.Parent != "main" || feat.Attrs["desc"] != "new work" {
		t.Fatalf("unexpected feature branch: %+v", feat)
	}

	branches, err := svc.Branches(ctx, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}
}

func TestBranchLineage(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	state, _, _ = svc.CreateBranch(ctx, state, "main", "", identity.VertexID{}, nil)
	state, _, _ = svc.CreateBranch(ctx, state, "release", "main", identity.VertexID{}, nil)
	state, _, err := svc.CreateBranch(ctx, state, "feature", "release", identity.VertexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	lineage, err := svc.BranchLineage(ctx, state, "feature")
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(lineage))
	for i, b := range lineage {
		got[i] = b.Name
	}
	want := []string{"feature", "release", "main"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("lineage = %v, want %v", got, want)
	}
}

func TestCreateBranchDuplicate(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()
	state, _, _ = svc.CreateBranch(ctx, state, "main", "", identity.VertexID{}, nil)
	if _, _, err := svc.CreateBranch(ctx, state, "main", "", identity.VertexID{}, nil); !errors.Is(err, repo.ErrBranchExists) {
		t.Fatalf("expected ErrBranchExists, got %v", err)
	}
}

func TestCreateBranchUnknownParent(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()
	if _, _, err := svc.CreateBranch(ctx, state, "x", "ghost", identity.VertexID{}, nil); !errors.Is(err, repo.ErrUnknownBranch) {
		t.Fatalf("expected ErrUnknownBranch, got %v", err)
	}
}

func TestCreateBranchBindsTip(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	state := newState()

	// A vertex to point the tip at.
	tip := graph.Vertex{ID: vid("tip-art"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{tip}})
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = svc.CreateBranch(ctx, state, "main", "", tip.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := state.Namespace().ResolveRef(ctx, "main"); !ok || got != tip.ID {
		t.Fatalf("branch tip not bound: got %v ok=%v", got, ok)
	}
}

func TestBranchSentinels(t *testing.T) {
	for _, e := range []error{repo.ErrBranchExists, repo.ErrUnknownBranch} {
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}
