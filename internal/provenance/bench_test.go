package provenance_test

import (
	"context"
	"crypto/sha256"
	"strconv"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/provenance"
)

// causalChain builds n vertices linked by derived_from edges: v0 ← v1 ← ... ← v(n-1).
func causalChain(n int) (graph.Graph, []identity.VertexID) {
	g := graph.NewGraph(ontology.NewDefaultSchema())
	ids := make([]identity.VertexID, n)
	for i := 0; i < n; i++ {
		id := identity.VertexID(sha256.Sum256([]byte("prov-v-" + strconv.Itoa(i))))
		ids[i] = id
		g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})
	}
	for i := 1; i < n; i++ {
		eid := identity.EdgeID(sha256.Sum256([]byte("prov-e-" + strconv.Itoa(i))))
		g, _ = g.WithEdge(graph.Edge{
			ID: eid, Type: ontology.DerivedFrom, From: ids[i], To: ids[i-1],
		})
	}
	return g, ids
}

func BenchmarkClose_1000(b *testing.B) {
	ctx := context.Background()
	g, ids := causalChain(1000)
	e := provenance.NewEngine(ontology.CausalEdges)
	seed := []identity.VertexID{ids[len(ids)/2]}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := e.Close(ctx, g, seed); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCauses_endToEnd_1000(b *testing.B) {
	ctx := context.Background()
	g, ids := causalChain(1000)
	e := provenance.NewEngine(ontology.CausalEdges)
	from, to := ids[0], ids[len(ids)-1]
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := e.Causes(ctx, g, from, to); err != nil {
			b.Fatal(err)
		}
	}
}
