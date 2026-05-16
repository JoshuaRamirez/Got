package graph_test

import (
	"crypto/sha256"
	"strconv"
	"sync"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// TestGraphConcurrentReads stresses the immutable Graph's read API
// under concurrent goroutines. Run with `go test -race` to surface any
// hidden data race in mem.go's accessors.
func TestGraphConcurrentReads(t *testing.T) {
	const vcount = 500
	const goroutines = 32
	const reads = 500

	b := graph.NewBuilder(ontology.NewDefaultSchema())
	ids := make([]identity.VertexID, vcount)
	for i := 0; i < vcount; i++ {
		id := identity.VertexID(sha256.Sum256([]byte("concurrent-v-" + strconv.Itoa(i))))
		ids[i] = id
		if err := b.AddVertex(graph.Vertex{ID: id, Type: ontology.Artifact}); err != nil {
			t.Fatal(err)
		}
	}
	g := b.Build()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for w := 0; w < goroutines; w++ {
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < reads; i++ {
				idx := (seed*31 + i) % vcount
				if _, ok := g.Vertex(ids[idx]); !ok {
					t.Errorf("Vertex(%d) returned not-found", idx)
					return
				}
			}
			_ = g.VertexIDs()
			_ = g.Vertices()
		}(w)
	}
	wg.Wait()

	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestGraphConcurrentInduce stresses Induce under concurrent goroutines
// each requesting a different subset.
func TestGraphConcurrentInduce(t *testing.T) {
	const vcount = 300
	const goroutines = 16

	b := graph.NewBuilder(ontology.NewDefaultSchema())
	ids := make([]identity.VertexID, vcount)
	for i := 0; i < vcount; i++ {
		id := identity.VertexID(sha256.Sum256([]byte("induce-v-" + strconv.Itoa(i))))
		ids[i] = id
		b.AddVertex(graph.Vertex{ID: id, Type: ontology.Artifact})
	}
	g := b.Build()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for w := 0; w < goroutines; w++ {
		go func(seed int) {
			defer wg.Done()
			stride := (seed*7 + 1) % vcount
			if stride == 0 {
				stride = 1
			}
			subset := make([]identity.VertexID, 0, vcount/stride)
			for i := 0; i < vcount; i += stride {
				subset = append(subset, ids[i])
			}
			if _, err := g.Induce(subset); err != nil {
				t.Errorf("Induce stride=%d failed: %v", stride, err)
			}
		}(w)
	}
	wg.Wait()
}
