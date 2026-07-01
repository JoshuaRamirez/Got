package composition_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func TestConflictKindValues(t *testing.T) {
	cases := map[composition.ConflictKind]string{
		composition.Textual:    "textual",
		composition.Structural: "structural",
		composition.Schema:     "schema",
		composition.Policy:     "policy",
		composition.Trust:      "trust",
		composition.Capability: "capability",
		composition.Evaluation: "evaluation",
		composition.Temporal:   "temporal",
	}
	for k, want := range cases {
		if string(k) != want {
			t.Errorf("ConflictKind %q has wrong string form %q", want, string(k))
		}
	}
}

func TestMergeWitnessStruct(t *testing.T) {
	id := vid("merge")
	w := composition.MergeWitness{ID: id}
	if w.ID != id {
		t.Fatal("MergeWitness.ID round-trip failed")
	}
}

func TestSentinels(t *testing.T) {
	for _, e := range []error{composition.ErrConflictUnresolvable, composition.ErrNoPushout} {
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}

// --- helpers ---

type fixedPolicy struct {
	name string
	d    governance.Decision
	obs  []governance.Obligation
}

func (p fixedPolicy) Name() string { return p.name }
func (p fixedPolicy) Check(graph.Graph, projection.Frontier) (governance.Decision, []governance.Obligation, error) {
	return p.d, p.obs, nil
}

func newEngines(t *testing.T) (*composition.DefaultEngine, projection.Engine, graph.Graph) {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, nil)
	e := composition.NewEngine(gov, ver)
	return e, projection.NewEngine(), g
}

func makeFrontier(t *testing.T, pe projection.Engine, g graph.Graph, ids ...identity.VertexID) projection.Frontier {
	t.Helper()
	// We bypass projection.IDsSelector's "must be in graph" validation by
	// adding the vertices first.
	for _, id := range ids {
		if _, ok := g.Vertex(id); !ok {
			t.Fatalf("vertex %v not in graph; add it before calling makeFrontier", id)
		}
	}
	f, err := pe.Select(context.Background(), g, projection.IDsSelector{IDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func graphWith(t *testing.T, ids ...identity.VertexID) graph.Graph {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	for _, id := range ids {
		var err error
		g, err = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})
		if err != nil {
			t.Fatal(err)
		}
	}
	return g
}

// --- behavioral tests ---

// Main path: Merge with Sat policies returns a populated MergeResult.
func TestMergeHappyPath(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	b := vid("b")
	g := graphWith(t, a, b)
	e, pe, _ := newEngines(t)
	left := makeFrontier(t, pe, g, a)
	right := makeFrontier(t, pe, g, b)

	mr, err := e.Merge(ctx, g, left, right, []governance.Policy{fixedPolicy{name: "p", d: governance.Sat}})
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier == nil {
		t.Fatal("expected populated Frontier")
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(mr.Conflicts))
	}
	if mr.Certificate == nil {
		t.Fatal("expected certificate when merge succeeds")
	}
	if mr.Witness == (composition.MergeWitness{}) {
		t.Fatal("expected non-zero MergeWitness")
	}
	if len(mr.Frontier.VertexIDs()) != 2 {
		t.Fatalf("merged frontier has %d IDs, want 2", len(mr.Frontier.VertexIDs()))
	}
}

// Successful variation: empty policies → trivially Sat → merge succeeds.
func TestMergeEmptyPolicies(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	g := graphWith(t, a)
	e, pe, _ := newEngines(t)
	left := makeFrontier(t, pe, g, a)
	right := makeFrontier(t, pe, g, a)

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts with empty policy set, got %v", mr.Conflicts)
	}
}

