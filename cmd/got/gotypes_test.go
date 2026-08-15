package main

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
)

func fileVS(path, content string) graph.VertexSnapshot {
	id := vid(path)
	return graph.VertexSnapshot{
		ID:   hex.EncodeToString(id[:]),
		Type: string(ontology.Artifact),
		Attrs: graph.AttrMap{
			filePathAttr:    path,
			fileContentAttr: base64.StdEncoding.EncodeToString([]byte(content)),
		},
	}
}

// A .go file deleted by the merge (present in base, absent in merged) must mark
// its directory changed, so a resulting dangling reference is still gated —
// even though no surviving file in the directory changed.
func TestSemanticGateDetectsDeletion(t *testing.T) {
	main := "package p\n\nfunc M() int { return helper() }\n"
	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("main.go", main),
		fileVS("helper.go", "package p\n\nfunc helper() int { return 7 }\n"),
	}}
	// Merged graph: helper.go removed; main.go unchanged.
	mergedSnap := graph.Snapshot{Vertices: []graph.VertexSnapshot{fileVS("main.go", main)}}
	mg, err := mergedSnap.Build(schema())
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := semanticGateOK(mg, base); ok {
		t.Fatal("deleting helper.go leaves a dangling reference; the gate should refuse")
	}
}

// Control: an unrelated deletion that breaks nothing passes.
func TestSemanticGateDeletionNoBreak(t *testing.T) {
	base := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("main.go", "package p\n\nfunc M() int { return 1 }\n"),
		fileVS("extra.go", "package p\n\nfunc unused() int { return 2 }\n"),
	}}
	mergedSnap := graph.Snapshot{Vertices: []graph.VertexSnapshot{
		fileVS("main.go", "package p\n\nfunc M() int { return 1 }\n"),
	}}
	mg, err := mergedSnap.Build(schema())
	if err != nil {
		t.Fatal(err)
	}
	if ok, detail := semanticGateOK(mg, base); !ok {
		t.Fatalf("deleting an unused file should pass: %s", detail)
	}
}
