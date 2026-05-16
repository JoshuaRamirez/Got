package verification_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

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

func eid(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

func TestEnvironmentBindingStruct(t *testing.T) {
	id := vid("env")
	eb := verification.EnvironmentBinding{ID: id, Version: "1.2.3"}
	if eb.ID != id || eb.Version != "1.2.3" {
		t.Fatal("EnvironmentBinding round-trip failed")
	}
}

func TestSentinels(t *testing.T) {
	for _, e := range []error{verification.ErrCertificationFailed, verification.ErrEnvironmentMismatch} {
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}

// --- helpers ---

func emptyFrontier(t *testing.T) (graph.Graph, projection.Frontier) {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	pe := projection.NewEngine()
	f, err := pe.Select(context.Background(), g, projection.IDsSelector{})
	if err != nil {
		t.Fatal(err)
	}
	return g, f
}

type fixedPolicy struct {
	d   governance.Decision
	obs []governance.Obligation
}

func (p fixedPolicy) Name() string { return "fixed" }
func (p fixedPolicy) Check(graph.Graph, projection.Frontier) (governance.Decision, []governance.Obligation, error) {
	return p.d, p.obs, nil
}

type stubClaim struct{ id identity.VertexID }

func (c stubClaim) ID() identity.VertexID { return c.id }

type stubProof struct{ id identity.VertexID }

func (p stubProof) ID() identity.VertexID { return p.id }

// --- behavioral tests ---

// Main path: Evaluate dispatches to the registered evaluator.
func TestEvaluateMainPath(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	eval := verification.EvaluatorFunc(func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
		return verification.ScalarResult(0.95), nil
	})
	e := verification.NewEngine(gov, eval)

	env := verification.EnvironmentBinding{ID: vid("env"), Version: "v1"}
	got, err := e.Evaluate(ctx, g, f, env)
	if err != nil {
		t.Fatal(err)
	}
	if got.Environment() != env {
		t.Fatalf("Evaluation.Environment = %+v, want %+v", got.Environment(), env)
	}
	if s, ok := got.Result().(verification.ScalarResult); !ok || s != 0.95 {
		t.Fatalf("Evaluation.Result = %v, want ScalarResult(0.95)", got.Result())
	}
}

// Failure: no evaluator configured.
func TestEvaluateNoEvaluator(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	e := verification.NewEngine(gov, nil)

	_, err := e.Evaluate(ctx, g, f, verification.EnvironmentBinding{})
	if err == nil {
		t.Fatal("expected error when no evaluator configured")
	}
}

// Failure: evaluator returns an error.
func TestEvaluatorErrors(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	eval := verification.EvaluatorFunc(func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
		return nil, errors.New("evaluator offline")
	})
	e := verification.NewEngine(gov, eval)

	_, err := e.Evaluate(ctx, g, f, verification.EnvironmentBinding{})
	if err == nil {
		t.Fatal("expected wrapped error from failing evaluator")
	}
}

// Main path: Prove returns true when a Proves edge connects proof → claim.
func TestProveMainPath(t *testing.T) {
	ctx := context.Background()
	claimID := vid("claim")
	proofID := vid("proof")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: claimID, Type: ontology.Claim})
	g, _ = g.WithVertex(graph.Vertex{ID: proofID, Type: ontology.Proof})
	g, err := g.WithEdge(graph.Edge{ID: eid("p1"), Type: ontology.Proves, From: proofID, To: claimID})
	if err != nil {
		t.Fatal(err)
	}

	e := verification.NewEngine(governance.NewEngine(), nil)
	ok, err := e.Prove(ctx, g, stubClaim{id: claimID}, stubProof{id: proofID})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Prove to return true for Proves edge")
	}
}

// Main path: no edge → false (refutes-by-absence path).
func TestProveNoEdge(t *testing.T) {
	ctx := context.Background()
	claimID := vid("claim")
	proofID := vid("proof")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: claimID, Type: ontology.Claim})
	g, _ = g.WithVertex(graph.Vertex{ID: proofID, Type: ontology.Proof})

	e := verification.NewEngine(governance.NewEngine(), nil)
	ok, err := e.Prove(ctx, g, stubClaim{id: claimID}, stubProof{id: proofID})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected Prove to return false when no Proves edge exists")
	}
}

// Failure: claim vertex not in graph.
func TestProveClaimNotFound(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	e := verification.NewEngine(governance.NewEngine(), nil)
	_, err := e.Prove(ctx, g, stubClaim{id: vid("ghost")}, stubProof{id: vid("proof")})
	if !errors.Is(err, graph.ErrVertexNotFound) {
		t.Fatalf("expected graph.ErrVertexNotFound, got %v", err)
	}
}

