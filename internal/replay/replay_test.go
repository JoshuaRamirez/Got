package replay_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/replay"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func TestOutcomeStruct(t *testing.T) {
	o := replay.Outcome{Deterministic: true}
	if !o.Deterministic {
		t.Fatal("Outcome.Deterministic round-trip failed")
	}
	var zero replay.Outcome
	if zero.Deterministic {
		t.Fatal("zero-value Outcome must not be deterministic")
	}
}

func TestErrNonDeterministicSentinel(t *testing.T) {
	if !errors.Is(replay.ErrNonDeterministic, replay.ErrNonDeterministic) {
		t.Fatal("sentinel must match itself")
	}
}

// --- helpers ---

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

// Main path: all vertices present, environment matches → Deterministic.
func TestReplayHappyPath(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	envID := vid("env")
	g := graphWith(t, a)

	rev := revision.NewEngine()
	e := replay.NewEngine(rev)

	capsule := revision.ChangeCapsule{
		Consumed:    []identity.VertexID{a},
		Environment: envID,
	}
	env := verification.EnvironmentBinding{ID: envID}

	out, err := e.Replay(ctx, g, capsule, env)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Deterministic {
		t.Fatal("expected deterministic outcome")
	}
}

// Success: empty environment on capsule is treated as "any".
func TestReplayEmptyEnvironment(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	g := graphWith(t, a)

	rev := revision.NewEngine()
	e := replay.NewEngine(rev)

	capsule := revision.ChangeCapsule{
		Consumed: []identity.VertexID{a},
	}
	env := verification.EnvironmentBinding{ID: vid("env")}

	out, err := e.Replay(ctx, g, capsule, env)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Deterministic {
		t.Fatal("expected deterministic with zero capsule environment")
	}
}

// Failure: capsule consumed vertex not in graph → ErrNoMatch from
// revision.Replayable.
func TestReplayConsumedMissing(t *testing.T) {
	ctx := context.Background()
	g := graphWith(t)

	rev := revision.NewEngine()
	e := replay.NewEngine(rev)

	capsule := revision.ChangeCapsule{
		Consumed: []identity.VertexID{vid("ghost")},
	}
	_, err := e.Replay(ctx, g, capsule, verification.EnvironmentBinding{})
	if !errors.Is(err, revision.ErrNoMatch) {
		t.Fatalf("expected revision.ErrNoMatch wrap, got %v", err)
	}
}

// Failure: environment mismatch → ErrNonDeterministic.
func TestReplayEnvironmentMismatch(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	g := graphWith(t, a)

	rev := revision.NewEngine()
	e := replay.NewEngine(rev)

	capsule := revision.ChangeCapsule{
		Consumed:    []identity.VertexID{a},
		Environment: vid("env-A"),
	}
	env := verification.EnvironmentBinding{ID: vid("env-B")}

	out, err := e.Replay(ctx, g, capsule, env)
	if !errors.Is(err, replay.ErrNonDeterministic) {
		t.Fatalf("expected ErrNonDeterministic, got %v", err)
	}
	if out.Deterministic {
		t.Fatal("Outcome.Deterministic should be false on env mismatch")
	}
}

// Failure: ctx cancelled.
func TestReplayContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g := graphWith(t)

	rev := revision.NewEngine()
	e := replay.NewEngine(rev)
	_, err := e.Replay(ctx, g, revision.ChangeCapsule{}, verification.EnvironmentBinding{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
