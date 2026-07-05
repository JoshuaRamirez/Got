package graph_test

import (
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// queryGraph builds a small graph: two Artifacts (one tagged status=done),
// one Model, and an admissible Artifact->Artifact derived_from edge.
func queryGraph(t *testing.T) graph.Graph {
	t.Helper()
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(b.AddVertex(graph.Vertex{ID: vid("a1"), Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "done"}}))
	must(b.AddVertex(graph.Vertex{ID: vid("a2"), Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}}))
	must(b.AddVertex(graph.Vertex{ID: vid("m1"), Type: ontology.Model}))
	must(b.AddEdge(graph.Edge{ID: eid("e"), Type: ontology.DerivedFrom, From: vid("a1"), To: vid("a2")}))
	return b.Build()
}

func idSet(ids []identity.VertexID) map[identity.VertexID]bool {
	m := make(map[identity.VertexID]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// ByType selects all vertices of a type and induces the edges among them.
func TestQueryByType(t *testing.T) {
	g := queryGraph(t)
	sub, err := g.Query(graph.ByType{Type: ontology.Artifact})
	if err != nil {
		t.Fatal(err)
	}
	got := idSet(sub.VertexIDs())
	if len(got) != 2 || !got[vid("a1")] || !got[vid("a2")] {
		t.Fatalf("expected {a1,a2}, got %d vertices", len(got))
	}
	// The a1->a2 edge is induced (both endpoints matched).
	if len(sub.Edges()) != 1 {
		t.Fatalf("expected the induced derived_from edge, got %d edges", len(sub.Edges()))
	}
}

// ByType with no matches yields an empty subgraph.
func TestQueryByTypeNoMatch(t *testing.T) {
	g := queryGraph(t)
	sub, err := g.Query(graph.ByType{Type: ontology.Tool})
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.VertexIDs()) != 0 {
		t.Fatal("expected no Tool vertices")
	}
}

// ByAttr matches only vertices carrying the key/value; absent key does not match.
func TestQueryByAttr(t *testing.T) {
	g := queryGraph(t)
	sub, err := g.Query(graph.ByAttr{Key: "status", Value: "done"})
	if err != nil {
		t.Fatal(err)
	}
	got := idSet(sub.VertexIDs())
	if len(got) != 1 || !got[vid("a1")] {
		t.Fatalf("expected {a1}, got %v", sub.VertexIDs())
	}
}

// And is the intersection of its sub-queries.
func TestQueryAnd(t *testing.T) {
	g := queryGraph(t)
	q := graph.And{Queries: []graph.Query{
		graph.ByType{Type: ontology.Artifact},
		graph.ByAttr{Key: "status", Value: "draft"},
	}}
	sub, err := g.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	got := idSet(sub.VertexIDs())
	if len(got) != 1 || !got[vid("a2")] {
		t.Fatalf("expected {a2}, got %v", sub.VertexIDs())
	}
}

// Or is the union of its sub-queries.
func TestQueryOr(t *testing.T) {
	g := queryGraph(t)
	q := graph.Or{Queries: []graph.Query{
		graph.ByType{Type: ontology.Model},
		graph.ByAttr{Key: "status", Value: "done"},
	}}
	sub, err := g.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	got := idSet(sub.VertexIDs())
	if len(got) != 2 || !got[vid("m1")] || !got[vid("a1")] {
		t.Fatalf("expected {m1,a1}, got %v", sub.VertexIDs())
	}
}

// Empty And / Or match nothing.
func TestQueryEmptyComposites(t *testing.T) {
	g := queryGraph(t)
	for name, q := range map[string]graph.Query{"and": graph.And{}, "or": graph.Or{}} {
		sub, err := g.Query(q)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(sub.VertexIDs()) != 0 {
			t.Fatalf("%s: expected empty result", name)
		}
	}
}

// A nested composite (And of Or) evaluates correctly.
func TestQueryNested(t *testing.T) {
	g := queryGraph(t)
	q := graph.And{Queries: []graph.Query{
		graph.ByType{Type: ontology.Artifact},
		graph.Or{Queries: []graph.Query{
			graph.ByAttr{Key: "status", Value: "done"},
			graph.ByAttr{Key: "status", Value: "draft"},
		}},
	}}
	sub, err := g.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.VertexIDs()) != 2 {
		t.Fatalf("expected both artifacts, got %d", len(sub.VertexIDs()))
	}
}

// unknownQuery is a Query type the engine does not recognize.
type unknownQuery struct{}

func (unknownQuery) QueryKind() string { return "unknown" }

// An unrecognized query type yields ErrQueryUnsupported (also inside a composite).
func TestQueryUnsupported(t *testing.T) {
	g := queryGraph(t)
	if _, err := g.Query(unknownQuery{}); err == nil {
		t.Fatal("expected ErrQueryUnsupported for unknown query")
	}
	if _, err := g.Query(graph.And{Queries: []graph.Query{graph.ByType{Type: ontology.Artifact}, unknownQuery{}}}); err == nil {
		t.Fatal("expected error to propagate out of a composite")
	}
}
