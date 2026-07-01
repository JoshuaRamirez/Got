package composition_test

import (
	"context"
	"testing"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// editedFrontier builds an EditedFrontier whose membership is the IDs of the
// supplied vertices, each carrying its content in the edit map.
func editedFrontier(verts ...graph.Vertex) *projection.EditedFrontier {
	ids := make([]identity.VertexID, 0, len(verts))
	for _, v := range verts {
		ids = append(ids, v.ID)
	}
	f := projection.NewEditedFrontier(ids)
	for _, v := range verts {
		f.Vertices[v.ID] = v
	}
	return f
}

// vtx is a small helper for an Artifact vertex with one Attrs key.
func vtxAttr(id identity.VertexID, key string, val any) graph.Vertex {
	return graph.Vertex{ID: id, Type: ontology.Artifact, Attrs: graph.AttrMap{key: val}}
}

func threeWayEngine(t *testing.T) *composition.DefaultEngine {
	t.Helper()
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, nil)
	return composition.NewEngine(gov, ver)
}

// Compile-time + runtime assertion that *DefaultEngine satisfies the optional
// ThreeWayMerger capability.
func TestThreeWayMergerCapability(t *testing.T) {
	var e composition.Engine = threeWayEngine(t)
	if _, ok := e.(composition.ThreeWayMerger); !ok {
		t.Fatal("*DefaultEngine should satisfy composition.ThreeWayMerger")
	}
}

// Only the left side changed a vertex relative to the ancestor → take left.
func TestMergeThreeWayOnlyLeftChanged(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")

	ancestor := editedFrontier(vtxAttr(id, "x", 1))
	left := editedFrontier(vtxAttr(id, "x", 2))  // changed
	right := editedFrontier(vtxAttr(id, "x", 1)) // unchanged

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(mr.Conflicts))
	}
	if mr.Frontier == nil {
		t.Fatal("expected a merged frontier")
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if got := ed.Vertices[id].Attrs["x"]; got != 2 {
		t.Fatalf("expected left's value x=2 to win, got %v", got)
	}
	if mr.Certificate == nil {
		t.Fatal("expected a certificate on a clean three-way merge")
	}
}

// Only the right side changed → take right.
func TestMergeThreeWayOnlyRightChanged(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")

	ancestor := editedFrontier(vtxAttr(id, "x", 1))
	left := editedFrontier(vtxAttr(id, "x", 1))  // unchanged
	right := editedFrontier(vtxAttr(id, "x", 9)) // changed

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(mr.Conflicts))
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if got := ed.Vertices[id].Attrs["x"]; got != 9 {
		t.Fatalf("expected right's value x=9 to win, got %v", got)
	}
}

// Both sides made the identical change → take it, no conflict.
func TestMergeThreeWayBothSameChange(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")

	ancestor := editedFrontier(vtxAttr(id, "x", 1))
	left := editedFrontier(vtxAttr(id, "x", 5))
	right := editedFrontier(vtxAttr(id, "x", 5))

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts for an agreed change, got %d", len(mr.Conflicts))
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if got := ed.Vertices[id].Attrs["x"]; got != 5 {
		t.Fatalf("expected agreed value x=5, got %v", got)
	}
}

// Both sides changed the same attr to different values → Textual conflict.
func TestMergeThreeWayModifyModifyConflict(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")

	ancestor := editedFrontier(vtxAttr(id, "x", 1))
	left := editedFrontier(vtxAttr(id, "x", 2))
	right := editedFrontier(vtxAttr(id, "x", 3))

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("modify/modify conflict must not produce a merged frontier")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Textual {
		t.Fatalf("expected one Textual conflict, got %+v", mr.Conflicts)
	}
}

// Both sides changed the vertex type differently → Schema conflict (classified
// by the first differing dimension).
func TestMergeThreeWaySchemaConflict(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")

	ancestor := editedFrontier(graph.Vertex{ID: id, Type: ontology.Artifact})
	left := editedFrontier(graph.Vertex{ID: id, Type: ontology.Model})
	right := editedFrontier(graph.Vertex{ID: id, Type: ontology.Tool})

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("schema conflict must not produce a merged frontier")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Schema {
		t.Fatalf("expected one Schema conflict, got %+v", mr.Conflicts)
	}
}

