package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
)

func fileVSMode(path, content, mode string) graph.VertexSnapshot {
	v := fileVS(path, content)
	if mode != "" {
		v.Attrs[fileModeAttr] = mode
	}
	return v
}

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
	if !meetsRenameThreshold(lineSimFrac(same, edited)) {
		t.Fatalf("near-identical should score high, got %v", lineSimilarity(same, edited))
	}
	if meetsRenameThreshold(lineSimFrac("aaa\nbbb\nccc\n", "xxx\nyyy\nzzz\n")) {
		t.Fatalf("unrelated should score low, got %v", lineSimilarity("aaa\nbbb\nccc\n", "xxx\nyyy\nzzz\n"))
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

func TestMergeModeThreeWay(t *testing.T) {
	if m, ok := mergeMode("644", "644", "755"); !ok || m != "755" {
		t.Fatalf("one-sided chmod: mode=%q ok=%v", m, ok)
	}
	if m, ok := mergeMode("644", "755", "644"); !ok || m != "755" {
		t.Fatalf("rename-side chmod: mode=%q ok=%v", m, ok)
	}
	if m, ok := mergeMode("644", "755", "755"); !ok || m != "755" {
		t.Fatalf("identical chmod: mode=%q ok=%v", m, ok)
	}
	if _, ok := mergeMode("644", "755", "600"); ok {
		t.Fatal("divergent modes must conflict")
	}
}

func TestReconcileRenamesPreservesChmod(t *testing.T) {
	baseBody := "l1\nl2\nl3\nl4\nl5\n"
	rightBody := "l1\nl2\nL3\nl4\nl5\n"
	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVSMode("old.txt", baseBody, "644")}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVSMode("new.txt", baseBody, "644")}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVSMode("old.txt", rightBody, "755")}}

	lOut, rOut := reconcileRenames(base, left, right)
	if fileModeAt(lOut, "new.txt") != "755" || fileModeAt(rOut, "new.txt") != "755" {
		t.Fatalf("chmod on the non-renaming side must survive: left=%q right=%q",
			fileModeAt(lOut, "new.txt"), fileModeAt(rOut, "new.txt"))
	}
	if fileContentByPath(lOut)["new.txt"] != rightBody {
		t.Fatalf("content should still merge, got %q", fileContentByPath(lOut)["new.txt"])
	}
}

func TestReconcileRenamesRetargetsIncidentEdges(t *testing.T) {
	baseBody := "l1\nl2\nl3\nl4\nl5\n"
	rightBody := "l1\nl2\nL3\nl4\nl5\n"
	oldV, newV := fileVS("old.txt", baseBody), fileVS("new.txt", baseBody)
	editV := fileVS("old.txt", rightBody)
	execID, edgeID := vid("exec"), eid("e1")
	exec := graph.VertexSnapshot{
		ID:    hex.EncodeToString(execID[:]),
		Type:  string(ontology.Execution),
		Attrs: graph.AttrMap{nameAttr: "exec"},
	}
	edge := graph.EdgeSnapshot{
		ID:   hex.EncodeToString(edgeID[:]),
		Type: string(ontology.Materializes),
		From: exec.ID,
		To:   oldV.ID,
	}
	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{oldV, exec}, Edges: []graph.EdgeSnapshot{edge}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{newV, exec}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{editV, exec}, Edges: []graph.EdgeSnapshot{edge}}

	lOut, rOut := reconcileRenames(base, left, right)
	if _, err := lOut.Build(schema()); err != nil {
		t.Fatalf("left snapshot after rename+edge must be well-formed: %v", err)
	}
	if _, err := rOut.Build(schema()); err != nil {
		t.Fatalf("right snapshot after rename+edge must be well-formed: %v", err)
	}
	destVID := vid("new.txt")
	destID := hex.EncodeToString(destVID[:])
	found := false
	for _, e := range rOut.Edges {
		if e.From == exec.ID && e.To == destID && e.Type == string(ontology.Materializes) {
			found = true
		}
		if e.To == oldV.ID || e.From == oldV.ID {
			t.Fatalf("old file id still on an edge: %+v", e)
		}
	}
	if !found {
		t.Fatalf("materializes edge should follow the file to %s, edges=%v", destID, rOut.Edges)
	}
}

func TestCmpSimRationalTie(t *testing.T) {
	if cmpSim(simFrac{3, 5}, simFrac{6, 10}) != 0 {
		t.Fatal("3/5 vs 6/10 must tie")
	}
	if cmpSim(simFrac{4, 5}, simFrac{3, 5}) != 1 {
		t.Fatal("4/5 must beat 3/5")
	}
}

