package history_test

import (
	"crypto/sha256"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/history"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

func vid(s string) identity.VertexID { return identity.VertexID(sha256.Sum256([]byte(s))) }

// snap builds a snapshot of a graph containing the named Artifact vertices.
func snap(t *testing.T, names ...string) graph.Snapshot {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	for _, n := range names {
		var err error
		g, err = g.WithVertex(graph.Vertex{ID: vid(n), Type: ontology.Artifact})
		if err != nil {
			t.Fatal(err)
		}
	}
	return graph.EncodeSnapshot(g)
}

// Same (parents, message, actor, state) → same ID; a changed field → new ID.
func TestCommitContentAddressed(t *testing.T) {
	a := history.NewCommit(nil, "init", "alice", nil, []identity.VertexID{vid("x")}, snap(t, "x"))
	b := history.NewCommit(nil, "init", "alice", nil, []identity.VertexID{vid("x")}, snap(t, "x"))
	if a.ID != b.ID {
		t.Fatal("identical commits should share an ID")
	}
	c := history.NewCommit(nil, "different message", "alice", nil, nil, snap(t, "x"))
	if c.ID == a.ID {
		t.Fatal("different message should change the ID")
	}
	d := history.NewCommit(nil, "init", "alice", nil, nil, snap(t, "x", "y"))
	if d.ID == a.ID {
		t.Fatal("different resulting state should change the ID")
	}
}

// A vertex id is content-addressed on its name (e.g. a file path), so editing a
// file in place keeps the same vertex id but changes its attributes. The commit
// id must still differ, or Log.Add would silently drop the second commit and
// checkout would return stale content.
func TestCommitIDReflectsAttrs(t *testing.T) {
	snapWithContent := func(content string) graph.Snapshot {
		g := graph.NewGraph(ontology.NewDefaultSchema())
		g, err := g.WithVertex(graph.Vertex{
			ID:    vid("main.go"),
			Type:  ontology.Artifact,
			Attrs: graph.AttrMap{"file.content": content},
		})
		if err != nil {
			t.Fatal(err)
		}
		return graph.EncodeSnapshot(g)
	}
	v1 := history.NewCommit(nil, "wip", "alice", nil, nil, snapWithContent("package main // v1"))
	v2 := history.NewCommit(nil, "wip", "alice", nil, nil, snapWithContent("package main // v2"))
	if v1.ID == v2.ID {
		t.Fatal("same path + same message but different content must yield distinct commit IDs")
	}
	// And identical content still dedups.
	v1again := history.NewCommit(nil, "wip", "alice", nil, nil, snapWithContent("package main // v1"))
	if v1.ID != v1again.ID {
		t.Fatal("identical content should share a commit ID")
	}
}

// Consumed/Produced are annotation, not identity.
func TestCommitDeltaNotInID(t *testing.T) {
	a := history.NewCommit(nil, "m", "a", []identity.VertexID{vid("z")}, []identity.VertexID{vid("x")}, snap(t, "x"))
	b := history.NewCommit(nil, "m", "a", nil, nil, snap(t, "x"))
	if a.ID != b.ID {
		t.Fatal("delta should not affect commit identity")
	}
}

// Ancestors walks the parent DAG, including a merge commit's two parents.
func TestLogAncestorsAndMerge(t *testing.T) {
	l := history.NewLog()
	root := history.NewCommit(nil, "root", "a", nil, nil, snap(t, "r"))
	left := history.NewCommit([]history.CommitID{root.ID}, "left", "a", nil, nil, snap(t, "r", "l"))
	right := history.NewCommit([]history.CommitID{root.ID}, "right", "a", nil, nil, snap(t, "r", "rt"))
	merge := history.NewCommit([]history.CommitID{left.ID, right.ID}, "merge", "a", nil, nil, snap(t, "r", "l", "rt"))
	for _, c := range []history.Commit{root, left, right, merge} {
		if err := l.Add(c); err != nil {
			t.Fatal(err)
		}
	}

	anc, err := l.Ancestors(merge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(anc) != 4 {
		t.Fatalf("expected 4 commits in ancestry, got %d", len(anc))
	}
	if anc[0].ID != merge.ID {
		t.Fatal("ancestry should start with the queried commit")
	}
	// Root must appear (reachable via both parents, deduped).
	found := false
	for _, c := range anc {
		if c.ID == root.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("root should be in the ancestry")
	}
}

func TestAddUnknownParent(t *testing.T) {
	l := history.NewLog()
	orphan := history.NewCommit([]history.CommitID{{1, 2, 3}}, "x", "a", nil, nil, snap(t, "x"))
	if err := l.Add(orphan); err == nil {
		t.Fatal("expected ErrUnknownParent")
	}
}

func TestAncestorsUnknownCommit(t *testing.T) {
	l := history.NewLog()
	if _, err := l.Ancestors(history.CommitID{9}); err == nil {
		t.Fatal("expected ErrUnknownCommit")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	l := history.NewLog()
	root := history.NewCommit(nil, "root", "alice", nil, []identity.VertexID{vid("r")}, snap(t, "r"))
	child := history.NewCommit([]history.CommitID{root.ID}, "child", "bob", nil, []identity.VertexID{vid("c")}, snap(t, "r", "c"))
	_ = l.Add(root)
	_ = l.Add(child)

	data, err := history.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	back, err := history.Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := back.Get(child.ID)
	if !ok {
		t.Fatal("child commit did not survive round-trip")
	}
	if got.Message != "child" || got.Actor != "bob" || len(got.Parents) != 1 || got.Parents[0] != root.ID {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if len(back.Commits()) != 2 {
		t.Fatalf("expected 2 commits after round-trip, got %d", len(back.Commits()))
	}
}

func TestMergeBase(t *testing.T) {
	l := history.NewLog()
	root := history.NewCommit(nil, "root", "a", nil, nil, snap(t, "r"))
	left := history.NewCommit([]history.CommitID{root.ID}, "left", "a", nil, nil, snap(t, "r", "l"))
	right := history.NewCommit([]history.CommitID{root.ID}, "right", "a", nil, nil, snap(t, "r", "rt"))
	for _, c := range []history.Commit{root, left, right} {
		if err := l.Add(c); err != nil {
			t.Fatal(err)
		}
	}
	base, ok := l.MergeBase(left.ID, right.ID)
	if !ok || base != root.ID {
		t.Fatalf("merge-base(left,right) = %v ok=%v, want root", base, ok)
	}
	// An ancestor is its own merge-base with a descendant.
	if base, ok := l.MergeBase(root.ID, left.ID); !ok || base != root.ID {
		t.Fatalf("merge-base(root,left) should be root")
	}
	// Unrelated commit → no common ancestor.
	orphan := history.NewCommit(nil, "orphan", "a", nil, nil, snap(t, "o"))
	_ = l.Add(orphan)
	if _, ok := l.MergeBase(left.ID, orphan.ID); ok {
		t.Fatal("unrelated commits should have no merge-base")
	}
}
