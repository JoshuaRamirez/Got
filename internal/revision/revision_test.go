package revision_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/revision"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func eid(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

// --- struct-level smoke tests retained from the pre-impl placeholder ---

func TestTransformKindStringForm(t *testing.T) {
	var k revision.TransformKind = "merge-pushout"
	if string(k) != "merge-pushout" {
		t.Fatal("TransformKind string conversion broken")
	}
}

func TestChangeCapsuleZeroValue(t *testing.T) {
	var c revision.ChangeCapsule
	if c.Consumed != nil || c.Produced != nil {
		t.Fatal("zero-value ChangeCapsule should have nil slices")
	}
}

func TestSentinels(t *testing.T) {
	for _, e := range []error{revision.ErrSideConditionFailed, revision.ErrNoMatch} {
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}

// --- helpers for behavioral tests ---

// inlineSubgraph implements graph.Subgraph for test fixtures.
type inlineSubgraph struct {
	ids    []identity.VertexID
	verts  []graph.Vertex
	edges  []graph.Edge
	hypers []graph.Hyperedge
}

func (s *inlineSubgraph) VertexIDs() []identity.VertexID { return s.ids }
func (s *inlineSubgraph) Vertices() []graph.Vertex       { return s.verts }
func (s *inlineSubgraph) Edges() []graph.Edge            { return s.edges }
func (s *inlineSubgraph) Hyperedges() []graph.Hyperedge  { return s.hypers }

// testRule is a Rule built from inline subgraphs.
type testRule struct {
	left, ctx, right graph.Subgraph
	preds            []revision.Predicate
}

func (r testRule) Left() graph.Subgraph    { return r.left }
func (r testRule) Context() graph.Subgraph { return r.ctx }
func (r testRule) Right() graph.Subgraph   { return r.right }
func (r testRule) SideConditions() []revision.Predicate {
	return r.preds
}

// testMatch is a Match built from a plain map.
type testMatch struct {
	m map[identity.VertexID]identity.VertexID
}

func (m testMatch) Mapping() map[identity.VertexID]identity.VertexID { return m.m }

// testPredicate is a Predicate built from a closure.
type testPredicate struct {
	fn func(graph.Graph, revision.Match) error
}

func (p testPredicate) Check(g graph.Graph, m revision.Match) error { return p.fn(g, m) }

// --- behavioral tests ---

// Add-vertex rule: L = ∅, K = ∅, R = {newV}. Result: graph gains newV.
func TestApplyAddVertex(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	newV := graph.Vertex{ID: vid("new"), Type: ontology.Artifact}
	rule := testRule{
		left:  &inlineSubgraph{},
		ctx:   &inlineSubgraph{},
		right: &inlineSubgraph{ids: []identity.VertexID{newV.ID}, verts: []graph.Vertex{newV}},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{}}

	e := revision.NewEngine()
	g2, capsule, err := e.Apply(ctx, g, rule, match)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g2.Vertex(newV.ID); !ok {
		t.Fatal("expected new vertex to be present in result graph")
	}
	if len(capsule.Consumed) != 0 || len(capsule.Produced) != 1 || capsule.Produced[0] != newV.ID {
		t.Fatalf("capsule = %+v", capsule)
	}
}

// Add-edge rule: L = {a, b}, K = {a, b}, R = {a, b, edge(a→b)}.
func TestApplyAddEdge(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	b := vid("b")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: a, Type: ontology.Execution})
	g, _ = g.WithVertex(graph.Vertex{ID: b, Type: ontology.Model})

	verts := []graph.Vertex{
		{ID: a, Type: ontology.Execution},
		{ID: b, Type: ontology.Model},
	}
	leftSub := &inlineSubgraph{ids: []identity.VertexID{a, b}, verts: verts}
	ctxSub := &inlineSubgraph{ids: []identity.VertexID{a, b}, verts: verts}

	newEdge := graph.Edge{ID: eid("e"), Type: ontology.Executes, From: a, To: b}
	rightSub := &inlineSubgraph{
		ids:   []identity.VertexID{a, b},
		verts: verts,
		edges: []graph.Edge{newEdge},
	}
	rule := testRule{left: leftSub, ctx: ctxSub, right: rightSub}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{a: a, b: b}}

	e := revision.NewEngine()
	g2, _, err := e.Apply(ctx, g, rule, match)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g2.Edge(newEdge.ID); !ok {
		t.Fatal("expected new edge to be present in result graph")
	}
}

// Delete-vertex rule: L = {x}, K = ∅, R = ∅. Result: graph loses x.
func TestApplyDeleteVertex(t *testing.T) {
	ctx := context.Background()
	x := vid("x")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: x, Type: ontology.Artifact})

	rule := testRule{
		left:  &inlineSubgraph{ids: []identity.VertexID{x}, verts: []graph.Vertex{{ID: x, Type: ontology.Artifact}}},
		ctx:   &inlineSubgraph{},
		right: &inlineSubgraph{},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{x: x}}

	e := revision.NewEngine()
	g2, capsule, err := e.Apply(ctx, g, rule, match)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g2.Vertex(x); ok {
		t.Fatal("expected vertex x to be deleted")
	}
	if len(capsule.Consumed) != 1 || capsule.Consumed[0] != x {
		t.Fatalf("capsule.Consumed = %v, want [%v]", capsule.Consumed, x)
	}
}