// Failure: Unsat policy → MergeResult with Policy-kind conflict.
func TestMergeUnsatProducesPolicyConflict(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	g := graphWith(t, a)
	e, pe, _ := newEngines(t)
	left := makeFrontier(t, pe, g, a)
	right := makeFrontier(t, pe, g, a)

	mr, err := e.Merge(ctx, g, left, right, []governance.Policy{
		fixedPolicy{name: "blocker", d: governance.Unsat, obs: []governance.Obligation{{Code: "X"}}},
	})
	if err != nil {
		t.Fatalf("Merge should not error on Unsat — got %v", err)
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Policy {
		t.Fatalf("expected one Policy conflict, got %v", mr.Conflicts)
	}
	if mr.Certificate != nil {
		t.Fatal("expected nil Certificate when merge conflicts")
	}
}

// Successful variation: identical frontiers union to themselves.
func TestMergeIdenticalFrontiers(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	g := graphWith(t, a)
	e, pe, _ := newEngines(t)
	f := makeFrontier(t, pe, g, a)

	mr, err := e.Merge(ctx, g, f, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Frontier.VertexIDs()) != 1 {
		t.Fatalf("union of identical frontiers should have 1 ID, got %d", len(mr.Frontier.VertexIDs()))
	}
}

// Failure: ctx cancelled.
func TestMergeContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := vid("a")
	g := graphWith(t, a)
	e, pe, _ := newEngines(t)
	f := makeFrontier(t, pe, g, a)

	_, err := e.Merge(ctx, g, f, f, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// Resolve: apply a no-op resolution and re-merge.
type noopResolution struct{}

func (noopResolution) Apply(g graph.Graph) (graph.Graph, error) { return g, nil }

func TestResolveAppliesResolutionsAndReMerges(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	g := graphWith(t, a)
	e, _, _ := newEngines(t)

	// Set up a MergeResult with a conflict whose boundary is {a}.
	prior := composition.MergeResult{
		Conflicts: []composition.Conflict{policyConflictStub{boundary: []identity.VertexID{a}}},
	}
	mr, err := e.Resolve(ctx, g, prior, []composition.Resolution{noopResolution{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts after resolution, got %v", mr.Conflicts)
	}
}

// Failure: a Resolution returns an error → ErrConflictUnresolvable.
type erroringResolution struct{}

func (erroringResolution) Apply(graph.Graph) (graph.Graph, error) {
	return nil, errors.New("resolution failed")
}

func TestResolveResolutionErrors(t *testing.T) {
	ctx := context.Background()
	g := graphWith(t)
	e, _, _ := newEngines(t)

	_, err := e.Resolve(ctx, g, composition.MergeResult{}, []composition.Resolution{erroringResolution{}})
	if !errors.Is(err, composition.ErrConflictUnresolvable) {
		t.Fatalf("expected ErrConflictUnresolvable, got %v", err)
	}
}

// --- minimal Conflict implementation for tests ---

type policyConflictStub struct {
	boundary []identity.VertexID
}

func (c policyConflictStub) Kind() composition.ConflictKind { return composition.Policy }
func (c policyConflictStub) Boundary() []identity.VertexID  { return c.boundary }

// --- Strict-mode tests ---

func eid(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

func newStrictEngines(t *testing.T) (*composition.DefaultEngine, projection.Engine) {
	t.Helper()
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, nil)
	e := composition.NewEngineStrict(gov, ver)
	return e, projection.NewEngine()
}

// Strictness is reported back via the accessor.
func TestStrictnessAccessor(t *testing.T) {
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, nil)
	if composition.NewEngine(gov, ver).Strictness() != composition.Lenient {
		t.Fatal("NewEngine should default to Lenient")
	}
	if composition.NewEngineStrict(gov, ver).Strictness() != composition.Strict {
		t.Fatal("NewEngineStrict should be Strict")
	}
}

// Strict mode succeeds on a clean merge (no audit findings, gate passes).
func TestStrictMergeCleanHappyPath(t *testing.T) {
	ctx := context.Background()
	a := vid("strict-a")
	b := vid("strict-b")
	g := graphWith(t, a, b)
	e, pe := newStrictEngines(t)
	left := makeFrontier(t, pe, g, a)
	right := makeFrontier(t, pe, g, b)

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("clean strict merge should have no conflicts, got %d: %v", len(mr.Conflicts), mr.Conflicts)
	}
	if mr.Frontier == nil || mr.Certificate == nil {
		t.Fatal("expected populated frontier and certificate on clean strict merge")
	}
}

// permissiveSchema admits any vertex type and any edge type on any pair
// of vertex types. Used by tests that need to construct edge
// configurations the default schema would reject.
type permissiveSchema struct{}

func (permissiveSchema) KnownVertexType(ontology.VertexType) bool { return true }
func (permissiveSchema) KnownEdgeType(ontology.EdgeType) bool     { return true }
func (permissiveSchema) EdgeAllowed(ontology.VertexType, ontology.EdgeType, ontology.VertexType) bool {
	return true
}
func (permissiveSchema) HyperedgeAllowed([]ontology.VertexType, ontology.EdgeType, []ontology.VertexType) bool {
	return true
}

// Strict mode emits a Structural conflict when two distinct edges
// connect the same (from, to) pair with different types. The default
// schema does not admit two edge types on any single (src, dst) pair,
// so this fixture uses a permissive test schema.
func TestStrictMergeDetectsStructuralConflict(t *testing.T) {
	ctx := context.Background()
	agent := vid("strict-agent")
	artifact := vid("strict-artifact")

	g := graph.NewGraph(permissiveSchema{})
	g, _ = g.WithVertex(graph.Vertex{ID: agent, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})
	g, err := g.WithEdge(graph.Edge{ID: eid("e-auth"), Type: ontology.AuthoredBy, From: agent, To: artifact})
	if err != nil {
		t.Fatal(err)
	}
	g, err = g.WithEdge(graph.Edge{ID: eid("e-approve"), Type: ontology.ApprovedBy, From: agent, To: artifact})
	if err != nil {
		t.Fatal(err)
	}

	e, pe := newStrictEngines(t)
	left := makeFrontier(t, pe, g, agent)
	right := makeFrontier(t, pe, g, artifact)
	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) == 0 {
		t.Fatal("expected at least one Structural conflict")
	}
	for _, c := range mr.Conflicts {
		if c.Kind() == composition.Structural {
			return
		}
	}
	t.Fatalf("no Structural conflict found in %v", mr.Conflicts)
}

// Strict mode emits a Temporal conflict for a vertex with a malformed
// TimeTriple (ValidTo > 0 and ValidTo < ValidFrom).
func TestStrictMergeDetectsTemporalConflict(t *testing.T) {
	ctx := context.Background()
	bad := vid("strict-bad-time")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{
		ID:   bad,
		Type: ontology.Artifact,
		Time: graph.TimeTriple{ValidFrom: 500, ValidTo: 100},
	})

	e, pe := newStrictEngines(t)
	f := makeFrontier(t, pe, g, bad)

	mr, err := e.Merge(ctx, g, f, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) == 0 {
		t.Fatal("expected a Temporal conflict for malformed TimeTriple")
	}
	if mr.Conflicts[0].Kind() != composition.Temporal {
		t.Fatalf("expected Temporal conflict, got %v", mr.Conflicts[0].Kind())
	}
	if len(mr.Conflicts[0].Boundary()) != 1 || mr.Conflicts[0].Boundary()[0] != bad {
		t.Fatalf("boundary should be [bad], got %v", mr.Conflicts[0].Boundary())
	}
}