// A vertex added only on the left (absent from ancestor and right) is included.
func TestMergeThreeWayLeftAddition(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	base := vid("base")
	added := vid("added")

	ancestor := editedFrontier(vtxAttr(base, "x", 1))
	left := editedFrontier(vtxAttr(base, "x", 1), vtxAttr(added, "y", 7))
	right := editedFrontier(vtxAttr(base, "x", 1))

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("addition should not conflict, got %d conflicts", len(mr.Conflicts))
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if _, ok := ed.Vertices[added]; !ok {
		t.Fatal("expected the left-side addition to be present")
	}
}

// A vertex added on both sides with different content → add/add conflict.
func TestMergeThreeWayAddAddConflict(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	added := vid("added")

	ancestor := editedFrontier() // empty ancestor
	left := editedFrontier(vtxAttr(added, "y", 1))
	right := editedFrontier(vtxAttr(added, "y", 2))

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("add/add conflict must not produce a merged frontier")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Textual {
		t.Fatalf("expected one Textual add/add conflict, got %+v", mr.Conflicts)
	}
}

// A deletion on one side, with the other side unchanged, is honored.
func TestMergeThreeWayDeletionHonored(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	keep := vid("keep")
	gone := vid("gone")

	ancestor := editedFrontier(vtxAttr(keep, "x", 1), vtxAttr(gone, "x", 1))
	left := editedFrontier(vtxAttr(keep, "x", 1))                         // deleted `gone`
	right := editedFrontier(vtxAttr(keep, "x", 1), vtxAttr(gone, "x", 1)) // unchanged

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("honored deletion should not conflict, got %d", len(mr.Conflicts))
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if _, ok := ed.Vertices[gone]; ok {
		t.Fatal("deleted vertex must not reappear in the merged frontier")
	}
	if _, ok := ed.Vertices[keep]; !ok {
		t.Fatal("retained vertex should be present")
	}
}

// Deletion on one side, modification on the other → modify/delete conflict.
func TestMergeThreeWayModifyDeleteConflict(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")

	ancestor := editedFrontier(vtxAttr(id, "x", 1))
	left := editedFrontier()                     // deleted
	right := editedFrontier(vtxAttr(id, "x", 2)) // modified

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("modify/delete conflict must not produce a merged frontier")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Structural {
		t.Fatalf("expected one Structural modify/delete conflict, got %+v", mr.Conflicts)
	}
}

// Both sides delete the same vertex → omitted, no conflict.
func TestMergeThreeWayBothDelete(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	keep := vid("keep")
	gone := vid("gone")

	ancestor := editedFrontier(vtxAttr(keep, "x", 1), vtxAttr(gone, "x", 1))
	left := editedFrontier(vtxAttr(keep, "x", 1))
	right := editedFrontier(vtxAttr(keep, "x", 1))

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("agreed deletion should not conflict, got %d", len(mr.Conflicts))
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if _, ok := ed.Vertices[gone]; ok {
		t.Fatal("vertex deleted on both sides must be omitted")
	}
}

// Plain (non-Edited) frontiers degrade to presence-only three-way: a deletion
// is still honored because all three sides read identical content from g, so
// the kept side counts as "unchanged".
func TestMergeThreeWayPlainFrontiersHonorDeletion(t *testing.T) {
	ctx := context.Background()
	a := vid("a")
	b := vid("b")
	g := graphWith(t, a, b)
	e, pe, _ := newEngines(t)

	ancestor := makeFrontier(t, pe, g, a, b)
	left := makeFrontier(t, pe, g, a)     // deleted b
	right := makeFrontier(t, pe, g, a, b) // unchanged

	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("plain-frontier deletion should be honored without conflict, got %d", len(mr.Conflicts))
	}
	ids := mr.Frontier.VertexIDs()
	if len(ids) != 1 || ids[0] != a {
		t.Fatalf("expected merged frontier {a}, got %v", ids)
	}
}

// Unsat policy gate blocks the merge with a Policy conflict.
func TestMergeThreeWayUnsatPolicyBlocks(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")

	ancestor := editedFrontier(vtxAttr(id, "x", 1))
	left := editedFrontier(vtxAttr(id, "x", 2))
	right := editedFrontier(vtxAttr(id, "x", 1))

	e := threeWayEngine(t)
	mr, err := e.MergeThreeWay(ctx, g, ancestor, left, right,
		[]governance.Policy{fixedPolicy{name: "deny", d: governance.Unsat}})
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("Unsat policy must block the merge")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Policy {
		t.Fatalf("expected one Policy conflict, got %+v", mr.Conflicts)
	}
}