func TestUniqueBestRationalTieRefuses(t *testing.T) {
	// Same rational similarity, different denominators: 6/9 == 4/6 == 2/3.
	target := "a\nb\nc\nd\ne\nf\n"
	cands := []string{"wide.txt", "tall.txt"}
	content := map[string]string{
		"wide.txt": "a\nb\nc\nd\ne\nf\nX\nY\nZ\n", // LCS 6 / max 9
		"tall.txt": "a\nb\nc\nd\nP\nQ\n",          // LCS 4 / max 6
	}
	if _, ok := uniqueBest("old.txt", target, cands, func(p string) string { return content[p] }); ok {
		t.Fatal("equal rational scores must be a tie, not a unique match")
	}
	base := map[string]string{"old.txt": target}
	if m := matchRenames(base, content); len(m) != 0 {
		t.Fatalf("tied rationals must not match: %v", m)
	}
}

func TestLineSimFracLargePairStillMatchRefuse(t *testing.T) {
	// n×m exceeds renameLCSMaxWork, so scoring uses bag overlap.
	n := 1500
	same := nLines(n, "L")
	other := nLines(n, "R")
	if !meetsRenameThreshold(lineSimFrac(same, same)) {
		t.Fatal("identical large files must still meet the rename threshold")
	}
	if meetsRenameThreshold(lineSimFrac(same, other)) {
		t.Fatal("unrelated large files must still refuse")
	}
	base := map[string]string{"old.txt": same}
	if got := matchRenames(base, map[string]string{"new.txt": same}); got["old.txt"] != "new.txt" {
		t.Fatalf("identical large delete+add should match, got %v", got)
	}
	if m := matchRenames(base, map[string]string{"other.txt": other}); len(m) != 0 {
		t.Fatalf("unrelated large delete+add must not match: %v", m)
	}
}

func nLines(n int, prefix string) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%s%d\n", prefix, i)
	}
	return b.String()
}

func tenLineBody() string {
	return "l0\nl1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\n"
}

func halfChangedBody() string {
	return "l0\nl1\nl2\nl3\nl4\nX5\nX6\nX7\nX8\nX9\n"
}

func TestRenameScoreBasenameBonus(t *testing.T) {
	full := simFrac{1, 1}
	same := renameScore("src/foo.go", "pkg/foo.go", full)
	other := renameScore("src/foo.go", "pkg/bar.go", full)
	if cmpSim(same, other) != 1 {
		t.Fatal("same-basename 1.0 must beat different-basename 1.0")
	}
	half := simFrac{1, 2}
	if !meetsRenameThreshold(renameScore("src/foo.go", "pkg/foo.go", half)) {
		t.Fatal("1/2 content + basename bonus must meet 3/5")
	}
	if meetsRenameThreshold(renameScore("src/foo.go", "pkg/bar.go", half)) {
		t.Fatal("1/2 content without basename must miss 3/5")
	}
	low := simFrac{1, 10}
	if meetsRenameThreshold(renameScore("src/foo.go", "pkg/foo.go", low)) {
		t.Fatal("low content + basename bonus must still miss 3/5")
	}
}

func TestMatchRenamesBasenameBreaksContentTie(t *testing.T) {
	body := "a\nb\nc\nd\ne\n"
	base := map[string]string{"src/foo.go": body}
	side := map[string]string{
		"pkg/foo.go": body,
		"pkg/bar.go": body,
	}
	got := matchRenames(base, side)
	if got["src/foo.go"] != "pkg/foo.go" {
		t.Fatalf("same basename should uniquely match, got %v", got)
	}
	// Two same-basename dests still tie.
	tie := map[string]string{
		"pkg/foo.go":   body,
		"other/foo.go": body,
	}
	if m := matchRenames(base, tie); len(m) != 0 {
		t.Fatalf("two same-basename dests must still tie: %v", m)
	}
}

func TestMatchRenamesBasenameNearMiss(t *testing.T) {
	base := map[string]string{"src/foo.go": tenLineBody()}
	if got := matchRenames(base, map[string]string{"pkg/foo.go": halfChangedBody()}); got["src/foo.go"] != "pkg/foo.go" {
		t.Fatalf("50%% same-basename should match, got %v", got)
	}
	if m := matchRenames(base, map[string]string{"pkg/bar.go": halfChangedBody()}); len(m) != 0 {
		t.Fatalf("50%% different-basename must not match: %v", m)
	}
}

