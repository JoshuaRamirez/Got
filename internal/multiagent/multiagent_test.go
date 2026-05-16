package multiagent_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/multiagent"
	"github.com/joshuaramirez/got/internal/ontology"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func eid(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

func TestResponsibilityStruct(t *testing.T) {
	a := vid("agent")
	r := multiagent.Responsibility{Path: []identity.VertexID{a}}
	if len(r.Path) != 1 || r.Path[0] != a {
		t.Fatal("Responsibility.Path round-trip failed")
	}
}

func TestErrNoAuthorshipSentinel(t *testing.T) {
	if !errors.Is(multiagent.ErrNoAuthorship, multiagent.ErrNoAuthorship) {
		t.Fatal("sentinel must match itself")
	}
}

// Main path: Authorship returns the agent that authored an artifact.
func TestAuthorshipMainPath(t *testing.T) {
	ctx := context.Background()
	agent := vid("alice")
	artifact := vid("doc")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: agent, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})
	g, err := g.WithEdge(graph.Edge{
		ID: eid("e1"), Type: ontology.AuthoredBy, From: agent, To: artifact,
	})
	if err != nil {
		t.Fatal(err)
	}

	e := multiagent.NewDefaultEngine()
	authors, err := e.Authorship(ctx, g, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(authors) != 1 || authors[0] != agent {
		t.Fatalf("Authorship = %v, want [%v]", authors, agent)
	}
}

// Main path: multiple authors return all of them.
func TestAuthorshipMultiple(t *testing.T) {
	ctx := context.Background()
	a1 := vid("alice")
	a2 := vid("bob")
	artifact := vid("doc")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: a1, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: a2, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})
	g, _ = g.WithEdge(graph.Edge{ID: eid("e1"), Type: ontology.AuthoredBy, From: a1, To: artifact})
	g, _ = g.WithEdge(graph.Edge{ID: eid("e2"), Type: ontology.AuthoredBy, From: a2, To: artifact})

	e := multiagent.NewDefaultEngine()
	authors, err := e.Authorship(ctx, g, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(authors) != 2 {
		t.Fatalf("expected 2 authors, got %d", len(authors))
	}
}

// Success path: no authorship edges → empty slice, no error.
func TestAuthorshipNoEdges(t *testing.T) {
	ctx := context.Background()
	artifact := vid("doc")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})

	e := multiagent.NewDefaultEngine()
	authors, err := e.Authorship(ctx, g, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(authors) != 0 {
		t.Fatalf("expected zero authors, got %v", authors)
	}
}

// Failure path: target vertex not in graph.
func TestAuthorshipVertexNotFound(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	e := multiagent.NewDefaultEngine()
	_, err := e.Authorship(ctx, g, vid("ghost"))
	if !errors.Is(err, graph.ErrVertexNotFound) {
		t.Fatalf("expected graph.ErrVertexNotFound, got %v", err)
	}
}

// Main path: ResponsibilityTrace walks the chain.
func TestResponsibilityTraceMainPath(t *testing.T) {
	ctx := context.Background()
	human := vid("human")
	agent := vid("alice")
	artifact := vid("doc")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: human, Type: ontology.Human})
	g, _ = g.WithVertex(graph.Vertex{ID: agent, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})
	g, _ = g.WithEdge(graph.Edge{ID: eid("e1"), Type: ontology.AuthoredBy, From: agent, To: artifact})
	g, _ = g.WithEdge(graph.Edge{ID: eid("e2"), Type: ontology.ApprovedBy, From: human, To: artifact})

	e := multiagent.NewDefaultEngine()
	resp, err := e.ResponsibilityTrace(ctx, g, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Path) != 2 {
		t.Fatalf("expected 2 in path, got %d: %v", len(resp.Path), resp.Path)
	}
}

// Failure path: target has no authorship edges → ErrNoAuthorship.
func TestResponsibilityTraceNoAuthorship(t *testing.T) {
	ctx := context.Background()
	artifact := vid("doc")

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})

	e := multiagent.NewDefaultEngine()
	_, err := e.ResponsibilityTrace(ctx, g, artifact)
	if !errors.Is(err, multiagent.ErrNoAuthorship) {
		t.Fatalf("expected ErrNoAuthorship, got %v", err)
	}
}

// Failure path: ctx cancelled.
func TestAuthorshipContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: vid("a"), Type: ontology.Artifact})

	e := multiagent.NewDefaultEngine()
	_, err := e.Authorship(ctx, g, vid("a"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