// Strict mode passes through to the governance gate after audits pass.
// A Unsat policy still emits a Policy-kind conflict.
func TestStrictMergeStillRunsPolicyGate(t *testing.T) {
	ctx := context.Background()
	a := vid("strict-policy-a")
	g := graphWith(t, a)
	e, pe := newStrictEngines(t)
	f := makeFrontier(t, pe, g, a)

	mr, err := e.Merge(ctx, g, f, f, []governance.Policy{
		fixedPolicy{name: "blocker", d: governance.Unsat, obs: []governance.Obligation{{Code: "X"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Policy {
		t.Fatalf("expected one Policy conflict, got %v", mr.Conflicts)
	}
}

// --- Per-side audit tests (Edited frontiers) ---

func mkEditedFrontier(t *testing.T, vs ...graph.Vertex) *projection.EditedFrontier {
	t.Helper()
	ids := make([]identity.VertexID, 0, len(vs))
	for _, v := range vs {
		ids = append(ids, v.ID)
	}
	f := projection.NewEditedFrontier(ids)
	for _, v := range vs {
		f.Vertices[v.ID] = v
	}
	return f
}

// Two Edited frontiers disagreeing on the Type of the same vertex →
// Schema conflict.
func TestStrictPerSideSchemaConflict(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-schema")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e, _ := newStrictEngines(t)
	left := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Artifact})
	right := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Revision})

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Schema) {
		t.Fatalf("expected Schema conflict, got %v", mr.Conflicts)
	}
}

// Two Edited frontiers disagreeing on Attrs of the same vertex →
// Textual conflict.
func TestStrictPerSideTextualConflict(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-attrs")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e, _ := newStrictEngines(t)
	left := mkEditedFrontier(t, graph.Vertex{
		ID: id, Type: ontology.Artifact,
		Attrs: graph.AttrMap{"status": "draft"},
	})
	right := mkEditedFrontier(t, graph.Vertex{
		ID: id, Type: ontology.Artifact,
		Attrs: graph.AttrMap{"status": "review"},
	})

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Textual) {
		t.Fatalf("expected Textual conflict, got %v", mr.Conflicts)
	}
}

