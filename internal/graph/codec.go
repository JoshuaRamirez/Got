package graph

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// This file provides a lossless, canonical serialization of a Graph. It lets
// a host persist or transport a graph without replaying the ingest history —
// the graph is content-addressed, so a decoded snapshot is structurally
// identical to the original. IDs are hex-encoded; every vertex/edge/hyperedge
// field is carried. Decoding rebuilds via Builder and runs Validate, so a
// snapshot that would form an inadmissible graph is rejected on load.

// Snapshot is the serializable form of a Graph. It is a plain data holder with
// JSON tags; callers marshal it with encoding/json (or use Marshal/Unmarshal).
type Snapshot struct {
	Vertices   []VertexSnapshot    `json:"vertices"`
	Edges      []EdgeSnapshot      `json:"edges"`
	Hyperedges []HyperedgeSnapshot `json:"hyperedges,omitempty"`
}

// VertexSnapshot is the serializable form of a Vertex.
type VertexSnapshot struct {
	ID    string     `json:"id"`
	Type  string     `json:"type"`
	Attrs AttrMap    `json:"attrs,omitempty"`
	Time  TimeTriple `json:"time"`
	Trust TrustSnap  `json:"trust"`
}

// TrustSnap is the serializable form of a TrustAnnotation.
type TrustSnap struct {
	Score uint32 `json:"score"`
	Class string `json:"class"`
}

// EdgeSnapshot is the serializable form of an Edge.
type EdgeSnapshot struct {
	ID    string  `json:"id"`
	Type  string  `json:"type"`
	From  string  `json:"from"`
	To    string  `json:"to"`
	Attrs AttrMap `json:"attrs,omitempty"`
}

// HyperedgeSnapshot is the serializable form of a Hyperedge.
type HyperedgeSnapshot struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Inputs  []string `json:"inputs"`
	Outputs []string `json:"outputs"`
	Attrs   AttrMap  `json:"attrs,omitempty"`
}

// EncodeSnapshot captures the full contents of g as a Snapshot.
func EncodeSnapshot(g Graph) Snapshot {
	s := Snapshot{}
	for _, v := range g.Vertices() {
		s.Vertices = append(s.Vertices, VertexSnapshot{
			ID:    hexID(v.ID[:]),
			Type:  string(v.Type),
			Attrs: v.Attrs,
			Time:  v.Time,
			Trust: TrustSnap{Score: v.Trust.Score, Class: string(v.Trust.Class)},
		})
	}
	for _, e := range g.Edges() {
		s.Edges = append(s.Edges, EdgeSnapshot{
			ID:    hexID(e.ID[:]),
			Type:  string(e.Type),
			From:  hexID(e.From[:]),
			To:    hexID(e.To[:]),
			Attrs: e.Attrs,
		})
	}
	for _, h := range g.Hyperedges() {
		hs := HyperedgeSnapshot{
			ID:    hexID(h.ID[:]),
			Type:  string(h.Type),
			Attrs: h.Attrs,
		}
		for _, in := range h.Inputs {
			hs.Inputs = append(hs.Inputs, hexID(in[:]))
		}
		for _, out := range h.Outputs {
			hs.Outputs = append(hs.Outputs, hexID(out[:]))
		}
		s.Hyperedges = append(s.Hyperedges, hs)
	}
	return s
}

// Build reconstructs a Graph from the Snapshot using the given schema. It
// rebuilds via Builder and runs Validate, returning an error if any ID is
// malformed, an endpoint is missing, or the result is not well-formed.
func (s Snapshot) Build(schema ontology.Schema) (Graph, error) {
	b := NewBuilder(schema)

	for _, vs := range s.Vertices {
		id, err := decodeVertexID(vs.ID)
		if err != nil {
			return nil, err
		}
		_ = b.AddVertex(Vertex{
			ID:    id,
			Type:  ontology.VertexType(vs.Type),
			Attrs: vs.Attrs,
			Time:  vs.Time,
			Trust: TrustAnnotation{Score: vs.Trust.Score, Class: ontology.RoleType(vs.Trust.Class)},
		})
	}
	for _, es := range s.Edges {
		id, err := decodeEdgeID(es.ID)
		if err != nil {
			return nil, err
		}
		from, err := decodeVertexID(es.From)
		if err != nil {
			return nil, err
		}
		to, err := decodeVertexID(es.To)
		if err != nil {
			return nil, err
		}
		if err := b.AddEdge(Edge{ID: id, Type: ontology.EdgeType(es.Type), From: from, To: to, Attrs: es.Attrs}); err != nil {
			return nil, fmt.Errorf("graph: decode edge %s: %w", es.ID, err)
		}
	}
	for _, hs := range s.Hyperedges {
		id, err := decodeHyperedgeID(hs.ID)
		if err != nil {
			return nil, err
		}
		ins, err := decodeVertexIDs(hs.Inputs)
		if err != nil {
			return nil, err
		}
		outs, err := decodeVertexIDs(hs.Outputs)
		if err != nil {
			return nil, err
		}
		if err := b.AddHyperedge(Hyperedge{ID: id, Type: ontology.EdgeType(hs.Type), Inputs: ins, Outputs: outs, Attrs: hs.Attrs}); err != nil {
			return nil, fmt.Errorf("graph: decode hyperedge %s: %w", hs.ID, err)
		}
	}

	g := b.Build()
	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("graph: decoded snapshot is not well-formed: %w", err)
	}
	return g, nil
}

// Marshal serializes g to JSON. Attribute values must be JSON-serializable.
func Marshal(g Graph) ([]byte, error) {
	return json.Marshal(EncodeSnapshot(g))
}

// Unmarshal deserializes a JSON graph snapshot and rebuilds it against schema,
// validating the result.
func Unmarshal(schema ontology.Schema, data []byte) (Graph, error) {
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("graph: corrupt snapshot: %w", err)
	}
	return s.Build(schema)
}

// --- id hex helpers ---

func hexID(b []byte) string { return hex.EncodeToString(b) }

func decodeHash(s string) ([32]byte, error) {
	var h [32]byte
	b, err := hex.DecodeString(s)
	if err != nil {
		return h, fmt.Errorf("graph: bad hex id %q: %w", s, err)
	}
	if len(b) != 32 {
		return h, fmt.Errorf("graph: id %q has wrong length %d", s, len(b))
	}
	copy(h[:], b)
	return h, nil
}

func decodeVertexID(s string) (identity.VertexID, error) {
	h, err := decodeHash(s)
	return identity.VertexID(h), err
}

func decodeEdgeID(s string) (identity.EdgeID, error) {
	h, err := decodeHash(s)
	return identity.EdgeID(h), err
}

func decodeHyperedgeID(s string) (identity.HyperedgeID, error) {
	h, err := decodeHash(s)
	return identity.HyperedgeID(h), err
}

func decodeVertexIDs(ss []string) ([]identity.VertexID, error) {
	out := make([]identity.VertexID, 0, len(ss))
	for _, s := range ss {
		id, err := decodeVertexID(s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
