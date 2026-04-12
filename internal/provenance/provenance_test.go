package provenance_test

import (
	"crypto/sha256"
	"sort"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/provenance"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func eid(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

func newEngine() provenance.Engine {
	return provenance.NewEngine(ontology.CausalEdges)
}

// buildChain creates: Execution -executes-> Model, Execution -derived_from-> Prompt,
// Execution -materializes-> Artifact.
func buildChain(t *testing.T) graph.Graph {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	exec := vid("exec")
	model := vid("model")
	prompt := vid("prompt")
	artifact := vid("artifact")

	var err error
	g, _ = g.WithVertex(graph.Vertex{ID: exec, Type: ontology.Execution})
	g, _ = g.WithVertex(graph.Vertex{ID: model, Type: ontology.Model})
	g, _ = g.WithVertex(graph.Vertex{ID: prompt, Type: ontology.Prompt})
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})

	g, err = g.WithEdge(graph.Edge{ID: eid("e-m"), Type: ontology.Executes, From: exec, To: model})
	if err != nil {
		t.Fatal(err)
	}
	g, err = g.WithEdge(graph.Edge{ID: eid("e-p"), Type: ontology.DerivedFrom, From: exec, To: prompt})
	if err != nil {
		t.Fatal(err)
	}
	g, err = g.WithEdge(graph.Edge{ID: eid("e-a"), Type: ontology.Materializes, From: exec, To: artifact})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func idSet(ids []identity.VertexID) map[identity.VertexID]bool {
	m := make(map[identity.VertexID]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// Axiom: S subset Close(G, S) — extensive.
func TestCloseExtensive(t *testing.T) {
	g := buildChain(t)
	e := newEngine()
	seed := []identity.VertexID{vid("exec")}

	closed, err := e.Close(g, seed)
	if err != nil {
		t.Fatal(err)
	}
	cs := idSet(closed)
	for _, s := range seed {
		if !cs[s] {
			t.Fatal("closure does not contain seed — extensiveness violated")
		}
	}
}

// Axiom: Close(G, Close(G, S)) = Close(G, S) — idempotent.
func TestCloseIdempotent(t *testing.T) {
	g := buildChain(t)
	e := newEngine()
	seed := []identity.VertexID{vid("exec")}

	c1, _ := e.Close(g, seed)
	c2, _ := e.Close(g, c1)

	if len(c1) != len(c2) {
		t.Fatalf("idempotence violated: |Close| = %d, |Close(Close)| = %d", len(c1), len(c2))
	}
	s1 := idSet(c1)
	for _, id := range c2 {
		if !s1[id] {
			t.Fatal("idempotence violated: second closure contains extra vertex")
		}
	}
}

// Axiom: S1 subset S2 => Close(G, S1) subset Close(G, S2) — monotone.
func TestCloseMonotone(t *testing.T) {
	g := buildChain(t)
	e := newEngine()
	s1 := []identity.VertexID{vid("exec")}
	s2 := []identity.VertexID{vid("exec"), vid("model")}

	c1, _ := e.Close(g, s1)
	c2, _ := e.Close(g, s2)
	cs2 := idSet(c2)

	for _, id := range c1 {
		if !cs2[id] {
			t.Fatal("monotonicity violated: Close(S1) contains vertex not in Close(S2)")
		}
	}
}

// Axiom: provCone(G, v) = provClose(G, {v}).
func TestConeEqualsSingletonClose(t *testing.T) {
	g := buildChain(t)
	e := newEngine()
	v := vid("exec")

	cone, _ := e.Cone(g, v)
	closed, _ := e.Close(g, []identity.VertexID{v})

	if len(cone) != len(closed) {
		t.Fatalf("Cone != Close singleton: %d vs %d", len(cone), len(closed))
	}
	cs := idSet(closed)
	for _, id := range cone {
		if !cs[id] {
			t.Fatal("Cone contains vertex not in Close singleton")
		}
	}
}

// The full chain should be causally connected (undirected traversal).
func TestCausesConnectedChain(t *testing.T) {
	g := buildChain(t)
	e := newEngine()

	// All four vertices are connected via causal edges.
	pairs := [][2]string{
		{"exec", "model"},
		{"exec", "prompt"},
		{"exec", "artifact"},
		{"model", "artifact"}, // transitive through exec
		{"prompt", "artifact"},
	}
	for _, p := range pairs {
		ok, err := e.Causes(g, vid(p[0]), vid(p[1]))
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("Causes(%s, %s) = false, expected true", p[0], p[1])
		}
	}
}

// Causes is reflexive.
func TestCausesReflexive(t *testing.T) {
	g := buildChain(t)
	e := newEngine()
	ok, _ := e.Causes(g, vid("exec"), vid("exec"))
	if !ok {
		t.Fatal("Causes should be reflexive")
	}
}

// Non-causal edges (authored_by) should NOT be traversed.
func TestNonCausalEdgeExcluded(t *testing.T) {
	g := graph.NewGraph(ontology.NewDefaultSchema())
	agent := vid("agent")
	artifact := vid("artifact")
	g, _ = g.WithVertex(graph.Vertex{ID: agent, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: artifact, Type: ontology.Artifact})
	g, _ = g.WithEdge(graph.Edge{
		ID: eid("auth"), Type: ontology.AuthoredBy,
		From: agent, To: artifact,
	})

	e := newEngine()
	ok, _ := e.Causes(g, agent, artifact)
	if ok {
		t.Fatal("non-causal edge (authored_by) should not establish causation")
	}
}

// TraceSet finds all simple paths.
func TestTraceSet(t *testing.T) {
	g := buildChain(t)
	e := newEngine()

	// model -> exec -> artifact (length 3, through exec)
	traces, err := e.TraceSet(g, vid("model"), vid("artifact"))
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) == 0 {
		t.Fatal("expected at least one trace from model to artifact")
	}
	// Verify trace contains valid vertices.
	for _, tr := range traces {
		vs := tr.Vertices()
		if vs[0] != vid("model") {
			t.Error("trace should start with model")
		}
		if vs[len(vs)-1] != vid("artifact") {
			t.Error("trace should end with artifact")
		}
	}
}

// Closure of disconnected vertex returns only itself.
func TestCloseIsolatedVertex(t *testing.T) {
	g := graph.NewGraph(ontology.NewDefaultSchema())
	lonely := vid("lonely")
	g, _ = g.WithVertex(graph.Vertex{ID: lonely, Type: ontology.Artifact})

	e := newEngine()
	closed, _ := e.Close(g, []identity.VertexID{lonely})
	if len(closed) != 1 || closed[0] != lonely {
		t.Fatal("closure of isolated vertex should be {vertex}")
	}
}

// Cone of exec should include all 4 vertices (fully connected via causal edges).
func TestConeSize(t *testing.T) {
	g := buildChain(t)
	e := newEngine()
	cone, _ := e.Cone(g, vid("exec"))

	expected := []string{"exec", "model", "prompt", "artifact"}
	if len(cone) != len(expected) {
		names := make([]string, len(cone))
		for i, id := range cone {
			for _, n := range expected {
				if vid(n) == id {
					names[i] = n
				}
			}
		}
		sort.Strings(names)
		t.Fatalf("expected %d vertices in cone, got %d: %v", len(expected), len(cone), names)
	}
}
