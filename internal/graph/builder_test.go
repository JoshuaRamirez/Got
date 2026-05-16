package graph_test

import (
	"crypto/sha256"
	"errors"
	"strconv"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

func TestBuilderBuildEmpty(t *testing.T) {
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	g := b.Build()
	if len(g.Vertices()) != 0 || len(g.Edges()) != 0 || len(g.Hyperedges()) != 0 {
		t.Fatal("empty builder should produce empty graph")
	}
}

func TestBuilderBulkInsertEquivalent(t *testing.T) {
	const n = 50
	schema := ontology.NewDefaultSchema()

	// Build via Builder.
	b := graph.NewBuilder(schema)
	ids := make([]identity.VertexID, n)
	for i := 0; i < n; i++ {
		id := identity.VertexID(sha256.Sum256([]byte("v-" + strconv.Itoa(i))))
		ids[i] = id
		if err := b.AddVertex(graph.Vertex{ID: id, Type: ontology.Artifact}); err != nil {
			t.Fatal(err)
		}
	}
	gBuilder := b.Build()

	// Build via repeated WithVertex.
	gWith := graph.NewGraph(schema)
	for _, id := range ids {
		var err error
		gWith, err = gWith.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(gBuilder.Vertices()) != len(gWith.Vertices()) {
		t.Fatalf("vertex count differs: builder=%d with=%d",
			len(gBuilder.Vertices()), len(gWith.Vertices()))
	}
	for _, id := range ids {
		_, okB := gBuilder.Vertex(id)
		_, okW := gWith.Vertex(id)
		if okB != okW {
			t.Fatalf("vertex %x presence differs", id)
		}
	}
}

func TestBuilderEdgeMissingEndpoint(t *testing.T) {
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	err := b.AddEdge(graph.Edge{
		ID:   identity.EdgeID(sha256.Sum256([]byte("e"))),
		Type: ontology.DerivedFrom,
		From: identity.VertexID(sha256.Sum256([]byte("from"))),
		To:   identity.VertexID(sha256.Sum256([]byte("to"))),
	})
	if !errors.Is(err, graph.ErrMissingEndpoint) {
		t.Fatalf("expected ErrMissingEndpoint, got %v", err)
	}
}

func TestBuilderHyperedgeMissingEndpoint(t *testing.T) {
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	err := b.AddHyperedge(graph.Hyperedge{
		ID:      identity.HyperedgeID(sha256.Sum256([]byte("h"))),
		Type:    ontology.Materializes,
		Inputs:  []identity.VertexID{identity.VertexID(sha256.Sum256([]byte("in")))},
		Outputs: []identity.VertexID{},
	})
	if !errors.Is(err, graph.ErrMissingEndpoint) {
		t.Fatalf("expected ErrMissingEndpoint, got %v", err)
	}
}

// Build snapshot independence: mutating the Builder after Build does not
// affect the returned Graph.
func TestBuilderBuildIsSnapshot(t *testing.T) {
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	a := identity.VertexID(sha256.Sum256([]byte("a")))
	b.AddVertex(graph.Vertex{ID: a, Type: ontology.Artifact})

	g1 := b.Build()

	x := identity.VertexID(sha256.Sum256([]byte("x")))
	b.AddVertex(graph.Vertex{ID: x, Type: ontology.Artifact})

	if _, ok := g1.Vertex(x); ok {
		t.Fatal("Build snapshot should not see vertices added after Build")
	}
}

// Validate works on Built graph identically to a freshly constructed one.
func TestBuilderBuildValidates(t *testing.T) {
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	agent := identity.VertexID(sha256.Sum256([]byte("agent")))
	artifact := identity.VertexID(sha256.Sum256([]byte("artifact")))
	b.AddVertex(graph.Vertex{ID: agent, Type: ontology.Agent})
	b.AddVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})
	if err := b.AddEdge(graph.Edge{
		ID: identity.EdgeID(sha256.Sum256([]byte("e1"))), Type: ontology.AuthoredBy,
		From: agent, To: artifact,
	}); err != nil {
		t.Fatal(err)
	}
	if err := b.Build().Validate(); err != nil {
		t.Fatal(err)
	}
}