// Failure: match references a pattern vertex not in the host graph.
func TestApplyMatchNotFound(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	x := vid("x")
	rule := testRule{
		left:  &inlineSubgraph{ids: []identity.VertexID{x}},
		ctx:   &inlineSubgraph{},
		right: &inlineSubgraph{},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{x: vid("ghost")}}

	e := revision.NewEngine()
	_, _, err := e.Apply(ctx, g, rule, match)
	if !errors.Is(err, revision.ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

// Failure: side condition predicate returns error.
func TestApplySideConditionFails(t *testing.T) {
	ctx := context.Background()
	x := vid("x")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: x, Type: ontology.Artifact})

	rule := testRule{
		left:  &inlineSubgraph{ids: []identity.VertexID{x}, verts: []graph.Vertex{{ID: x, Type: ontology.Artifact}}},
		ctx:   &inlineSubgraph{ids: []identity.VertexID{x}, verts: []graph.Vertex{{ID: x, Type: ontology.Artifact}}},
		right: &inlineSubgraph{ids: []identity.VertexID{x}, verts: []graph.Vertex{{ID: x, Type: ontology.Artifact}}},
		preds: []revision.Predicate{testPredicate{fn: func(graph.Graph, revision.Match) error {
			return errors.New("forbidden")
		}}},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{x: x}}

	e := revision.NewEngine()
	_, _, err := e.Apply(ctx, g, rule, match)
	if !errors.Is(err, revision.ErrSideConditionFailed) {
		t.Fatalf("expected ErrSideConditionFailed, got %v", err)
	}
}

// Failure: ctx cancelled before work.
func TestApplyContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	rule := testRule{left: &inlineSubgraph{}, ctx: &inlineSubgraph{}, right: &inlineSubgraph{}}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{}}

	e := revision.NewEngine()
	_, _, err := e.Apply(ctx, g, rule, match)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// Failure path UC-S02 4a: inserting an edge whose type is not admissible
// for the endpoint vertex types fails graph.Validate after the rewrite.
func TestApplyInsertViolatesSchema(t *testing.T) {
	ctx := context.Background()
	agent := vid("agent")
	other := vid("other-agent")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: agent, Type: ontology.Agent})
	g, _ = g.WithVertex(graph.Vertex{ID: other, Type: ontology.Agent})

	// Right-hand side wants to add an AuthoredBy edge from Agent to Agent
	// — not admissible (AuthoredBy requires Agent -> Artifact).
	verts := []graph.Vertex{
		{ID: agent, Type: ontology.Agent},
		{ID: other, Type: ontology.Agent},
	}
	leftSub := &inlineSubgraph{ids: []identity.VertexID{agent, other}, verts: verts}
	ctxSub := &inlineSubgraph{ids: []identity.VertexID{agent, other}, verts: verts}
	badEdge := graph.Edge{ID: eid("bad"), Type: ontology.AuthoredBy, From: agent, To: other}
	rightSub := &inlineSubgraph{
		ids:   []identity.VertexID{agent, other},
		verts: verts,
		edges: []graph.Edge{badEdge},
	}
	rule := testRule{left: leftSub, ctx: ctxSub, right: rightSub}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{agent: agent, other: other}}

	e := revision.NewEngine()
	_, _, err := e.Apply(ctx, g, rule, match)
	if !errors.Is(err, graph.ErrNotWellFormed) {
		t.Fatalf("expected graph.ErrNotWellFormed for schema-violating insertion, got %v", err)
	}
}

