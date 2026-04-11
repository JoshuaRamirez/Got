// Package graph implements the GraphCore specification.
//
// It defines the typed attributed hypergraph that is the immutable core of the
// architecture. Vertices, edges, and hyperedges are content-addressed via the
// identity package and typed via the ontology package.
//
// Imports: internal/identity, internal/ontology.
// Must not import: provenance, revision, projection, governance, verification,
// composition, realization, namespace, or repo.
package graph

import (
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// AttrMap holds arbitrary key-value metadata on graph elements.
type AttrMap map[string]any

// TimeTriple captures the temporal coordinates of a vertex.
type TimeTriple struct {
	EventTime  int64  // wall-clock time of the originating event
	CausalTime uint64 // logical / Lamport timestamp
	ValidFrom  int64  // start of validity interval
	ValidTo    int64  // end of validity interval
}

// TrustAnnotation records the trust score and role class of a vertex.
type TrustAnnotation struct {
	Score uint32
	Class ontology.RoleType
}

// Vertex is a typed, attributed, temporally and trust-annotated node.
type Vertex struct {
	ID    identity.VertexID
	Type  ontology.VertexType
	Attrs AttrMap
	Time  TimeTriple
	Trust TrustAnnotation
}

// Edge is a typed, attributed directed link between two vertices.
type Edge struct {
	ID   identity.EdgeID
	Type ontology.EdgeType
	From identity.VertexID
	To   identity.VertexID
	Attrs AttrMap
}

// Hyperedge is a typed, attributed link from a set of input vertices to a
// set of output vertices.
type Hyperedge struct {
	ID      identity.HyperedgeID
	Type    ontology.EdgeType
	Inputs  []identity.VertexID
	Outputs []identity.VertexID
	Attrs   AttrMap
}

// Query is a marker interface for graph query objects.
type Query interface {
	isQuery()
}

// Subgraph is a read-only view of a subset of a graph.
type Subgraph interface {
	VertexIDs() []identity.VertexID
	Vertices() []Vertex
	Edges() []Edge
	Hyperedges() []Hyperedge
}

// Graph is the primary immutable hypergraph abstraction.
//
// All mutating operations (WithVertex, WithEdge, WithHyperedge) return a new
// Graph value — the original is never modified. This is the append-only
// semantics from the categorical formulation: every graph morphism is a mono.
//
// Axiom: vertexId(v) in vertexIDs(addV(G, v)).
// Axiom: addE(G, e) = G if src(e) or dst(e) not in vertexIDs(G).
// Axiom: addH(G, h) = G if (inputs(h) union outputs(h)) not subset vertexIDs(G).
// Axiom: extends(G, G) — reflexive.
// Axiom: extends is transitive.
// Axiom: wellFormed(G) => every edge and hyperedge satisfies admissibility.
type Graph interface {
	Vertex(identity.VertexID) (Vertex, bool)
	Edge(identity.EdgeID) (Edge, bool)
	Hyperedge(identity.HyperedgeID) (Hyperedge, bool)

	VertexIDs() []identity.VertexID
	Vertices() []Vertex
	Edges() []Edge
	Hyperedges() []Hyperedge

	Induce(ids []identity.VertexID) (Subgraph, error)
	Query(q Query) (Subgraph, error)

	WithVertex(Vertex) (Graph, error)
	WithEdge(Edge) (Graph, error)
	WithHyperedge(Hyperedge) (Graph, error)

	Validate() error
}
