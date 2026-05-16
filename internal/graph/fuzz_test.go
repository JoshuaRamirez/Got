package graph_test

import (
	"crypto/sha256"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// FuzzWithVertexValidate asserts that for any vertex with a known type,
// inserting into an empty graph yields a graph that Validate() passes.
// Catches cases where vertex insertion accidentally produces a graph
// whose edges or hyperedges reference missing endpoints.
func FuzzWithVertexValidate(f *testing.F) {
	for _, seed := range []string{"", "a", "vertex-1", "long-but-not-too-long-id"} {
		f.Add(seed)
	}

	schema := ontology.NewDefaultSchema()

	f.Fuzz(func(t *testing.T, idSeed string) {
		id := identity.VertexID(sha256.Sum256([]byte(idSeed)))
		g := graph.NewGraph(schema)
		g2, err := g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})
		if err != nil {
			t.Fatalf("WithVertex returned error for empty graph: %v", err)
		}
		if err := g2.Validate(); err != nil {
			t.Fatalf("graph with one Artifact vertex failed Validate: %v", err)
		}
		if _, ok := g2.Vertex(id); !ok {
			t.Fatalf("inserted vertex %x not retrievable", id)
		}
	})
}

// FuzzEmptyPreservesSchema asserts that Empty() returns a graph that
// validates and accepts the same kind of vertices the original would.
// If Empty leaked nil or a different schema, this would surface.
func FuzzEmptyPreservesSchema(f *testing.F) {
	f.Add("seed")
	f.Add("")
	schema := ontology.NewDefaultSchema()

	f.Fuzz(func(t *testing.T, idSeed string) {
		g := graph.NewGraph(schema)
		g, _ = g.WithVertex(graph.Vertex{
			ID:   identity.VertexID(sha256.Sum256([]byte(idSeed + "-orig"))),
			Type: ontology.Artifact,
		})

		empty := g.Empty()
		if len(empty.Vertices()) != 0 {
			t.Fatal("Empty graph should have zero vertices")
		}
		// Inserting a new vertex into Empty should still pass Validate.
		nid := identity.VertexID(sha256.Sum256([]byte(idSeed + "-new")))
		empty2, err := empty.WithVertex(graph.Vertex{ID: nid, Type: ontology.Artifact})
		if err != nil {
			t.Fatalf("WithVertex on Empty failed: %v", err)
		}
		if err := empty2.Validate(); err != nil {
			t.Fatalf("Empty-derived graph failed Validate: %v", err)
		}
	})
}
