package realization_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/realization"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func TestTargetType(t *testing.T) {
	var tgt realization.Target = "html-bundle-v1"
	if string(tgt) != "html-bundle-v1" {
		t.Fatal("Target string conversion broken")
	}
}

func TestFidelityContractStruct(t *testing.T) {
	fc := realization.FidelityContract{Name: "lossless"}
	if fc.Name != "lossless" {
		t.Fatal("FidelityContract.Name round-trip failed")
	}
}

func TestErrTargetUnsupportedSentinel(t *testing.T) {
	if !errors.Is(realization.ErrTargetUnsupported, realization.ErrTargetUnsupported) {
		t.Fatal("sentinel must match itself")
	}
}

// --- helpers ---

func viewOver(t *testing.T, ids ...identity.VertexID) projection.View {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	for _, id := range ids {
		var err error
		g, err = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})
		if err != nil {
			t.Fatal(err)
		}
	}
	pe := projection.NewEngine()
	v, err := pe.Project(context.Background(), g, projection.InduceSpec{IDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// --- behavioral tests ---

// Main path: ManifestTarget emits one path per vertex with provenance.
func TestManifestMaterialize(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	b := vid("b")
	v := viewOver(t, a, b)

	e := realization.NewEngine()
	bundle, err := e.Materialize(ctx, v, realization.ManifestTarget)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Target() != realization.ManifestTarget {
		t.Fatalf("Bundle.Target = %v, want %v", bundle.Target(), realization.ManifestTarget)
	}
	paths := bundle.Paths()
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	for _, p := range paths {
		prov := bundle.Provenance(p)
		if len(prov) != 1 {
			t.Fatalf("path %q has %d provenance entries, want 1", p, len(prov))
		}
	}
	if bundle.Fidelity().Name == "" {
		t.Fatal("Fidelity contract should have a name")
	}
}

// Main path: empty view → empty bundle, no error.
func TestManifestEmpty(t *testing.T) {
	ctx := context.Background()
	v := viewOver(t)

	e := realization.NewEngine()
	bundle, err := e.Materialize(ctx, v, realization.ManifestTarget)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Paths()) != 0 {
		t.Fatalf("expected empty bundle, got %d paths", len(bundle.Paths()))
	}
}

// Failure: target has no registered materializer.
func TestMaterializeUnsupportedTarget(t *testing.T) {
	ctx := context.Background()
	v := viewOver(t, vid("a"))

	e := realization.NewEngine()
	_, err := e.Materialize(ctx, v, realization.Target("html"))
	if !errors.Is(err, realization.ErrTargetUnsupported) {
		t.Fatalf("expected ErrTargetUnsupported, got %v", err)
	}
}

// Main path: a custom Materializer registered for a new Target.
func TestRegisterCustomMaterializer(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	v := viewOver(t, a)

	called := false
	e := realization.NewEngine()
	e.Register("counter", realization.MaterializerFunc(func(s graph.Subgraph) (realization.Bundle, error) {
		called = true
		return &countBundle{n: len(s.VertexIDs())}, nil
	}))
	bundle, err := e.Materialize(ctx, v, "counter")
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("custom materializer was not invoked")
	}
	if bundle.Target() != "counter" {
		t.Fatalf("Bundle.Target = %v, want counter", bundle.Target())
	}
}

// Failure: ctx cancelled.
func TestMaterializeContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v := viewOver(t, vid("a"))

	e := realization.NewEngine()
	_, err := e.Materialize(ctx, v, realization.ManifestTarget)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// --- minimal custom Bundle for the register test ---

type countBundle struct{ n int }

func (b *countBundle) Target() realization.Target            { return "counter" }
func (b *countBundle) Paths() []string                       { return []string{"count"} }
func (b *countBundle) Provenance(string) []identity.VertexID { return nil }
func (b *countBundle) Fidelity() realization.FidelityContract {
	return realization.FidelityContract{Name: "count"}
}
