package temporal_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/temporal"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func graphWith(t *testing.T, id identity.VertexID, from, to int64) graph.Graph {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{
		ID:   id,
		Type: ontology.Artifact,
		Time: graph.TimeTriple{ValidFrom: from, ValidTo: to},
	})
	return g
}

func TestIntervalStruct(t *testing.T) {
	iv := temporal.Interval{From: 100, To: 200}
	if iv.From != 100 || iv.To != 200 {
		t.Fatal("Interval round-trip failed")
	}
}

func TestErrUnknownVertexSentinel(t *testing.T) {
	if !errors.Is(temporal.ErrUnknownVertex, temporal.ErrUnknownVertex) {
		t.Fatal("sentinel must match itself")
	}
}

// Main path: Validity reads the vertex's TimeTriple.
func TestValidityMainPath(t *testing.T) {
	ctx := context.Background()
	id := vid("a")
	g := graphWith(t, id, 100, 200)

	e := temporal.NewEngine()
	iv, err := e.Validity(ctx, g, id)
	if err != nil {
		t.Fatal(err)
	}
	if iv.From != 100 || iv.To != 200 {
		t.Fatalf("Validity = %+v, want {100 200}", iv)
	}
}

// Failure path: vertex not in graph.
func TestValidityUnknownVertex(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	e := temporal.NewEngine()
	_, err := e.Validity(ctx, g, vid("ghost"))
	if !errors.Is(err, temporal.ErrUnknownVertex) {
		t.Fatalf("expected ErrUnknownVertex, got %v", err)
	}
}

// Failure path: ValidTo < ValidFrom.
func TestValidityMalformed(t *testing.T) {
	ctx := context.Background()
	id := vid("malformed")
	g := graphWith(t, id, 500, 100)

	e := temporal.NewEngine()
	_, err := e.Validity(ctx, g, id)
	if err == nil {
		t.Fatal("expected error for malformed time triple")
	}
}

// Main path: Fresh respects the half-open interval.
func TestFreshHalfOpen(t *testing.T) {
	ctx := context.Background()
	id := vid("a")
	g := graphWith(t, id, 100, 200)

	e := temporal.NewEngine()
	cases := []struct {
		now      int64
		expected bool
	}{
		{99, false},
		{100, true},
		{150, true},
		{199, true},
		{200, false},
		{201, false},
	}
	for _, c := range cases {
		ok, err := e.Fresh(ctx, g, id, c.now)
		if err != nil {
			t.Fatal(err)
		}
		if ok != c.expected {
			t.Errorf("Fresh(now=%d) = %v, want %v", c.now, ok, c.expected)
		}
	}
}

// Fresh: ValidTo == 0 is "indefinite"; only ValidFrom bounds.
func TestFreshIndefinite(t *testing.T) {
	ctx := context.Background()
	id := vid("indef")
	g := graphWith(t, id, 100, 0)

	e := temporal.NewEngine()
	for _, now := range []int64{99, 100, 1e18} {
		ok, err := e.Fresh(ctx, g, id, now)
		if err != nil {
			t.Fatal(err)
		}
		expected := now >= 100
		if ok != expected {
			t.Errorf("Fresh(now=%d) = %v, want %v", now, ok, expected)
		}
	}
}

// Failure path: ctx cancelled.
func TestValidityContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	id := vid("a")
	g := graphWith(t, id, 100, 200)

	e := temporal.NewEngine()
	_, err := e.Validity(ctx, g, id)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