// ctx cancellation is honored.
func TestMergeThreeWayContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	id := vid("v")
	f := editedFrontier(vtxAttr(id, "x", 1))

	e := threeWayEngine(t)
	_, err := e.MergeThreeWay(ctx, g, f, f, f, nil)
	if err == nil {
		t.Fatal("expected ctx cancellation error")
	}
}

// --- edge-level three-way (PR D) ---

// edgedFrontier builds an EditedFrontier carrying the given edges (and their
// endpoint vertices as members).
func edgedFrontier(edges ...graph.Edge) *projection.EditedFrontier {
	seen := map[identity.VertexID]bool{}
	var ids []identity.VertexID
	for _, e := range edges {
		for _, v := range []identity.VertexID{e.From, e.To} {
			if !seen[v] {
				seen[v] = true
				ids = append(ids, v)
			}
		}
	}
	f := projection.NewEditedFrontier(ids)
	for _, e := range edges {
		f.Edges[e.ID] = e
	}
	return f
}

func edge(name string, typ ontology.EdgeType, from, to string) graph.Edge {
	return graph.Edge{ID: eid(name), Type: typ, From: vid(from), To: vid(to)}
}

// Only the left side changed an edge → take left, no conflict; merged carries it.
func TestMergeThreeWayEdgeOnlyLeftChanged(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	base := edge("e", ontology.DerivedFrom, "a", "b")
	changed := base
	changed.Attrs = graph.AttrMap{"note": "left"}

	ancestor := edgedFrontier(base)
	left := edgedFrontier(changed) // changed attrs
	right := edgedFrontier(base)   // unchanged

	mr, err := threeWayEngine(t).MergeThreeWay(ctx, g, ancestor, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", mr.Conflicts)
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if got := ed.Edges[base.ID].Attrs["note"]; got != "left" {
		t.Fatalf("expected left's edge change to win, got %v", got)
	}
}

// Both sides changed the same edge differently → Structural conflict.
func TestMergeThreeWayEdgeModifyModify(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	base := edge("e", ontology.DerivedFrom, "a", "b")
	l := base
	l.To = vid("c")
	r := base
	r.To = vid("d")

	mr, err := threeWayEngine(t).MergeThreeWay(ctx, g, edgedFrontier(base), edgedFrontier(l), edgedFrontier(r), nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("edge modify/modify must not produce a merged frontier")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Structural {
		t.Fatalf("expected one Structural edge conflict, got %+v", mr.Conflicts)
	}
}

// An edge added only on the left is included.
func TestMergeThreeWayEdgeAddition(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	added := edge("e-add", ontology.DerivedFrom, "a", "b")

	mr, err := threeWayEngine(t).MergeThreeWay(ctx, g, edgedFrontier(), edgedFrontier(added), edgedFrontier(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("edge addition should not conflict, got %v", mr.Conflicts)
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if _, ok := ed.Edges[added.ID]; !ok {
		t.Fatal("expected the added edge in the merged frontier")
	}
}

// Edge deleted on one side, unchanged on the other → deletion honored.
func TestMergeThreeWayEdgeDeletionHonored(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	e := edge("e", ontology.DerivedFrom, "a", "b")

	mr, err := threeWayEngine(t).MergeThreeWay(ctx, g, edgedFrontier(e), edgedFrontier(), edgedFrontier(e), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("honored edge deletion should not conflict, got %v", mr.Conflicts)
	}
	ed := mr.Frontier.(*projection.EditedFrontier)
	if _, ok := ed.Edges[e.ID]; ok {
		t.Fatal("deleted edge must not reappear")
	}
}

// Edge deleted on one side, modified on the other → Structural modify/delete.
func TestMergeThreeWayEdgeModifyDelete(t *testing.T) {
	ctx := context.Background()
	g := graph.NewGraph(ontology.NewDefaultSchema())
	base := edge("e", ontology.DerivedFrom, "a", "b")
	modified := base
	modified.To = vid("c")

	mr, err := threeWayEngine(t).MergeThreeWay(ctx, g, edgedFrontier(base), edgedFrontier(), edgedFrontier(modified), nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("edge modify/delete must not produce a merged frontier")
	}
	if len(mr.Conflicts) != 1 || mr.Conflicts[0].Kind() != composition.Structural {
		t.Fatalf("expected one Structural modify/delete conflict, got %+v", mr.Conflicts)
	}
}
