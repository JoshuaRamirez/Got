package projection_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func smallGraph(t *testing.T) (graph.Graph, identity.VertexID, identity.VertexID) {
	t.Helper()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	a := vid("artifact")
	b := vid("revision")
	g, _ = g.WithVertex(graph.Vertex{ID: a, Type: ontology.Artifact})
	g, _ = g.WithVertex(graph.Vertex{ID: b, Type: ontology.Revision})
	return g, a, b
}

func TestErrInvalidSelectorSentinel(t *testing.T) {
	if !errors.Is(projection.ErrInvalidSelector, projection.ErrInvalidSelector) {
		t.Fatal("sentinel must match itself")
	}
}

// Main path: Select returns a Frontier whose IDs come from the selector.
func TestSelectMainPath(t *testing.T) {
	ctx := context.Background()
	g, a, b := smallGraph(t)

	e := projection.NewEngine()
	f, err := e.Select(ctx, g, projection.IDsSelector{IDs: []identity.VertexID{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	got := f.VertexIDs()
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("Frontier.VertexIDs = %v, want [%v %v]", got, a, b)
	}
}

// Empty selector → empty Frontier (success path).
func TestSelectEmpty(t *testing.T) {
	ctx := context.Background()
	g, _, _ := smallGraph(t)

	e := projection.NewEngine()
	f, err := e.Select(ctx, g, projection.IDsSelector{IDs: nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.VertexIDs()) != 0 {
		t.Fatal("empty selector should produce empty Frontier")
	}
}

// Failure path: selector returns an ID not in the graph.
func TestSelectIDNotInGraph(t *testing.T) {
	ctx := context.Background()
	g, a, _ := smallGraph(t)

	e := projection.NewEngine()
	ghost := vid("ghost")
	_, err := e.Select(ctx, g, projection.IDsSelector{IDs: []identity.VertexID{a, ghost}})
	if !errors.Is(err, projection.ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector, got %v", err)
	}
}

// Failure path: selector itself errors.
type erroringSelector struct{}

func (erroringSelector) Frontier(graph.Graph) ([]identity.VertexID, error) {
	return nil, errors.New("selector exploded")
}

func TestSelectSelectorErrors(t *testing.T) {
	ctx := context.Background()
	g, _, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Select(ctx, g, erroringSelector{})
	if !errors.Is(err, projection.ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector wrap, got %v", err)
	}
}

// Failure path: ctx cancelled before work.
func TestSelectContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, a, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Select(ctx, g, projection.IDsSelector{IDs: []identity.VertexID{a}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// Main path: Project produces a View whose Subgraph contains the requested vertices.
func TestProjectMainPath(t *testing.T) {
	ctx := context.Background()
	g, a, b := smallGraph(t)

	e := projection.NewEngine()
	v, err := e.Project(ctx, g, projection.InduceSpec{IDs: []identity.VertexID{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	sub := v.Subgraph()
	if len(sub.VertexIDs()) != 2 {
		t.Fatalf("expected 2 vertices in view, got %d", len(sub.VertexIDs()))
	}
}

// Failure path: spec apply errors (vertex not in graph).
func TestProjectSpecErrors(t *testing.T) {
	ctx := context.Background()
	g, _, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Project(ctx, g, projection.InduceSpec{IDs: []identity.VertexID{vid("ghost")}})
	if !errors.Is(err, projection.ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector wrap, got %v", err)
	}
}

// Failure path: ctx cancelled.
func TestProjectContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, a, _ := smallGraph(t)

	e := projection.NewEngine()
	_, err := e.Project(ctx, g, projection.InduceSpec{IDs: []identity.VertexID{a}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// --- EditedFrontier.Clone ---

// Clone produces an independent copy: mutating the original after Clone
// does not affect the clone.
func TestEditedFrontierCloneIndependence(t *testing.T) {
	a := vid("clone-a")
	b := vid("clone-b")

	orig := projection.NewEditedFrontier([]identity.VertexID{a, b})
	orig.Vertices[a] = graph.Vertex{
		ID: a, Type: ontology.Artifact,
		Attrs: graph.AttrMap{"status": "draft"},
	}
	orig.Edges[identity.EdgeID(sha256.Sum256([]byte("clone-e")))] = graph.Edge{
		Type: ontology.AuthoredBy,
		From: a, To: b,
		Attrs: graph.AttrMap{"weight": 1},
	}

	cl := orig.Clone()

	// Mutate every layer of the original.
	orig.IDs = append(orig.IDs, vid("extra"))
	orig.Vertices[a] = graph.Vertex{ID: a, Type: ontology.Revision}
	orig.Vertices[vid("new")] = graph.Vertex{ID: vid("new"), Type: ontology.Artifact}
	for _, e := range orig.Edges {
		delete(orig.Edges, e.ID) // also exercises a different mutation shape
	}

	if len(cl.IDs) != 2 {
		t.Fatalf("clone.IDs leaked from original: %v", cl.IDs)
	}
	if cl.Vertices[a].Type != ontology.Artifact {
		t.Fatalf("clone.Vertices[a].Type leaked from original mutation: %v", cl.Vertices[a].Type)
	}
	if _, ok := cl.Vertices[vid("new")]; ok {
		t.Fatal("clone.Vertices saw an insertion done on the original")
	}
	if len(cl.Edges) != 1 {
		t.Fatalf("clone.Edges should still have 1 entry, got %d", len(cl.Edges))
	}
}

// Clone deep-copies Attrs so mutating clone's Attrs does not affect the
// original's.
func TestEditedFrontierCloneAttrsIndependence(t *testing.T) {
	a := vid("clone-attrs-a")
	orig := projection.NewEditedFrontier([]identity.VertexID{a})
	orig.Vertices[a] = graph.Vertex{
		ID: a, Type: ontology.Artifact,
		Attrs: graph.AttrMap{"status": "draft"},
	}

	cl := orig.Clone()

	// Mutate clone's Attrs.
	cv := cl.Vertices[a]
	cv.Attrs["status"] = "review"
	cl.Vertices[a] = cv

	if orig.Vertices[a].Attrs["status"] != "draft" {
		t.Fatalf("original Attrs leaked clone's mutation: %v", orig.Vertices[a].Attrs)
	}
	if cl.Vertices[a].Attrs["status"] != "review" {
		t.Fatalf("clone Attrs did not retain its own mutation: %v", cl.Vertices[a].Attrs)
	}
}

// Clone satisfies Frontier and Edited.
func TestEditedFrontierCloneInterfaceSatisfaction(t *testing.T) {
	orig := projection.NewEditedFrontier([]identity.VertexID{vid("iface")})
	cl := orig.Clone()
	var _ projection.Frontier = cl
	var _ projection.Edited = cl
}

// Clone on nil returns nil.
func TestEditedFrontierCloneNil(t *testing.T) {
	var f *projection.EditedFrontier
	if got := f.Clone(); got != nil {
		t.Fatalf("Clone on nil should return nil, got %+v", got)
	}
}

// Clone on an empty frontier produces an empty clone with non-nil maps.
func TestEditedFrontierCloneEmpty(t *testing.T) {
	orig := projection.NewEditedFrontier(nil)
	cl := orig.Clone()
	if cl == nil {
		t.Fatal("Clone of empty frontier should not be nil")
	}
	if cl.Vertices == nil || cl.Edges == nil {
		t.Fatal("Clone should preserve non-nil empty maps")
	}
	if len(cl.IDs) != 0 || len(cl.Vertices) != 0 || len(cl.Edges) != 0 {
		t.Fatalf("Clone of empty frontier should be empty, got %+v", cl)
	}
}

// Clone preserves vertex content for use after the original frontier is
// modified by a resolver — the motivating use case for the method.
func TestEditedFrontierCloneSupportsResolverRetry(t *testing.T) {
	a := vid("clone-retry-a")
	orig := projection.NewEditedFrontier([]identity.VertexID{a})
	orig.Vertices[a] = graph.Vertex{
		ID: a, Type: ontology.Artifact,
		Attrs: graph.AttrMap{"status": "draft"},
	}

	snapshot := orig.Clone()

	// Simulate what PreferLeftAttr would do to the right frontier in
	// ResolveTyped: overwrite the Attrs entry.
	mv := orig.Vertices[a]
	mv.Attrs["status"] = "OVERWRITTEN"
	orig.Vertices[a] = mv

	if snapshot.Vertices[a].Attrs["status"] != "draft" {
		t.Fatalf("snapshot should retain original draft after orig mutation, got %v",
			snapshot.Vertices[a].Attrs)
	}
}