func TestMatchRenamesLowSimilaritySameBasenameRefuse(t *testing.T) {
	base := map[string]string{"src/foo.go": tenLineBody()}
	unrel := "z0\nz1\nz2\nz3\nz4\nz5\nz6\nz7\nz8\nz9\n"
	if m := matchRenames(base, map[string]string{"pkg/foo.go": unrel}); len(m) != 0 {
		t.Fatalf("low-similarity same basename must not match: %v", m)
	}
}

func TestMatchDirMoveIdenticalFiles(t *testing.T) {
	ident := "l1\nl2\nl3\nl4\nl5\n"
	base := map[string]string{
		"pkg/a.txt": ident,
		"pkg/b.txt": ident,
		"pkg/c.txt": ident,
	}
	side := map[string]string{
		"lib/a.txt": ident,
		"lib/b.txt": ident,
		"lib/c.txt": ident,
	}
	got := matchRenames(base, side)
	if got["pkg/a.txt"] != "lib/a.txt" || got["pkg/b.txt"] != "lib/b.txt" || got["pkg/c.txt"] != "lib/c.txt" {
		t.Fatalf("directory move should pair by relative path, got %v", got)
	}
}

func TestMatchDirMoveAmbiguousSplitRefuse(t *testing.T) {
	ident := "l1\nl2\nl3\nl4\nl5\n"
	base := map[string]string{
		"pkg/a.txt": ident,
		"pkg/b.txt": ident,
		"pkg/c.txt": ident,
		"pkg/d.txt": ident,
	}
	side := map[string]string{
		"d1/a.txt": ident,
		"d1/b.txt": ident,
		"d2/c.txt": ident,
		"d2/d.txt": ident,
	}
	if m := matchRenames(base, side); len(m) != 0 {
		t.Fatalf("50/50 directory split must not be a tree rename: %v", m)
	}
}

func TestReconcileDirMoveEditAtNewPath(t *testing.T) {
	ident := "l1\nl2\nl3\nl4\nl5\n"
	edited := "l1\nl2\nL3\nl4\nl5\n"
	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("pkg/a.txt", ident),
		fileVS("pkg/b.txt", ident),
		fileVS("pkg/c.txt", ident),
	}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("lib/a.txt", ident),
		fileVS("lib/b.txt", ident),
		fileVS("lib/c.txt", ident),
	}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("pkg/a.txt", edited),
		fileVS("pkg/b.txt", ident),
		fileVS("pkg/c.txt", ident),
	}}

	lOut, rOut := reconcileRenames(base, left, right)
	lC, rC := fileContentByPath(lOut), fileContentByPath(rOut)
	if _, ok := rC["pkg/a.txt"]; ok {
		t.Fatal("old path should be dropped on the edit side after a directory move")
	}
	if lC["lib/a.txt"] != edited || rC["lib/a.txt"] != edited {
		t.Fatalf("edit should merge at new path: left=%q right=%q", lC["lib/a.txt"], rC["lib/a.txt"])
	}
	if lC["lib/b.txt"] != ident || rC["lib/b.txt"] != ident {
		t.Fatalf("unedited files should land at new path, got left=%q right=%q", lC["lib/b.txt"], rC["lib/b.txt"])
	}
}

func TestReconcileDirMoveAmbiguousSplitUnchanged(t *testing.T) {
	ident := "l1\nl2\nl3\nl4\nl5\n"
	edited := "l1\nl2\nL3\nl4\nl5\n"
	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("pkg/a.txt", ident),
		fileVS("pkg/b.txt", ident),
		fileVS("pkg/c.txt", ident),
		fileVS("pkg/d.txt", ident),
	}}
	left := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("d1/a.txt", ident),
		fileVS("d1/b.txt", ident),
		fileVS("d2/c.txt", ident),
		fileVS("d2/d.txt", ident),
	}}
	right := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("pkg/a.txt", edited),
		fileVS("pkg/b.txt", ident),
		fileVS("pkg/c.txt", ident),
		fileVS("pkg/d.txt", ident),
	}}

	_, rOut := reconcileRenames(base, left, right)
	rC := fileContentByPath(rOut)
	if rC["pkg/a.txt"] != edited {
		t.Fatalf("ambiguous split must keep the edit at the old path, got %q", rC["pkg/a.txt"])
	}
	if _, ok := rC["d1/a.txt"]; ok {
		t.Fatal("ambiguous split must not copy the edit onto a guessed dest")
	}
}