// Replayable: all vertices present → nil.
func TestReplayableHappyPath(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	b := vid("b")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: a, Type: ontology.Artifact})
	g, _ = g.WithVertex(graph.Vertex{ID: b, Type: ontology.Artifact})

	capsule := revision.ChangeCapsule{
		Consumed: []identity.VertexID{a},
		Produced: []identity.VertexID{b},
	}
	e := revision.NewEngine()
	if err := e.Replayable(ctx, g, capsule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Replayable: consumed vertex missing → ErrNoMatch.
func TestReplayableConsumedMissing(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	capsule := revision.ChangeCapsule{
		Consumed: []identity.VertexID{vid("missing")},
	}
	e := revision.NewEngine()
	err := e.Replayable(ctx, g, capsule)
	if !errors.Is(err, revision.ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

// Replayable: produced vertex missing → ErrNoMatch.
func TestReplayableProducedMissing(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	capsule := revision.ChangeCapsule{
		Produced: []identity.VertexID{vid("missing")},
	}
	e := revision.NewEngine()
	err := e.Replayable(ctx, g, capsule)
	if !errors.Is(err, revision.ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

// Replayable: empty capsule → nil.
func TestReplayableEmptyCapsule(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	e := revision.NewEngine()
	if err := e.Replayable(ctx, g, revision.ChangeCapsule{}); err != nil {
		t.Fatalf("empty capsule should be replayable, got %v", err)
	}
}

// --- Strict mode (dangling-edge detection) ---

// Lenient mode (default) silently drops orphaned edges when a vertex is
// deleted; the rewrite succeeds.
func TestApplyLenientDropsDanglingEdges(t *testing.T) {
	ctx := context.Background()
	x := vid("strict-x")
	y := vid("strict-y")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: x, Type: ontology.Artifact})
	g, _ = g.WithVertex(graph.Vertex{ID: y, Type: ontology.Revision})
	g, _ = g.WithEdge(graph.Edge{ID: eid("strict-e"), Type: ontology.DerivedFrom, From: y, To: x})

	// Delete x; the edge y->x becomes dangling.
	rule := testRule{
		left:  &inlineSubgraph{ids: []identity.VertexID{x}, verts: []graph.Vertex{{ID: x, Type: ontology.Artifact}}},
		ctx:   &inlineSubgraph{},
		right: &inlineSubgraph{},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{x: x}}

	e := revision.NewEngine() // Lenient
	g2, _, err := e.Apply(ctx, g, rule, match)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g2.Edge(eid("strict-e")); ok {
		t.Fatal("Lenient should silently drop the dangling edge")
	}
}

// Strict mode refuses the rewrite when a deletion would orphan an edge.
func TestApplyStrictRefusesDanglingEdge(t *testing.T) {
	ctx := context.Background()
	x := vid("strict-refuse-x")
	y := vid("strict-refuse-y")
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: x, Type: ontology.Artifact})
	g, _ = g.WithVertex(graph.Vertex{ID: y, Type: ontology.Revision})
	g, _ = g.WithEdge(graph.Edge{ID: eid("strict-refuse-e"), Type: ontology.DerivedFrom, From: y, To: x})

	rule := testRule{
		left:  &inlineSubgraph{ids: []identity.VertexID{x}, verts: []graph.Vertex{{ID: x, Type: ontology.Artifact}}},
		ctx:   &inlineSubgraph{},
		right: &inlineSubgraph{},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{x: x}}

	e := revision.NewEngineStrict()
	_, _, err := e.Apply(ctx, g, rule, match)
	if !errors.Is(err, revision.ErrDanglingEdge) {
		t.Fatalf("expected ErrDanglingEdge, got %v", err)
	}
}

// Strict mode allows the rewrite when no dangling edge would result
// (deletion is "clean" — all incident edges are also in L\K).
func TestApplyStrictAllowsCleanDeletion(t *testing.T) {
	ctx := context.Background()
	x := vid("strict-clean-x")
	y := vid("strict-clean-y")
	edge := graph.Edge{ID: eid("strict-clean-e"), Type: ontology.DerivedFrom, From: y, To: x}
	g := graph.NewGraph(ontology.NewDefaultSchema())
	g, _ = g.WithVertex(graph.Vertex{ID: x, Type: ontology.Artifact})
	g, _ = g.WithVertex(graph.Vertex{ID: y, Type: ontology.Revision})
	g, _ = g.WithEdge(edge)

	// Delete x AND the edge in the same rewrite.
	rule := testRule{
		left: &inlineSubgraph{
			ids:   []identity.VertexID{x},
			verts: []graph.Vertex{{ID: x, Type: ontology.Artifact}},
			edges: []graph.Edge{edge},
		},
		ctx:   &inlineSubgraph{},
		right: &inlineSubgraph{},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{x: x}}

	e := revision.NewEngineStrict()
	g2, _, err := e.Apply(ctx, g, rule, match)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g2.Vertex(x); ok {
		t.Fatal("x should be deleted")
	}
	if _, ok := g2.Edge(edge.ID); ok {
		t.Fatal("edge should be deleted")
	}
}

// Strict mode allows pure additions (no deletions, no dangling risk).
func TestApplyStrictAllowsPureAddition(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())

	newV := graph.Vertex{ID: vid("strict-add"), Type: ontology.Artifact}
	rule := testRule{
		left:  &inlineSubgraph{},
		ctx:   &inlineSubgraph{},
		right: &inlineSubgraph{ids: []identity.VertexID{newV.ID}, verts: []graph.Vertex{newV}},
	}
	match := testMatch{m: map[identity.VertexID]identity.VertexID{}}

	e := revision.NewEngineStrict()
	g2, _, err := e.Apply(ctx, g, rule, match)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g2.Vertex(newV.ID); !ok {
		t.Fatal("pure addition should succeed in Strict mode")
	}
}

func TestStrictnessAccessor(t *testing.T) {
	if revision.NewEngine().(interface{ Strictness() revision.Strictness }).Strictness() != revision.Lenient {
		t.Fatal("NewEngine should default to Lenient")
	}
	if revision.NewEngineStrict().(interface{ Strictness() revision.Strictness }).Strictness() != revision.Strict {
		t.Fatal("NewEngineStrict should be Strict")
	}
}

func TestErrDanglingEdgeSentinel(t *testing.T) {
	if !errors.Is(revision.ErrDanglingEdge, revision.ErrDanglingEdge) {
		t.Fatal("ErrDanglingEdge must match itself")
	}
}