// Two Edited frontiers disagreeing on Trust → Trust conflict.
func TestStrictPerSideTrustConflict(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-trust")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e, _ := newStrictEngines(t)
	left := mkEditedFrontier(t, graph.Vertex{
		ID: id, Type: ontology.Artifact,
		Trust: graph.TrustAnnotation{Score: 80, Class: "reviewer"},
	})
	right := mkEditedFrontier(t, graph.Vertex{
		ID: id, Type: ontology.Artifact,
		Trust: graph.TrustAnnotation{Score: 20, Class: "external"},
	})

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Trust) {
		t.Fatalf("expected Trust conflict, got %v", mr.Conflicts)
	}
}

// Two Edited frontiers disagreeing on Time → Temporal conflict.
func TestStrictPerSideTemporalConflict(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-time")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e, _ := newStrictEngines(t)
	left := mkEditedFrontier(t, graph.Vertex{
		ID: id, Type: ontology.Artifact,
		Time: graph.TimeTriple{ValidFrom: 100, ValidTo: 200},
	})
	right := mkEditedFrontier(t, graph.Vertex{
		ID: id, Type: ontology.Artifact,
		Time: graph.TimeTriple{ValidFrom: 100, ValidTo: 300},
	})

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Temporal) {
		t.Fatalf("expected Temporal conflict, got %v", mr.Conflicts)
	}
}

// Two Edited frontiers proposing different-typed edges on the same
// (from, to) → Structural conflict from the per-side path.
func TestStrictPerSideStructuralEdgeConflict(t *testing.T) {
	ctx := context.Background()
	a := vid("ps-edge-a")
	b := vid("ps-edge-b")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: a, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: b, Type: ontology.Artifact})

	e, _ := newStrictEngines(t)

	left := projection.NewEditedFrontier([]identity.VertexID{a, b})
	left.Vertices[a] = graph.Vertex{ID: a, Type: ontology.Agent}
	left.Vertices[b] = graph.Vertex{ID: b, Type: ontology.Artifact}
	left.Edges[eid("e-l")] = graph.Edge{ID: eid("e-l"), Type: ontology.AuthoredBy, From: a, To: b}

	right := projection.NewEditedFrontier([]identity.VertexID{a, b})
	right.Vertices[a] = graph.Vertex{ID: a, Type: ontology.Agent}
	right.Vertices[b] = graph.Vertex{ID: b, Type: ontology.Artifact}
	right.Edges[eid("e-r")] = graph.Edge{ID: eid("e-r"), Type: ontology.ApprovedBy, From: a, To: b}

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Structural) {
		t.Fatalf("expected Structural conflict, got %v", mr.Conflicts)
	}
}

// Both Edited frontiers agree on contents → no per-side conflicts.
func TestStrictPerSideAgreement(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-agree")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e, _ := newStrictEngines(t)
	same := graph.Vertex{
		ID: id, Type: ontology.Artifact,
		Attrs: graph.AttrMap{"status": "ok"},
		Time:  graph.TimeTriple{ValidFrom: 100, ValidTo: 200},
		Trust: graph.TrustAnnotation{Score: 50, Class: "trusted"},
	}
	left := mkEditedFrontier(t, same)
	right := mkEditedFrontier(t, same)

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts for identical edits, got %v", mr.Conflicts)
	}
}

// One side Edited, other plain Frontier → per-side audit skipped, only
// in-graph checks run.
func TestStrictPerSideMixed(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-mixed")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e, pe := newStrictEngines(t)
	plain := makeFrontier(t, pe, g, id)
	edited := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Revision})

	mr, err := e.Merge(ctx, g, plain, edited, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Per-side path is gated on BOTH satisfying Edited, so Schema
	// (type disagreement) cannot fire here.
	for _, c := range mr.Conflicts {
		if c.Kind() == composition.Schema {
			t.Fatalf("per-side Schema conflict should not fire when only one side is Edited; got %v", mr.Conflicts)
		}
	}
}

// hasKind reports whether any conflict in cs has the given kind.
func hasKind(cs []composition.Conflict, k composition.ConflictKind) bool {
	for _, c := range cs {
		if c.Kind() == k {
			return true
		}
	}
	return false
}

