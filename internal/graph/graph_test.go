package graph_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func eid(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

func hid(s string) identity.HyperedgeID {
	return identity.HyperedgeID(sha256.Sum256([]byte(s)))
}

func newGraph() graph.Graph {
	return graph.NewGraph(ontology.NewDefaultSchema())
}

func mustAddVertex(t *testing.T, g graph.Graph, id identity.VertexID, vt ontology.VertexType) graph.Graph {
	t.Helper()
	g2, err := g.WithVertex(graph.Vertex{ID: id, Type: vt})
	if err != nil {
		t.Fatalf("WithVertex(%v): %v", vt, err)
	}
	return g2
}

// Axiom: vertexId(v) in vertexIDs(addV(G, v)).
func TestWithVertexAddsID(t *testing.T) {
	g := newGraph()
	id := vid("artifact-1")
	g2 := mustAddVertex(t, g, id, ontology.Artifact)

	found := false
	for _, v := range g2.VertexIDs() {
		if v == id {
			found = true
		}
	}
	if !found {
		t.Fatal("vertex ID not found after WithVertex")
	}
}

// Axiom: addE(G, e) = G if src(e) not in vertexIDs(G).
func TestWithEdgeRejectsMissingSrc(t *testing.T) {
	g := newGraph()
	dst := vid("dst")
	g = mustAddVertex(t, g, dst, ontology.Model)

	_, err := g.WithEdge(graph.Edge{
		ID: eid("e1"), Type: ontology.Executes,
		From: vid("missing"), To: dst,
	})
	if err == nil {
		t.Fatal("expected error for missing source vertex")
	}
	if !errors.Is(err, graph.ErrMissingEndpoint) {
		t.Fatalf("expected ErrMissingEndpoint, got: %v", err)
	}
}

// Axiom: addE(G, e) = G if dst(e) not in vertexIDs(G).
func TestWithEdgeRejectsMissingDst(t *testing.T) {
	g := newGraph()
	src := vid("src")
	g = mustAddVertex(t, g, src, ontology.Execution)

	_, err := g.WithEdge(graph.Edge{
		ID: eid("e1"), Type: ontology.Executes,
		From: src, To: vid("missing"),
	})
	if err == nil {
		t.Fatal("expected error for missing destination vertex")
	}
}

// Axiom: addH(G, h) = G if (inputs(h) union outputs(h)) not subset vertexIDs(G).
func TestWithHyperedgeRejectsMissingVertex(t *testing.T) {
	g := newGraph()
	p := vid("prompt")
	g = mustAddVertex(t, g, p, ontology.Prompt)

	_, err := g.WithHyperedge(graph.Hyperedge{
		ID:      hid("h1"),
		Type:    ontology.Executes,
		Inputs:  []identity.VertexID{p},
		Outputs: []identity.VertexID{vid("missing")},
	})
	if err == nil {
		t.Fatal("expected error for missing hyperedge output")
	}
}

// Axiom: extends(G, addV(G, v)) — adding a vertex extends the graph.
func TestWithVertexExtends(t *testing.T) {
	g := newGraph()
	g2 := mustAddVertex(t, g, vid("a"), ontology.Artifact)

	if len(g2.VertexIDs()) <= len(g.VertexIDs()) {
		t.Fatal("graph should have grown after WithVertex")
	}
}

// WithVertex is pure: original graph is unchanged.
func TestWithVertexImmutability(t *testing.T) {
	g := newGraph()
	before := len(g.VertexIDs())
	mustAddVertex(t, g, vid("a"), ontology.Artifact)
	if len(g.VertexIDs()) != before {
		t.Fatal("original graph was mutated by WithVertex")
	}
}

// Axiom: wellFormed(G) => every edge satisfies admissibleEdge.
func TestValidateAcceptsAdmissibleEdge(t *testing.T) {
	g := newGraph()
	exec := vid("exec")
	model := vid("model")
	g = mustAddVertex(t, g, exec, ontology.Execution)
	g = mustAddVertex(t, g, model, ontology.Model)

	g, err := g.WithEdge(graph.Edge{
		ID: eid("e1"), Type: ontology.Executes,
		From: exec, To: model,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("valid graph failed Validate: %v", err)
	}
}

// Validate rejects inadmissible edges.
func TestValidateRejectsInadmissibleEdge(t *testing.T) {
	g := newGraph()
	a := vid("art")
	m := vid("model")
	g = mustAddVertex(t, g, a, ontology.Artifact)
	g = mustAddVertex(t, g, m, ontology.Model)

	// Artifact -executes-> Model is NOT in the admissibility table.
	g, err := g.WithEdge(graph.Edge{
		ID: eid("bad"), Type: ontology.Executes,
		From: a, To: m,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Validate(); err == nil {
		t.Fatal("expected Validate to reject inadmissible edge")
	}
}

// Validate accepts the canonical executes hyperedge.
func TestValidateAcceptsCanonicalHyperedge(t *testing.T) {
	g := newGraph()
	ids := map[string]identity.VertexID{
		"prompt": vid("prompt"), "model": vid("model"),
		"artifact": vid("artifact"), "policy": vid("policy"),
		"revision": vid("revision"), "observation": vid("observation"),
	}
	types := map[string]ontology.VertexType{
		"prompt": ontology.Prompt, "model": ontology.Model,
		"artifact": ontology.Artifact, "policy": ontology.Policy,
		"revision": ontology.Revision, "observation": ontology.Observation,
	}
	for name, id := range ids {
		g = mustAddVertex(t, g, id, types[name])
	}

	g, err := g.WithHyperedge(graph.Hyperedge{
		ID:   hid("h1"),
		Type: ontology.Executes,
		Inputs: []identity.VertexID{
			ids["prompt"], ids["model"], ids["artifact"], ids["policy"],
		},
		Outputs: []identity.VertexID{ids["revision"], ids["observation"]},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("canonical hyperedge failed Validate: %v", err)
	}
}

// Induce returns the correct subgraph.
func TestInduce(t *testing.T) {
	g := newGraph()
	a := vid("a")
	b := vid("b")
	c := vid("c")
	g = mustAddVertex(t, g, a, ontology.Artifact)
	g = mustAddVertex(t, g, b, ontology.Artifact)
	g = mustAddVertex(t, g, c, ontology.Revision)

	g, _ = g.WithEdge(graph.Edge{
		ID: eid("ab"), Type: ontology.DerivedFrom, From: a, To: b,
	})
	g, _ = g.WithEdge(graph.Edge{
		ID: eid("ac"), Type: ontology.DerivedFrom, From: a, To: c,
	})

	sub, err := g.Induce([]identity.VertexID{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.Vertices()) != 2 {
		t.Fatalf("expected 2 vertices, got %d", len(sub.Vertices()))
	}
	// Only edge a->b should be induced (a->c excluded since c not in set).
	if len(sub.Edges()) != 1 {
		t.Fatalf("expected 1 induced edge, got %d", len(sub.Edges()))
	}
}

// Induce rejects unknown vertex IDs.
func TestInduceRejectsMissing(t *testing.T) {
	g := newGraph()
	_, err := g.Induce([]identity.VertexID{vid("ghost")})
	if err == nil {
		t.Fatal("expected error for inducing with unknown vertex ID")
	}
}

// Empty graph validates cleanly.
func TestEmptyGraphValidates(t *testing.T) {
	g := newGraph()
	if err := g.Validate(); err != nil {
		t.Fatalf("empty graph should validate: %v", err)
	}
}

// Vertex-only graph validates cleanly.
func TestVertexOnlyGraphValidates(t *testing.T) {
	g := newGraph()
	for i := 0; i < 10; i++ {
		g = mustAddVertex(t, g, vid(fmt.Sprintf("v%d", i)), ontology.Artifact)
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("vertex-only graph should validate: %v", err)
	}
}
