package graph_test

import (
	"encoding/hex"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

func hexOf(id identity.VertexID) string { return hex.EncodeToString(id[:]) }
func hexOf32(id identity.EdgeID) string { return hex.EncodeToString(id[:]) }

// richGraph builds a valid graph exercising every field: attrs, time, trust,
// edges, and the canonical executes hyperedge.
func richGraph(t *testing.T) graph.Graph {
	t.Helper()
	b := graph.NewBuilder(ontology.NewDefaultSchema())
	add := func(id identity.VertexID, vt ontology.VertexType, v graph.Vertex) {
		v.ID = id
		v.Type = vt
		if err := b.AddVertex(v); err != nil {
			t.Fatal(err)
		}
	}
	add(vid("exec"), ontology.Execution, graph.Vertex{
		Attrs: graph.AttrMap{"note": "run-1"},
		Time:  graph.TimeTriple{EventTime: 10, CausalTime: 2, ValidFrom: 5, ValidTo: 100},
		Trust: graph.TrustAnnotation{Score: 7, Class: ontology.RoleType("author")},
	})
	add(vid("model"), ontology.Model, graph.Vertex{})
	add(vid("prompt"), ontology.Prompt, graph.Vertex{})
	add(vid("policy"), ontology.Policy, graph.Vertex{})
	add(vid("art"), ontology.Artifact, graph.Vertex{})
	add(vid("rev"), ontology.Revision, graph.Vertex{})
	add(vid("obs"), ontology.Observation, graph.Vertex{})

	if err := b.AddEdge(graph.Edge{ID: eid("e1"), Type: ontology.Materializes, From: vid("exec"), To: vid("art"), Attrs: graph.AttrMap{"k": "v"}}); err != nil {
		t.Fatal(err)
	}
	if err := b.AddHyperedge(graph.Hyperedge{
		ID:      hid("h1"),
		Type:    ontology.Executes,
		Inputs:  []identity.VertexID{vid("prompt"), vid("model"), vid("policy"), vid("art")},
		Outputs: []identity.VertexID{vid("rev"), vid("obs")},
	}); err != nil {
		t.Fatal(err)
	}
	g := b.Build()
	if err := g.Validate(); err != nil {
		t.Fatalf("fixture not well-formed: %v", err)
	}
	return g
}

func sameGraph(t *testing.T, a, b graph.Graph) {
	t.Helper()
	if len(a.Vertices()) != len(b.Vertices()) || len(a.Edges()) != len(b.Edges()) || len(a.Hyperedges()) != len(b.Hyperedges()) {
		t.Fatalf("size mismatch: v %d/%d e %d/%d h %d/%d",
			len(a.Vertices()), len(b.Vertices()), len(a.Edges()), len(b.Edges()), len(a.Hyperedges()), len(b.Hyperedges()))
	}
	for _, v := range a.Vertices() {
		got, ok := b.Vertex(v.ID)
		if !ok {
			t.Fatalf("vertex %v missing after round-trip", v.ID)
		}
		if got.Type != v.Type || got.Time != v.Time || got.Trust != v.Trust {
			t.Fatalf("vertex %v content differs: %+v vs %+v", v.ID, v, got)
		}
		if len(got.Attrs) != len(v.Attrs) {
			t.Fatalf("vertex %v attrs differ", v.ID)
		}
	}
	for _, e := range a.Edges() {
		got, ok := b.Edge(e.ID)
		if !ok || got.Type != e.Type || got.From != e.From || got.To != e.To {
			t.Fatalf("edge %v differs after round-trip", e.ID)
		}
	}
	for _, h := range a.Hyperedges() {
		got, ok := b.Hyperedge(h.ID)
		if !ok || got.Type != h.Type || len(got.Inputs) != len(h.Inputs) || len(got.Outputs) != len(h.Outputs) {
			t.Fatalf("hyperedge %v differs after round-trip", h.ID)
		}
	}
}

// Snapshot encode → Build round-trips a rich graph losslessly.
func TestSnapshotRoundTrip(t *testing.T) {
	g := richGraph(t)
	snap := graph.EncodeSnapshot(g)
	back, err := snap.Build(ontology.NewDefaultSchema())
	if err != nil {
		t.Fatal(err)
	}
	sameGraph(t, g, back)
}

// Marshal → Unmarshal JSON round-trips a rich graph losslessly.
func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	g := richGraph(t)
	data, err := graph.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	back, err := graph.Unmarshal(ontology.NewDefaultSchema(), data)
	if err != nil {
		t.Fatal(err)
	}
	sameGraph(t, g, back)
}

// Empty graph round-trips.
func TestSnapshotEmpty(t *testing.T) {
	g := graph.NewGraph(ontology.NewDefaultSchema())
	data, err := graph.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	back, err := graph.Unmarshal(ontology.NewDefaultSchema(), data)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Vertices()) != 0 {
		t.Fatal("empty graph should round-trip empty")
	}
}

// Corrupt JSON is rejected.
func TestUnmarshalCorrupt(t *testing.T) {
	if _, err := graph.Unmarshal(ontology.NewDefaultSchema(), []byte("{not json")); err == nil {
		t.Fatal("expected error on corrupt snapshot")
	}
}

// A malformed hex ID is rejected.
func TestSnapshotBadID(t *testing.T) {
	s := graph.Snapshot{Vertices: []graph.VertexSnapshot{{ID: "xyz", Type: "Artifact"}}}
	if _, err := s.Build(ontology.NewDefaultSchema()); err == nil {
		t.Fatal("expected error for malformed hex id")
	}
}

// A snapshot whose edge is inadmissible is rejected on load (validate-on-load).
func TestSnapshotValidatesOnLoad(t *testing.T) {
	// Agent -authored_by-> Agent is not admissible (needs Agent -> Artifact).
	s := graph.Snapshot{
		Vertices: []graph.VertexSnapshot{
			{ID: hexOf(vid("a1")), Type: string(ontology.Agent)},
			{ID: hexOf(vid("a2")), Type: string(ontology.Agent)},
		},
		Edges: []graph.EdgeSnapshot{
			{ID: hexOf32(eid("bad")), Type: string(ontology.AuthoredBy), From: hexOf(vid("a1")), To: hexOf(vid("a2"))},
		},
	}
	if _, err := s.Build(ontology.NewDefaultSchema()); err == nil {
		t.Fatal("expected inadmissible snapshot to be rejected on load")
	}
}

// An edge referencing a missing endpoint is rejected.
func TestSnapshotMissingEndpoint(t *testing.T) {
	s := graph.Snapshot{
		Vertices: []graph.VertexSnapshot{{ID: hexOf(vid("a1")), Type: string(ontology.Artifact)}},
		Edges: []graph.EdgeSnapshot{
			{ID: hexOf32(eid("e")), Type: string(ontology.DerivedFrom), From: hexOf(vid("a1")), To: hexOf(vid("ghost"))},
		},
	}
	if _, err := s.Build(ontology.NewDefaultSchema()); err == nil {
		t.Fatal("expected missing-endpoint snapshot to be rejected")
	}
}
