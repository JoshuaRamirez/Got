package provenance_test

import (
	"context"
	"math/rand"
	"sort"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// Property-based coverage for the provenance closure axioms.
//
// The fixture-driven tests in provenance_test.go pin specific graphs. These
// tests generate random causal graphs and assert the closure-operator axioms
// hold across the whole generated population, not just hand-picked shapes.
//
// Generation strategy: {Artifact, DerivedFrom, Artifact} is the one
// admissible homogeneous causal triple (see internal/ontology/schema.go), so a
// random set of DerivedFrom edges over a pool of Artifact vertices builds an
// arbitrary causal topology that the engine treats as an undirected
// reachability graph.

// propIters is the number of random graphs each property test exercises.
const propIters = 300

// randCausalGraph builds a random graph of n Artifact vertices connected by a
// random subset of DerivedFrom edges. It returns the graph and the vertex IDs
// in index order so callers can pick random seed sets.
func randCausalGraph(t *testing.T, r *rand.Rand, n int) (graph.Graph, []identity.VertexID) {
	t.Helper()
	b := graph.NewBuilder(ontology.NewDefaultSchema())

	ids := make([]identity.VertexID, n)
	for i := 0; i < n; i++ {
		ids[i] = vid(string(rune('A' + i)))
		if err := b.AddVertex(graph.Vertex{ID: ids[i], Type: ontology.Artifact}); err != nil {
			t.Fatalf("AddVertex: %v", err)
		}
	}

	// Each unordered pair gets an edge with ~35% probability. DerivedFrom
	// between two Artifacts is admissible in both directions.
	edgeNo := 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if r.Float64() < 0.35 {
				edgeNo++
				e := graph.Edge{
					ID:   eid(string(rune('a'+i)) + string(rune('a'+j))),
					Type: ontology.DerivedFrom,
					From: ids[i],
					To:   ids[j],
				}
				if err := b.AddEdge(e); err != nil {
					t.Fatalf("AddEdge: %v", err)
				}
			}
		}
	}
	return b.Build(), ids
}

// randSubset returns a random non-empty subset of ids.
func randSubset(r *rand.Rand, ids []identity.VertexID) []identity.VertexID {
	var out []identity.VertexID
	for _, id := range ids {
		if r.Float64() < 0.5 {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		out = append(out, ids[r.Intn(len(ids))])
	}
	return out
}

func subset(small, big []identity.VertexID) bool {
	bs := idSet(big)
	for _, id := range small {
		if !bs[id] {
			return false
		}
	}
	return true
}

func sameSet(a, b []identity.VertexID) bool {
	return len(a) == len(b) && subset(a, b) && subset(b, a)
}

// Axiom (extensive): S ⊆ Close(G, S) for every seed set over every graph.
func TestPropCloseExtensive(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(8))
		s := randSubset(r, ids)

		closed, err := e.Close(ctx, g, s)
		if err != nil {
			t.Fatalf("seed %d: Close: %v", seed, err)
		}
		if !subset(s, closed) {
			t.Fatalf("seed %d: extensiveness violated: seed not contained in closure", seed)
		}
	}
}

// Axiom (idempotent): Close(G, Close(G, S)) = Close(G, S) as sets.
func TestPropCloseIdempotent(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(8))
		s := randSubset(r, ids)

		c1, _ := e.Close(ctx, g, s)
		c2, _ := e.Close(ctx, g, c1)
		if !sameSet(c1, c2) {
			t.Fatalf("seed %d: idempotence violated: |Close|=%d |Close(Close)|=%d", seed, len(c1), len(c2))
		}
	}
}

// Axiom (monotone): S1 ⊆ S2 ⇒ Close(G, S1) ⊆ Close(G, S2).
func TestPropCloseMonotone(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(8))
		s1 := randSubset(r, ids)
		// s2 = s1 plus some extra ids — guaranteed superset.
		s2 := append([]identity.VertexID(nil), s1...)
		for _, id := range ids {
			if r.Float64() < 0.4 {
				s2 = append(s2, id)
			}
		}

		c1, _ := e.Close(ctx, g, s1)
		c2, _ := e.Close(ctx, g, s2)
		if !subset(c1, c2) {
			t.Fatalf("seed %d: monotonicity violated", seed)
		}
	}
}

