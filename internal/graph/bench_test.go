package graph_test

import (
	"crypto/sha256"
	"strconv"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// buildGraph constructs a graph with n vertices of type Artifact and a
// chain of derived_from edges connecting consecutive vertices. Useful as
// a benchmark fixture.
func buildGraph(n int) (graph.Graph, []identity.VertexID) {
	g := graph.NewGraph(ontology.NewDefaultSchema())
	ids := make([]identity.VertexID, n)
	for i := 0; i < n; i++ {
		id := identity.VertexID(sha256.Sum256([]byte("v-" + strconv.Itoa(i))))
		ids[i] = id
		g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})
	}
	for i := 1; i < n; i++ {
		eid := identity.EdgeID(sha256.Sum256([]byte("e-" + strconv.Itoa(i))))
		g, _ = g.WithEdge(graph.Edge{
			ID:   eid,
			Type: ontology.DerivedFrom,
			From: ids[i],
			To:   ids[i-1],
		})
	}
	return g, ids
}

func BenchmarkWithVertex_1000(b *testing.B) {
	schema := ontology.NewDefaultSchema()
	verts := make([]graph.Vertex, 1000)
	for i := range verts {
		verts[i] = graph.Vertex{
			ID:   identity.VertexID(sha256.Sum256([]byte("v-" + strconv.Itoa(i)))),
			Type: ontology.Artifact,
		}
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		g := graph.NewGraph(schema)
		for _, v := range verts {
			g, _ = g.WithVertex(v)
		}
	}
}

// Same workload via Builder: O(n) instead of O(n²).
func BenchmarkBuilder_1000(b *testing.B) {
	schema := ontology.NewDefaultSchema()
	verts := make([]graph.Vertex, 1000)
	for i := range verts {
		verts[i] = graph.Vertex{
			ID:   identity.VertexID(sha256.Sum256([]byte("v-" + strconv.Itoa(i)))),
			Type: ontology.Artifact,
		}
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		bld := graph.NewBuilder(schema)
		for _, v := range verts {
			bld.AddVertex(v)
		}
		_ = bld.Build()
	}
}

func BenchmarkValidate_1000(b *testing.B) {
	g, _ := buildGraph(1000)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if err := g.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInduce_500of1000(b *testing.B) {
	g, ids := buildGraph(1000)
	subset := ids[:500]
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := g.Induce(subset); err != nil {
			b.Fatal(err)
		}
	}
}
