package capability_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/capability"
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

func TestErrNoEmergenceSentinel(t *testing.T) {
	wrapped := errors.Join(capability.ErrNoEmergence, errors.New("detail"))
	if !errors.Is(wrapped, capability.ErrNoEmergence) {
		t.Fatal("wrapped error must match the sentinel via errors.Is")
	}
}

func TestWitnessStruct(t *testing.T) {
	w := capability.Witness{Name: "merge-fidelity"}
	if w.Name != "merge-fidelity" {
		t.Fatalf("Witness.Name = %q, want merge-fidelity", w.Name)
	}
}

// --- helpers ---

func emptyFrontier(t *testing.T) projection.Frontier {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	pe := projection.NewEngine()
	f, err := pe.Select(context.Background(), g, projection.IDsSelector{})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func nonEmptyFrontier(t *testing.T) (graph.Graph, projection.Frontier) {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: vid("a"), Type: ontology.Artifact})
	pe := projection.NewEngine()
	f, err := pe.Select(context.Background(), g, projection.IDsSelector{IDs: []identity.VertexID{vid("a")}})
	if err != nil {
		t.Fatal(err)
	}
	return g, f
}

type fixedPolicyName string

func (n fixedPolicyName) Name() string { return string(n) }
func (fixedPolicyName) Check(graph.Graph, projection.Frontier) (governance.Decision, []governance.Obligation, error) {
	return governance.Sat, nil, nil
}

type stubCertificate struct{}

func (stubCertificate) Target() projection.Frontier         { return nil }
func (stubCertificate) Evidence() []verification.Evaluation { return nil }
func (stubCertificate) Policies() []string                  { return nil }

// --- behavioral tests ---

// Main path: a predicate fires and returns a witness.
func TestEmergesPredicateFires(t *testing.T) {
	ctx := context.Background()
	g, f := nonEmptyFrontier(t)

	e := capability.NewEngine(capability.CertifiedNonEmpty("review-readiness"))
	ok, w, err := e.Emerges(ctx, g, f, nil, []verification.Certificate{stubCertificate{}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected emergence")
	}
	if w.Name != "review-readiness" {
		t.Fatalf("Witness.Name = %q, want review-readiness", w.Name)
	}
}

// Failure: no predicate fires → false plus ErrNoEmergence.
func TestEmergesNoPredicateFires(t *testing.T) {
	ctx := context.Background()
	f := emptyFrontier(t)
	g := graph.NewGraph(ontology.NewDefaultSchema())

	e := capability.NewEngine(capability.CertifiedNonEmpty("review-readiness"))
	ok, _, err := e.Emerges(ctx, g, f, nil, nil)
	if ok {
		t.Fatal("expected no emergence")
	}
	if !errors.Is(err, capability.ErrNoEmergence) {
		t.Fatalf("expected ErrNoEmergence, got %v", err)
	}
}

// Successful variation: first predicate fires; later predicates are not
// evaluated.
func TestEmergesFirstPredicateWins(t *testing.T) {
	ctx := context.Background()
	g, f := nonEmptyFrontier(t)

	called := 0
	first := capability.PredicateFunc(func(graph.Graph, projection.Frontier, []governance.Policy, []verification.Certificate) (bool, capability.Witness) {
		called++
		return true, capability.Witness{Name: "first"}
	})
	second := capability.PredicateFunc(func(graph.Graph, projection.Frontier, []governance.Policy, []verification.Certificate) (bool, capability.Witness) {
		t.Error("second predicate should not be evaluated")
		return false, capability.Witness{}
	})

	e := capability.NewEngine(first, second)
	_, w, _ := e.Emerges(ctx, g, f, nil, nil)
	if w.Name != "first" {
		t.Fatalf("Witness.Name = %q, want first", w.Name)
	}
	if called != 1 {
		t.Fatalf("first predicate called %d times, want 1", called)
	}
}

// Engine with no registered predicates always returns ErrNoEmergence.
func TestEmergesNoPredicatesRegistered(t *testing.T) {
	ctx := context.Background()
	g, f := nonEmptyFrontier(t)

	e := capability.NewEngine()
	_, _, err := e.Emerges(ctx, g, f, nil, nil)
	if !errors.Is(err, capability.ErrNoEmergence) {
		t.Fatalf("expected ErrNoEmergence, got %v", err)
	}
}

// Register adds a predicate post-construction.
func TestEmergesRegister(t *testing.T) {
	ctx := context.Background()
	g, f := nonEmptyFrontier(t)

	e := capability.NewEngine()
	e.Register(capability.CertifiedNonEmpty("late-bound"))

	ok, w, err := e.Emerges(ctx, g, f, nil, []verification.Certificate{stubCertificate{}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || w.Name != "late-bound" {
		t.Fatalf("expected emergence with name late-bound, got (%v, %q)", ok, w.Name)
	}
}

// AllPoliciesNamed: fires when all required names are present.
func TestAllPoliciesNamedFires(t *testing.T) {
	ctx := context.Background()
	g, f := nonEmptyFrontier(t)

	p1 := fixedPolicyName("security-review")
	p2 := fixedPolicyName("code-review")
	e := capability.NewEngine(capability.AllPoliciesNamed("ready", "security-review", "code-review"))
	ok, w, err := e.Emerges(ctx, g, f, []governance.Policy{p1, p2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || w.Name != "ready" {
		t.Fatalf("expected emergence with name ready, got (%v, %q)", ok, w.Name)
	}
}

// AllPoliciesNamed: does not fire when a required name is missing.
func TestAllPoliciesNamedMissingName(t *testing.T) {
	ctx := context.Background()
	g, f := nonEmptyFrontier(t)

	e := capability.NewEngine(capability.AllPoliciesNamed("ready", "security-review", "code-review"))
	_, _, err := e.Emerges(ctx, g, f, []governance.Policy{fixedPolicyName("security-review")}, nil)
	if !errors.Is(err, capability.ErrNoEmergence) {
		t.Fatalf("expected ErrNoEmergence when a required name is missing, got %v", err)
	}
}

// Failure: ctx cancelled.
func TestEmergesContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, f := nonEmptyFrontier(t)

	e := capability.NewEngine(capability.CertifiedNonEmpty("never-reached"))
	_, _, err := e.Emerges(ctx, g, f, nil, []verification.Certificate{stubCertificate{}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
