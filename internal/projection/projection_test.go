package projection_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func smallGraph(t *testing.T) (graph.Graph, identity.VertexID, identity.VertexID) {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	a := vid("artifact")
	b := vid("revision")
	g, _ = g.WithVertex(graph.Vertex{ID: a, Type: ontology.Artifact})
	g, _ = g.WithVertex(graph.Vertex{ID: b, Type: ontology.Revision})
	return g, a, b
}

func TestErrInvalidSelectorSentinel(t *testing.T) {
	if !errors.Is(projection.ErrInvalidSelector, projection.ErrInvalidSelector) {
		t.Fatal("sentinel must match itself")
	}
}

// Main path: Select returns a Frontier whose IDs come from the selector.
func TestSelectMainPath(t *testing.T) {
	ctx := context.Background()
	g, a, b := smallGraph(t)

	e := projection.NewEngine()
	f, err := e.Select(ctx, g, projection.IDsSelector{IDs: []identity.VertexID{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	got := f.VertexIDs()
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("Frontier.VertexIDs = %v, want [%v %v]", got, a, b)
	}
}

// Empty selector → empty Frontier (success path).
func TestSelectEmpty(t *testing.T) {
	ctx := context.Background()
	g, _, _ := smallGraph(t)

	e := projection.NewEngine()
	f, err := e.Select(ctx, g, projection.IDsSelector{IDs: nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.VertexIDs()) != 0 {
		t.Fatal("empty selector should produce empty Frontier")
	}
}

// Failure path: selector returns an ID not in the graph.
func TestSelectIDNotInGraph(t *testing.T) {
	ctx := context.Background()
	g, a, _ := smallGraph(t)

	e := projection.NewEngine()
	ghost := vid("ghost")
	_, err := e.Select(ctx, g, projection.IDsSelector{IDs: []identity.VertexID{a, ghost}})
	if !errors.Is(err, projection.ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector, got %v", err)
	}
}

// Failure path: selector itself errors.
type erroringSelector struct{}

func (erroringSelector) Frontier(graph.Graph) ([]identity.VertexID, error) {
	return nil, errors.New("selector exploded")
}

func TestSelectSelectorErrors(t *testing.T) {
	ctx := context.Background()
	g, _, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Select(ctx, g, erroringSelector{})
	if !errors.Is(err, projection.ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector wrap, got %v", err)
	}
}

// Failure path: ctx cancelled before work.
func TestSelectContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, a, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Select(ctx, g, projection.IDsSelector{IDs: []identity.VertexID{a}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// Main path: Project produces a View whose Subgraph contains the requested vertices.
func TestProjectMainPath(t *testing.T) {
	ctx := context.Background()
	g, a, b := smallGraph(t)

	e := projection.NewEngine()
	v, err := e.Project(ctx, g, projection.InduceSpec{IDs: []identity.VertexID{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	sub := v.Subgraph()
	if len(sub.VertexIDs()) != 2 {
		t.Fatalf("expected 2 vertices in view, got %d", len(sub.VertexIDs()))
	}
}

// Failure path: spec apply errors (vertex not in graph).
func TestProjectSpecErrors(t *testing.T) {
	ctx := context.Background()
	g, _, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Project(ctx, g, projection.InduceSpec{IDs: []identity.VertexID{vid("ghost")}})
	if !errors.Is(err, projection.ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector wrap, got %v", err)
	}
}

// Failure path: ctx cancelled.
func TestProjectContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, a, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Project(ctx, g, projection.InduceSpec{IDs: []identity.VertexID{a}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
