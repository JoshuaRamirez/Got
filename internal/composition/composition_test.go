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
