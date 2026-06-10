package composition_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
)

// Property-based coverage for the composition merge axioms declared on
// MergeResult (composition.go):
//
//	Axiom: (merged != none) xor (conflicts != {}).
//	Lenient merge frontier = set-union of the two input frontiers.
//
// The fixture tests pin specific frontiers; these generate random frontier
// pairs over a pool of Artifact vertices and assert the set-union laws
// (correctness, commutativity, idempotence), the XOR invariant, and witness
// determinism hold across the population.

const propIters = 300

// vertexPool returns n distinct Artifact vertex IDs and a graph containing
// them. The pool is shared across a single property iteration so frontiers
// drawn from it always reference present vertices.
func vertexPool(t *testing.T, n int) ([]identity.VertexID, graph.Graph) {
	t.Helper()
	ids := make([]identity.VertexID, n)
	g := graph.NewGraph(ontology.NewDefaultSchema())
	for i := 0; i < n; i++ {
		ids[i] = vid("pool-" + string(rune('A'+i)))
		var err error
		g, err = g.WithVertex(graph.Vertex{ID: ids[i], Type: ontology.Artifact})
		if err != nil {
			t.Fatal(err)
		}
	}
	return ids, g
}

// randFrontier selects a random non-empty subset of pool as a Frontier.
func randFrontier(t *testing.T, pe projection.Engine, g graph.Graph, r *rand.Rand, pool []identity.VertexID) projection.Frontier {
	t.Helper()
	var ids []identity.VertexID
	for _, id := range pool {
		if r.Float64() < 0.5 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		ids = append(ids, pool[r.Intn(len(pool))])
	}
	return makeFrontier(t, pe, g, ids...)
}

func frontierSet(f projection.Frontier) map[identity.VertexID]bool {
	m := make(map[identity.VertexID]bool)
	for _, id := range f.VertexIDs() {
		m[id] = true
	}
	return m
}

func idSliceSet(ids []identity.VertexID) map[identity.VertexID]bool {
	m := make(map[identity.VertexID]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func setsEqual(a, b map[identity.VertexID]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// Property: Lenient merge frontier = set-union of inputs; result satisfies the
// XOR invariant (merged present, no conflicts) under a trivially-Sat gate.
func TestPropMergeIsSetUnion(t *testing.T) {
	ctx := context.Background()
	e, pe, _ := newEngines(t)
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		pool, g := vertexPool(t, 2+r.Intn(7))
		left := randFrontier(t, pe, g, r, pool)
		right := randFrontier(t, pe, g, r, pool)

		mr, err := e.Merge(ctx, g, left, right, nil)
		if err != nil {
			t.Fatalf("seed %d: Merge: %v", seed, err)
		}

		// XOR invariant: with no policies the gate is trivially Sat, so the
		// merge must succeed and carry no conflicts.
		if mr.Frontier == nil {
			t.Fatalf("seed %d: expected merged frontier under empty policy set", seed)
		}
		if len(mr.Conflicts) != 0 {
			t.Fatalf("seed %d: XOR invariant violated: merged AND conflicted", seed)
		}

		want := make(map[identity.VertexID]bool)
		for k := range frontierSet(left) {
			want[k] = true
		}
		for k := range frontierSet(right) {
			want[k] = true
		}
		if !setsEqual(frontierSet(mr.Frontier), want) {
			t.Fatalf("seed %d: merged frontier is not the set-union of inputs", seed)
		}
	}
}

// Property (commutativity of the frontier set): merging L,R and R,L yields the
// same vertex-ID set. (The witness ID is order-sensitive by design and is NOT
// asserted equal here — see TestPropWitnessOrderSensitivity.)
func TestPropMergeFrontierCommutes(t *testing.T) {
	ctx := context.Background()
	e, pe, _ := newEngines(t)
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		pool, g := vertexPool(t, 2+r.Intn(7))
		left := randFrontier(t, pe, g, r, pool)
		right := randFrontier(t, pe, g, r, pool)

		lr, err := e.Merge(ctx, g, left, right, nil)
		if err != nil {
			t.Fatalf("seed %d: Merge(L,R): %v", seed, err)
		}
		rl, err := e.Merge(ctx, g, right, left, nil)
		if err != nil {
			t.Fatalf("seed %d: Merge(R,L): %v", seed, err)
		}
		if !setsEqual(frontierSet(lr.Frontier), frontierSet(rl.Frontier)) {
			t.Fatalf("seed %d: merge frontier not commutative as a set", seed)
		}
	}
}

// Property (idempotence): merging a frontier with itself yields exactly its own
// vertex-ID set.
func TestPropMergeIdempotent(t *testing.T) {
	ctx := context.Background()
	e, pe, _ := newEngines(t)
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		pool, g := vertexPool(t, 2+r.Intn(7))
		f := randFrontier(t, pe, g, r, pool)

		mr, err := e.Merge(ctx, g, f, f, nil)
		if err != nil {
			t.Fatalf("seed %d: Merge(f,f): %v", seed, err)
		}
		if !setsEqual(frontierSet(mr.Frontier), idSliceSet(f.VertexIDs())) {
			t.Fatalf("seed %d: Merge(f,f) is not idempotent on the frontier set", seed)
		}
	}
}

// Property (witness determinism): the same merge inputs always produce the same
// witness ID. Content-addressing requires this.
func TestPropWitnessDeterministic(t *testing.T) {
	ctx := context.Background()
	e, pe, _ := newEngines(t)
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		pool, g := vertexPool(t, 2+r.Intn(7))
		left := randFrontier(t, pe, g, r, pool)
		right := randFrontier(t, pe, g, r, pool)

		a, err := e.Merge(ctx, g, left, right, nil)
		if err != nil {
			t.Fatalf("seed %d: Merge a: %v", seed, err)
		}
		b, err := e.Merge(ctx, g, left, right, nil)
		if err != nil {
			t.Fatalf("seed %d: Merge b: %v", seed, err)
		}
		if a.Witness.ID != b.Witness.ID {
			t.Fatalf("seed %d: witness ID not deterministic for identical inputs", seed)
		}
	}
}

// Property (policy gate dominates): if any policy returns Unsat, the merge is
// rejected with a single Policy conflict and no merged frontier — the XOR
// invariant in the failure direction.
func TestPropUnsatPolicyBlocksMerge(t *testing.T) {
	ctx := context.Background()
	e, pe, _ := newEngines(t)
	unsat := []governance.Policy{fixedPolicy{name: "deny", d: governance.Unsat}}
	for iter := 0; iter < propIters; iter++ {
		seed := int64(iter)
		r := rand.New(rand.NewSource(seed))
		pool, g := vertexPool(t, 2+r.Intn(7))
		left := randFrontier(t, pe, g, r, pool)
		right := randFrontier(t, pe, g, r, pool)

		mr, err := e.Merge(ctx, g, left, right, unsat)
		if err != nil {
			t.Fatalf("seed %d: Merge: %v", seed, err)
		}
		if mr.Frontier != nil {
			t.Fatalf("seed %d: Unsat policy must block the merge", seed)
		}
		if len(mr.Conflicts) == 0 {
			t.Fatalf("seed %d: Unsat policy must surface a conflict", seed)
		}
		if mr.Conflicts[0].Kind() != composition.Policy {
			t.Fatalf("seed %d: expected Policy conflict, got %q", seed, mr.Conflicts[0].Kind())
		}
	}
}
