package composition_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

func strictEngine(t *testing.T) *composition.DefaultEngine {
	t.Helper()
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, nil)
	return composition.NewEngineStrict(gov, ver)
}

// Per-side audit conflicts carry typed payloads accessible via the
// Payloaded interface.
func TestPayloadedTextual(t *testing.T) {
	ctx := context.Background()
	id := identity.VertexID(sha256.Sum256([]byte("payload-textual")))

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e := strictEngine(t)
	left := projection.NewEditedFrontier([]identity.VertexID{id})
	left.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}}
	right := projection.NewEditedFrontier([]identity.VertexID{id})
	right.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "review"}}

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range mr.Conflicts {
		if c.Kind() != composition.Textual {
			continue
		}
		pl, ok := c.(composition.Payloaded)
		if !ok {
			t.Fatal("Textual conflict should satisfy Payloaded")
		}
		p, ok := pl.Payload().(composition.TextualPayload)
		if !ok {
			t.Fatalf("expected TextualPayload, got %T", pl.Payload())
		}
		if p.Vertex != id || p.Key != "status" || p.Left != "draft" || p.Right != "review" {
			t.Fatalf("payload = %+v", p)
		}
		found = true
	}
	if !found {
		t.Fatal("Textual conflict not found")
	}
}

// PreferLeftAttr resolves a Textual conflict, then re-merge succeeds.
func TestPreferLeftAttrResolvesTextualConflict(t *testing.T) {
	ctx := context.Background()
	id := identity.VertexID(sha256.Sum256([]byte("resolver-textual")))

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}})

	e := strictEngine(t)
	left := projection.NewEditedFrontier([]identity.VertexID{id})
	left.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}}
	right := projection.NewEditedFrontier([]identity.VertexID{id})
	right.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "review"}}

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Textual) {
		t.Fatalf("expected initial Textual conflict, got %v", mr.Conflicts)
	}

	resolved, err := e.ResolveTyped(ctx, g, left, right, mr, []composition.Resolver{composition.PreferLeftAttr("status")})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Conflicts) != 0 {
		t.Fatalf("expected no conflicts after PreferLeftAttr, got %v", resolved.Conflicts)
	}
}

// PreferHigherTrust picks the higher-Score TrustAnnotation.
func TestPreferHigherTrustResolvesTrustConflict(t *testing.T) {
	ctx := context.Background()
	id := identity.VertexID(sha256.Sum256([]byte("resolver-trust")))

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact, Trust: graph.TrustAnnotation{Score: 50}})

	e := strictEngine(t)
	left := projection.NewEditedFrontier([]identity.VertexID{id})
	left.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Trust: graph.TrustAnnotation{Score: 50}}
	right := projection.NewEditedFrontier([]identity.VertexID{id})
	right.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Trust: graph.TrustAnnotation{Score: 90}}

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(mr.Conflicts, composition.Trust) {
		t.Fatalf("expected initial Trust conflict, got %v", mr.Conflicts)
	}

	resolved, err := e.ResolveTyped(ctx, g, left, right, mr, []composition.Resolver{composition.PreferHigherTrust()})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Conflicts) != 0 {
		t.Fatalf("expected no conflicts after PreferHigherTrust, got %v", resolved.Conflicts)
	}
}

// Unmatched resolvers leave conflicts in place; re-merge reports them.
func TestResolveTypedLeavesUnmatchedConflicts(t *testing.T) {
	ctx := context.Background()
	id := identity.VertexID(sha256.Sum256([]byte("resolver-unmatched")))

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact, Trust: graph.TrustAnnotation{Score: 50}})

	e := strictEngine(t)
	// Produce a Trust conflict, but supply only a Textual resolver.
	left := projection.NewEditedFrontier([]identity.VertexID{id})
	left.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Trust: graph.TrustAnnotation{Score: 50}}
	right := projection.NewEditedFrontier([]identity.VertexID{id})
	right.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Trust: graph.TrustAnnotation{Score: 90}}

	mr, _ := e.Merge(ctx, g, left, right, nil)
	resolved, err := e.ResolveTyped(ctx, g, left, right, mr, []composition.Resolver{composition.PreferLeftAttr("status")})
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(resolved.Conflicts, composition.Trust) {
		t.Fatalf("expected Trust conflict still present, got %v", resolved.Conflicts)
	}
}

// A resolver that returns an error → ErrConflictUnresolvable.
type failingResolver struct{}

func (failingResolver) AppliesTo() composition.ConflictKind { return composition.Textual }
func (failingResolver) Apply(_ context.Context, g graph.Graph, _ composition.Conflict, _, _ *projection.EditedFrontier) (graph.Graph, error) {
	return g, errors.New("resolver exploded")
}

func TestResolveTypedResolverErrors(t *testing.T) {
	ctx := context.Background()
	id := identity.VertexID(sha256.Sum256([]byte("resolver-err")))

	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: id, Type: ontology.Artifact})

	e := strictEngine(t)
	left := projection.NewEditedFrontier([]identity.VertexID{id})
	left.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"k": "a"}}
	right := projection.NewEditedFrontier([]identity.VertexID{id})
	right.Vertices[id] = graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{"k": "b"}}

	mr, _ := e.Merge(ctx, g, left, right, nil)
	_, err := e.ResolveTyped(ctx, g, left, right, mr, []composition.Resolver{failingResolver{}})
	if !errors.Is(err, composition.ErrConflictUnresolvable) {
		t.Fatalf("expected ErrConflictUnresolvable, got %v", err)
	}
}

// ctx cancellation short-circuits ResolveTyped.
func TestResolveTypedContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	e := strictEngine(t)
	empty := projection.NewEditedFrontier(nil)
	_, err := e.ResolveTyped(ctx, g, empty, empty, composition.MergeResult{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