// Strict mode's audit short-circuits the governance gate: when an audit
// conflict fires, the policy check is skipped (Policy conflict not added).
func TestStrictMergeAuditShortCircuitsGate(t *testing.T) {
	ctx := context.Background()
	bad := vid("strict-shortcircuit")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{
		ID:   bad,
		Type: ontology.Artifact,
		Time: graph.TimeTriple{ValidFrom: 1000, ValidTo: 1},
	})

	e, pe := newStrictEngines(t)
	f := makeFrontier(t, pe, g, bad)

	mr, err := e.Merge(ctx, g, f, f, []governance.Policy{
		fixedPolicy{name: "blocker", d: governance.Unsat},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only the audit conflict; no Policy conflict appended.
	for _, c := range mr.Conflicts {
		if c.Kind() == composition.Policy {
			t.Fatalf("audit failure should short-circuit before gate; got Policy conflict in %v", mr.Conflicts)
		}
	}
	if len(mr.Conflicts) == 0 {
		t.Fatal("expected at least the Temporal audit conflict")
	}
}

// --- Audit capability (UC-S21) ---

func TestAuditorCapability(t *testing.T) {
	var e composition.Engine = func() composition.Engine { e, _, _ := newEngines(t); return e }()
	if _, ok := e.(composition.Auditor); !ok {
		t.Fatal("*DefaultEngine should satisfy composition.Auditor")
	}
}

// Audit flags a malformed TimeTriple (ValidTo < ValidFrom) as a Temporal
// conflict.
func TestAuditDetectsTemporal(t *testing.T) {
	ctx := context.Background()
	e, pe, _ := newEngines(t)
	bad := vid("audit-bad")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: bad, Type: ontology.Artifact, Time: graph.TimeTriple{ValidFrom: 500, ValidTo: 100}})
	f := makeFrontier(t, pe, g, bad)

	conflicts, err := e.Audit(ctx, g, f)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range conflicts {
		if c.Kind() == composition.Temporal {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a Temporal conflict, got %v", conflicts)
	}
}

// Audit passes a well-formed frontier.
func TestAuditClean(t *testing.T) {
	ctx := context.Background()
	e, pe, _ := newEngines(t)
	ok := vid("audit-ok")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: ok, Type: ontology.Artifact})
	f := makeFrontier(t, pe, g, ok)

	conflicts, err := e.Audit(ctx, g, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected clean audit, got %v", conflicts)
	}
}

// --- Capability / Evaluation per-side conflicts (PR C) ---

// Two Evaluation-typed vertices disagreeing on an attr → Evaluation conflict,
// not Textual, and it carries an EvaluationPayload.
func TestStrictPerSideEvaluationConflict(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-eval")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Evaluation})

	e, _ := newStrictEngines(t)
	left := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Evaluation, Attrs: graph.AttrMap{"score": "0.9"}})
	right := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Evaluation, Attrs: graph.AttrMap{"score": "0.4"}})

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Evaluation) {
		t.Fatalf("expected Evaluation conflict, got %v", mr.Conflicts)
	}
	if hasKind(mr.Conflicts, composition.Textual) {
		t.Fatalf("Evaluation-typed disagreement should not also be Textual: %v", mr.Conflicts)
	}
	// Payload is typed.
	for _, c := range mr.Conflicts {
		if c.Kind() != composition.Evaluation {
			continue
		}
		pl, ok := c.(composition.Payloaded)
		if !ok {
			t.Fatal("Evaluation conflict should be Payloaded")
		}
		p, ok := pl.Payload().(composition.EvaluationPayload)
		if !ok {
			t.Fatalf("expected EvaluationPayload, got %T", pl.Payload())
		}
		if p.Key != "score" || p.Left != "0.9" || p.Right != "0.4" {
			t.Fatalf("unexpected EvaluationPayload %+v", p)
		}
	}
}

// Two Capability-typed vertices disagreeing on an attr → Capability conflict.
func TestStrictPerSideCapabilityConflict(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-cap")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Capability})

	e, _ := newStrictEngines(t)
	left := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Capability, Attrs: graph.AttrMap{"status": "emergent"}})
	right := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Capability, Attrs: graph.AttrMap{"status": "absent"}})

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Capability) {
		t.Fatalf("expected Capability conflict, got %v", mr.Conflicts)
	}
	if hasKind(mr.Conflicts, composition.Textual) {
		t.Fatalf("Capability-typed disagreement should not also be Textual: %v", mr.Conflicts)
	}
}

// A non-Evaluation/Capability vertex (Artifact) still yields Textual, so the
// classification is scoped to the two semantic types.
func TestStrictPerSideAttrStaysTextualForArtifact(t *testing.T) {
	ctx := context.Background()
	id := vid("ps-artifact-attr")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e, _ := newStrictEngines(t)
	left := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"k": "a"}})
	right := mkEditedFrontier(t, graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"k": "b"}})

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Textual) {
		t.Fatalf("Artifact attr disagreement should stay Textual, got %v", mr.Conflicts)
	}
}
