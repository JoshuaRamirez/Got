package main

import (
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
)

func TestLineSimilarity(t *testing.T) {
	same := "l1\nl2\nl3\nl4\n"
	if g := lineSimilarity(same, same); g != 1 {
		t.Fatalf("identical: %v", g)
	}
	if g := lineSimilarity("", "x\n"); g != 0 {
		t.Fatalf("empty: %v", g)
	}
	// 3 of 4 lines shared with a one-line edit.
	edited := "l1\nl2\nL3\nl4\n"
	if g := lineSimilarity(same, edited); g < renameSimilarityThreshold {
		t.Fatalf("near-identical should score high, got %v", g)
	}
	if g := lineSimilarity("aaa\nbbb\nccc\n", "xxx\nyyy\nzzz\n"); g >= renameSimilarityThreshold {
		t.Fatalf("unrelated should score low, got %v", g)
	}
}

func TestMatchRenames(t *testing.T) {
	base := map[string]string{"old.txt": "a\nb\nc\nd\ne\n"}
	// High-similarity add at a new path matches.
	side := map[string]string{"new.txt": "a\nb\nc\nd\nE\n"}
	got := matchRenames(base, side)
	if got["old.txt"] != "new.txt" {
		t.Fatalf("expected old→new, got %v", got)
	}
	// Unrelated add does not match.
	unrel := map[string]string{"other.txt": "zz\nyy\nxx\nww\nvv\n"}
	if m := matchRenames(base, unrel); len(m) != 0 {
		t.Fatalf("unrelated delete+add must not match: %v", m)
	}
	// Two identical added files are an ambiguous tie — refuse.
	tie := map[string]string{
		"a.txt": "a\nb\nc\nd\ne\n",
		"b.txt": "a\nb\nc\nd\ne\n",
	}
	if m := matchRenames(base, tie); len(m) != 0 {
		t.Fatalf("tied candidates must not match: %v", m)
	}
}

func TestReconcileRenamesEditAtNewPath(t *testing.T) {
	baseBody := "l1\nl2\nl3\nl4\nl5\n"
	leftBody := "l1\nl2\nl3\nl4\nl5\n" // renamed, content unchanged
	rightBody := "l1\nl2\nL3\nl4\nl5\n"

	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("old.txt", baseBody)}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("new.txt", leftBody)}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("old.txt", rightBody)}}

	lOut, rOut := reconcileRenames(base, left, right)
	lC, rC := fileContentByPath(lOut), fileContentByPath(rOut)
	if _, ok := lC["old.txt"]; ok {
		t.Fatal("old path should be gone on the rename side")
	}
	if _, ok := rC["old.txt"]; ok {
		t.Fatal("old path should be dropped on the edit side after a matched rename")
	}
	want := "l1\nl2\nL3\nl4\nl5\n"
	if lC["new.txt"] != want || rC["new.txt"] != want {
		t.Fatalf("merged content at new path: left=%q right=%q want=%q", lC["new.txt"], rC["new.txt"], want)
	}
}

func TestReconcileRenamesUnrelatedDoesNotMatch(t *testing.T) {
	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("old.txt", "aaaa\nbbbb\ncccc\ndddd\n")}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("new.txt", "zzzz\nyyyy\nxxxx\nwwww\n")}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("old.txt", "aaaa\nBBBB\ncccc\ndddd\n")}}

	lOut, rOut := reconcileRenames(base, left, right)
	lC, rC := fileContentByPath(lOut), fileContentByPath(rOut)
	if _, ok := lC["old.txt"]; ok {
		t.Fatal("rename side still has only the add")
	}
	if rC["old.txt"] != "aaaa\nBBBB\ncccc\ndddd\n" {
		t.Fatalf("edit side must keep the old path, got %q", rC["old.txt"])
	}
	if _, ok := rC["new.txt"]; ok {
		t.Fatal("unrelated add must not be copied onto the edit side")
	}
}

func TestReconcileRenamesOverlappingContentStillConflicts(t *testing.T) {
	baseBody := "a\nb\nc\n"
	leftBody := "a\nX\nc\n"  // renamed + edit of b
	rightBody := "a\nY\nc\n" // edit of b, different

	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("old.txt", baseBody)}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("new.txt", leftBody)}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("old.txt", rightBody)}}

	lOut, rOut := reconcileRenames(base, left, right)
	lC, rC := fileContentByPath(lOut), fileContentByPath(rOut)
	// Content conflict: leave the snapshots as a delete+add / modify.
	if _, ok := rC["old.txt"]; !ok {
		t.Fatal("overlapping edit must not drop the old path")
	}
	if rC["old.txt"] != rightBody {
		t.Fatalf("edit side unchanged, got %q", rC["old.txt"])
	}
	if lC["new.txt"] != leftBody {
		t.Fatalf("rename side unchanged, got %q", lC["new.txt"])
	}
}

func TestReconcileRenamesBothSidesSameDest(t *testing.T) {
	baseBody := "a\nb\nc\nd\n"
	leftBody := "A\nb\nc\nd\n"
	rightBody := "a\nb\nc\nD\n"

	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("old.txt", baseBody)}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("new.txt", leftBody)}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("new.txt", rightBody)}}

	lOut, rOut := reconcileRenames(base, left, right)
	lC, rC := fileContentByPath(lOut), fileContentByPath(rOut)
	want := "A\nb\nc\nD\n"
	if lC["new.txt"] != want || rC["new.txt"] != want {
		t.Fatalf("both-sides rename: left=%q right=%q want=%q", lC["new.txt"], rC["new.txt"], want)
	}
}
