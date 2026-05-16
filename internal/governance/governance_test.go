package governance_test

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
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

// Decision values are ordered Unsat < Unknown < Sat so a callers can rely on
// the three-valued comparison.
func TestDecisionOrdering(t *testing.T) {
	if !(governance.Unsat < governance.Unknown && governance.Unknown < governance.Sat) {
		t.Fatalf("decision ordering broken: Unsat=%d Unknown=%d Sat=%d",
			governance.Unsat, governance.Unknown, governance.Sat)
	}
}

func TestObligationStruct(t *testing.T) {
	o := governance.Obligation{Code: "P-001", Detail: "missing reviewer"}
	if o.Code != "P-001" || o.Detail != "missing reviewer" {
		t.Fatal("Obligation field round-trip failed")
	}
}

func TestErrPolicyViolationSentinel(t *testing.T) {
	if !errors.Is(governance.ErrPolicyViolation, governance.ErrPolicyViolation) {
		t.Fatal("sentinel must match itself")
	}
}

// --- helpers ---

type fixedPolicy struct {
	name string
	d    governance.Decision
	obs  []governance.Obligation
	err  error
}

func (p fixedPolicy) Name() string { return p.name }
func (p fixedPolicy) Check(graph.Graph, projection.Frontier) (governance.Decision, []governance.Obligation, error) {
	return p.d, p.obs, p.err
}

func emptyFrontier(t *testing.T) (graph.Graph, projection.Frontier) {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	e := projection.NewEngine()
	f, err := e.Select(context.Background(), g, projection.IDsSelector{})
	if err != nil {
		t.Fatal(err)
	}
	return g, f
}

// --- behavioral tests ---

// Empty policy set: aggregate is Sat, no obligations.
func TestCheckEmptyPolicySet(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	d, obs, err := e.Check(ctx, g, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d != governance.Sat || len(obs) != 0 {
		t.Fatalf("Check empty = (%v, %v), want (Sat, nil)", d, obs)
	}
}

// All Sat → aggregate Sat.
func TestCheckAllSat(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	d, _, _ := e.Check(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Sat},
		fixedPolicy{name: "b", d: governance.Sat},
	})
	if d != governance.Sat {
		t.Fatalf("aggregate = %v, want Sat", d)
	}
}

// Mixed Sat + Unknown → aggregate Unknown.
func TestCheckMixedUnknown(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	d, _, _ := e.Check(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Sat},
		fixedPolicy{name: "b", d: governance.Unknown},
	})
	if d != governance.Unknown {
		t.Fatalf("aggregate = %v, want Unknown", d)
	}
}

// Any Unsat dominates → aggregate Unsat (even if some are Sat or Unknown).
func TestCheckUnsatDominates(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	d, _, _ := e.Check(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Sat},
		fixedPolicy{name: "b", d: governance.Unsat, obs: []governance.Obligation{{Code: "X"}}},
		fixedPolicy{name: "c", d: governance.Unknown},
	})
	if d != governance.Unsat {
		t.Fatalf("aggregate = %v, want Unsat", d)
	}
}

// Obligations are concatenated across policies.
func TestCheckObligationsConcatenated(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	_, obs, _ := e.Check(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Sat, obs: []governance.Obligation{{Code: "A"}}},
		fixedPolicy{name: "b", d: governance.Sat, obs: []governance.Obligation{{Code: "B"}}},
	})
	if len(obs) != 2 || obs[0].Code != "A" || obs[1].Code != "B" {
		t.Fatalf("obligations = %+v, want [{A} {B}]", obs)
	}
}

// Failure: a policy's Check returns an error.
func TestCheckPolicyError(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	_, _, err := e.Check(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "boom", err: errors.New("policy backend offline")},
	})
	if err == nil {
		t.Fatal("expected error from failing policy")
	}
}

// GateRelease: Sat with no obligations → true.
func TestGateReleaseHappyPath(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	ok, obs, err := e.GateRelease(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Sat},
	})
	if err != nil || !ok || len(obs) != 0 {
		t.Fatalf("GateRelease = (%v, %v, %v), want (true, nil, nil)", ok, obs, err)
	}
}

// GateRelease: empty policy set → trivially true.
func TestGateReleaseEmpty(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	ok, _, _ := e.GateRelease(ctx, g, f, nil)
	if !ok {
		t.Fatal("empty policy set should gate-true")
	}
}

// GateRelease: Sat but outstanding obligations → false.
func TestGateReleaseObligationsBlock(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	ok, obs, _ := e.GateRelease(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Sat, obs: []governance.Obligation{{Code: "X"}}},
	})
	if ok || len(obs) != 1 {
		t.Fatalf("GateRelease = (%v, %v), want (false, [{X}])", ok, obs)
	}
}

// GateRelease: Unsat → false with obligations.
func TestGateReleaseUnsat(t *testing.T) {
	ctx := context.Background()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	ok, obs, _ := e.GateRelease(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Unsat, obs: []governance.Obligation{{Code: "Y"}}},
	})
	if ok || len(obs) != 1 {
		t.Fatalf("GateRelease = (%v, %v), want (false, [{Y}])", ok, obs)
	}
}

// Failure: ctx cancelled.
func TestCheckContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, f := emptyFrontier(t)

	e := governance.NewEngine()
	_, _, err := e.Check(ctx, g, f, []governance.Policy{
		fixedPolicy{name: "a", d: governance.Sat},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// Sanity: vid imported to keep imports tidy for future tests.
func TestVidUnused(t *testing.T) {
	_ = vid("anchor")
}