// Main path: Certify with Sat policies and no obligations issues a certificate.
func TestCertifyHappyPath(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	e := verification.NewEngine(gov, nil)

	cert, err := e.Certify(ctx, g, f, nil, []governance.Policy{fixedPolicy{d: governance.Sat}})
	if err != nil {
		t.Fatal(err)
	}
	if cert.Target() != f {
		t.Fatal("Certificate.Target should match input frontier")
	}
	if len(cert.Policies()) != 1 || cert.Policies()[0] != "fixed" {
		t.Fatalf("Certificate.Policies = %v, want [fixed]", cert.Policies())
	}
}

// Failure: Unsat policy → ErrCertificationFailed.
func TestCertifyUnsat(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	e := verification.NewEngine(gov, nil)

	_, err := e.Certify(ctx, g, f, nil, []governance.Policy{fixedPolicy{
		d:   governance.Unsat,
		obs: []governance.Obligation{{Code: "X"}},
	}})
	if !errors.Is(err, verification.ErrCertificationFailed) {
		t.Fatalf("expected ErrCertificationFailed, got %v", err)
	}
}

// Failure: outstanding obligations even with Sat → ErrCertificationFailed.
func TestCertifyOutstandingObligations(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	e := verification.NewEngine(gov, nil)

	_, err := e.Certify(ctx, g, f, nil, []governance.Policy{fixedPolicy{
		d:   governance.Sat,
		obs: []governance.Obligation{{Code: "Y"}},
	}})
	if !errors.Is(err, verification.ErrCertificationFailed) {
		t.Fatalf("expected ErrCertificationFailed when obligations outstanding, got %v", err)
	}
}

// Main path: empty policies → trivial certificate.
func TestCertifyEmptyPolicies(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	e := verification.NewEngine(gov, nil)

	cert, err := e.Certify(ctx, g, f, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cert == nil {
		t.Fatal("expected non-nil certificate for empty policy set")
	}
}

// Main path: ScalarResult comparison.
func TestScalarResultCompare(t *testing.T) {
	cases := []struct {
		a, b verification.ScalarResult
		want int
	}{
		{0.1, 0.2, -1},
		{0.5, 0.5, 0},
		{0.9, 0.4, 1},
	}
	for _, c := range cases {
		if got := c.a.Compare(c.b); got != c.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// WeightedAverageEvaluator combines child evaluators by weighted mean.
func TestWeightedAverageEvaluator(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	score := func(s float64) verification.Evaluator {
		return verification.EvaluatorFunc(func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
			return verification.ScalarResult(s), nil
		})
	}
	w := verification.WeightedAverageEvaluator{
		Children: []verification.WeightedChild{
			{Weight: 1, Evaluator: score(0.8)},
			{Weight: 3, Evaluator: score(0.4)},
		},
	}
	got, err := w.Evaluate(g, f, verification.EnvironmentBinding{})
	if err != nil {
		t.Fatal(err)
	}
	want := verification.ScalarResult((0.8*1 + 0.4*3) / 4)
	if s, _ := got.(verification.ScalarResult); s != want {
		t.Fatalf("weighted average = %v, want %v", s, want)
	}

	// Usable as the evaluator inside an Engine.
	e := verification.NewEngine(governance.NewEngine(), w)
	eval, err := e.Evaluate(ctx, g, f, verification.EnvironmentBinding{})
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := eval.Result().(verification.ScalarResult); s != want {
		t.Fatalf("engine result = %v, want %v", s, want)
	}
}

// Certify failure path UC-S06 2b: aggregate Unknown blocks certification.
func TestCertifyUnknownBlocks(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	e := verification.NewEngine(gov, nil)

	_, err := e.Certify(ctx, g, f, nil, []governance.Policy{fixedPolicy{
		d: governance.Unknown,
	}})
	if !errors.Is(err, verification.ErrCertificationFailed) {
		t.Fatalf("expected ErrCertificationFailed for Unknown decision, got %v", err)
	}
}

// WeightedAverageEvaluator with zero total weight returns ScalarResult(0).
func TestWeightedAverageZeroWeight(t *testing.T) {
	g, f := emptyFrontier(t)
	w := verification.WeightedAverageEvaluator{}
	got, err := w.Evaluate(g, f, verification.EnvironmentBinding{})
	if err != nil {
		t.Fatal(err)
	}
	if got.(verification.ScalarResult) != 0 {
		t.Fatalf("zero-weight average = %v, want 0", got)
	}
}

// Failure: ctx cancelled.
func TestEvaluateContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, f := emptyFrontier(t)

	gov := governance.NewEngine()
	eval := verification.EvaluatorFunc(func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
		return verification.ScalarResult(1.0), nil
	})
	e := verification.NewEngine(gov, eval)

	_, err := e.Evaluate(ctx, g, f, verification.EnvironmentBinding{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