// Axiom: Cone(G, v) = Close(G, {v}).
func TestPropConeEqualsSingletonClose(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(8))
		v := ids[r.Intn(len(ids))]

		cone, _ := e.Cone(ctx, g, v)
		closed, _ := e.Close(ctx, g, []identity.VertexID{v})
		if !sameSet(cone, closed) {
			t.Fatalf("seed %d: Cone != Close singleton", seed)
		}
	}
}

// Axiom (closure is additive over seeds): Close(G, S) = ⋃_{s∈S} Cone(G, s).
func TestPropCloseIsUnionOfCones(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(8))
		s := randSubset(r, ids)

		closed, _ := e.Close(ctx, g, s)

		union := make(map[identity.VertexID]bool)
		for _, id := range s {
			cone, _ := e.Cone(ctx, g, id)
			for _, c := range cone {
				union[c] = true
			}
		}
		got := make([]identity.VertexID, 0, len(union))
		for id := range union {
			got = append(got, id)
		}
		if !sameSet(closed, got) {
			t.Fatalf("seed %d: Close(S) != union of cones: |Close|=%d |union|=%d", seed, len(closed), len(got))
		}
	}
}

// Axiom (Causes is undirected reachability): Causes(a,b) ⟺ b ∈ Close({a}),
// and Causes is symmetric because the engine treats causal edges as undirected.
func TestPropCausesMatchesClosureAndIsSymmetric(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(8))
		a := ids[r.Intn(len(ids))]
		b := ids[r.Intn(len(ids))]

		reach, _ := e.Causes(ctx, g, a, b)
		closed, _ := e.Close(ctx, g, []identity.VertexID{a})
		inClosure := idSet(closed)[b]
		if reach != inClosure {
			t.Fatalf("seed %d: Causes(a,b)=%v but (b ∈ Close({a}))=%v", seed, reach, inClosure)
		}

		rev, _ := e.Causes(ctx, g, b, a)
		if reach != rev {
			t.Fatalf("seed %d: Causes not symmetric: %v vs %v", seed, reach, rev)
		}
	}
}

// Axiom (Causes reflexive): Causes(G, v, v) = true for every vertex.
func TestPropCausesReflexive(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(8))
		v := ids[r.Intn(len(ids))]
		ok, _ := e.Causes(ctx, g, v, v)
		if !ok {
			t.Fatalf("seed %d: Causes(v,v) should be true", seed)
		}
	}
}

// Property: every trace TraceSet returns is a simple path from→to whose
// vertices all lie in Close({from}); and TraceSet is non-empty iff the
// endpoints are causally connected (for distinct endpoints).
func TestPropTraceSetWellFormed(t *testing.T) {
	ctx := context.Background()
	e := newEngine()
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		g, ids := randCausalGraph(t, r, 3+r.Intn(6))
		from := ids[r.Intn(len(ids))]
		to := ids[r.Intn(len(ids))]
		if from == to {
			continue
		}

		traces, err := e.TraceSet(ctx, g, from, to)
		if err != nil {
			t.Fatalf("seed %d: TraceSet: %v", seed, err)
		}

		closed := idSet(mustClose(t, e, g, from))
		for _, tr := range traces {
			vs := tr.Vertices
			if vs[0] != from || vs[len(vs)-1] != to {
				t.Fatalf("seed %d: trace endpoints wrong", seed)
			}
			seen := make(map[identity.VertexID]bool, len(vs))
			for _, v := range vs {
				if seen[v] {
					t.Fatalf("seed %d: trace is not a simple path (repeated vertex)", seed)
				}
				seen[v] = true
				if !closed[v] {
					t.Fatalf("seed %d: trace vertex escapes Close({from})", seed)
				}
			}
		}

		reach, _ := e.Causes(ctx, g, from, to)
		if reach && len(traces) == 0 {
			t.Fatalf("seed %d: endpoints reachable but TraceSet empty", seed)
		}
		if !reach && len(traces) != 0 {
			t.Fatalf("seed %d: endpoints unreachable but TraceSet non-empty", seed)
		}
	}
}

func mustClose(t *testing.T, e interface {
	Close(context.Context, graph.Graph, []identity.VertexID) ([]identity.VertexID, error)
}, g graph.Graph, from identity.VertexID) []identity.VertexID {
	t.Helper()
	c, err := e.Close(context.Background(), g, []identity.VertexID{from})
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	// sort for stable debugging output if a later assertion logs it
	sort.Slice(c, func(i, j int) bool { return string(c[i][:]) < string(c[j][:]) })
	return c
}
