package graph_test

import (
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
)

func snapOf(t *testing.T, build func(*graph.Builder)) graph.Snapshot {
	t.Helper()
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	build(b)
	return graph.EncodeSnapshot(b.Build())
}

func TestDiffIdenticalIsEmpty(t *testing.T) {
	s := snapOf(t, func(b *graph.Builder) {
		b.AddVertex(graph.Vertex{ID: vid("a"), Type: ontology.Artifact})
	})
	if !graph.Diff(s, s).Empty() {
		t.Fatal("diff of identical snapshots should be empty")
	}
}

func TestDiffAddedRemoved(t *testing.T) {
	old := snapOf(t, func(b *graph.Builder) {
		b.AddVertex(graph.Vertex{ID: vid("a"), Type: ontology.Artifact})
	})
	newer := snapOf(t, func(b *graph.Builder) {
		b.AddVertex(graph.Vertex{ID: vid("b"), Type: ontology.Artifact})
	})
	d := graph.Diff(old, newer)
	if len(d.AddedVertices) != 1 || d.AddedVertices[0].ID != hexOf(vid("b")) {
		t.Fatalf("expected b added, got %+v", d.AddedVertices)
	}
	if len(d.RemovedVertices) != 1 || d.RemovedVertices[0].ID != hexOf(vid("a")) {
		t.Fatalf("expected a removed, got %+v", d.RemovedVertices)
	}
}

// Same ID, different content → Changed (only possible because ID = sha256(name),
// not a full-content hash).
func TestDiffChangedVertex(t *testing.T) {
	old := snapOf(t, func(b *graph.Builder) {
		b.AddVertex(graph.Vertex{ID: vid("x"), Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}})
	})
	newer := snapOf(t, func(b *graph.Builder) {
		b.AddVertex(graph.Vertex{ID: vid("x"), Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "done"}})
	})
	d := graph.Diff(old, newer)
	if len(d.ChangedVertices) != 1 {
		t.Fatalf("expected 1 changed vertex, got %d", len(d.ChangedVertices))
	}
	if d.ChangedVertices[0].Old.Attrs["status"] != "draft" || d.ChangedVertices[0].New.Attrs["status"] != "done" {
		t.Fatalf("unexpected change payload: %+v", d.ChangedVertices[0])
	}
	if len(d.AddedVertices) != 0 || len(d.RemovedVertices) != 0 {
		t.Fatal("a content change should not also be add/remove")
	}
}

func TestDiffEdges(t *testing.T) {
	base := func(b *graph.Builder) {
		b.AddVertex(graph.Vertex{ID: vid("a"), Type: ontology.Artifact})
		b.AddVertex(graph.Vertex{ID: vid("b"), Type: ontology.Artifact})
	}
	old := snapOf(t, base)
	newer := snapOf(t, func(b *graph.Builder) {
		base(b)
		b.AddEdge(graph.Edge{ID: eid("e"), Type: ontology.DerivedFrom, From: vid("a"), To: vid("b")})
	})
	d := graph.Diff(old, newer)
	if len(d.AddedEdges) != 1 {
		t.Fatalf("expected 1 added edge, got %d", len(d.AddedEdges))
	}
	// reverse
	back := graph.Diff(newer, old)
	if len(back.RemovedEdges) != 1 {
		t.Fatalf("expected 1 removed edge in reverse diff, got %d", len(back.RemovedEdges))
	}
}
