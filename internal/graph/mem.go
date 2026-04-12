package graph

import (
	"errors"
	"fmt"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

var (
	ErrVertexNotFound  = errors.New("vertex not found")
	ErrMissingEndpoint = errors.New("edge endpoint not in graph")
	ErrNotWellFormed   = errors.New("graph is not well-formed")
	ErrQueryUnsupported = errors.New("unsupported query type")
)

// memGraph is an immutable, in-memory implementation of Graph.
// All With* methods return a new memGraph; the original is never modified.
type memGraph struct {
	schema   ontology.Schema
	vertices map[identity.VertexID]Vertex
	edges    map[identity.EdgeID]Edge
	hypers   map[identity.HyperedgeID]Hyperedge
}

// NewGraph creates an empty graph validated against the given schema.
func NewGraph(schema ontology.Schema) Graph {
	return &memGraph{
		schema:   schema,
		vertices: make(map[identity.VertexID]Vertex),
		edges:    make(map[identity.EdgeID]Edge),
		hypers:   make(map[identity.HyperedgeID]Hyperedge),
	}
}

func (g *memGraph) Vertex(id identity.VertexID) (Vertex, bool) {
	v, ok := g.vertices[id]
	return v, ok
}

func (g *memGraph) Edge(id identity.EdgeID) (Edge, bool) {
	e, ok := g.edges[id]
	return e, ok
}

func (g *memGraph) Hyperedge(id identity.HyperedgeID) (Hyperedge, bool) {
	h, ok := g.hypers[id]
	return h, ok
}

func (g *memGraph) VertexIDs() []identity.VertexID {
	ids := make([]identity.VertexID, 0, len(g.vertices))
	for id := range g.vertices {
		ids = append(ids, id)
	}
	return ids
}

func (g *memGraph) Vertices() []Vertex {
	vs := make([]Vertex, 0, len(g.vertices))
	for _, v := range g.vertices {
		vs = append(vs, v)
	}
	return vs
}

func (g *memGraph) Edges() []Edge {
	es := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		es = append(es, e)
	}
	return es
}

func (g *memGraph) Hyperedges() []Hyperedge {
	hs := make([]Hyperedge, 0, len(g.hypers))
	for _, h := range g.hypers {
		hs = append(hs, h)
	}
	return hs
}

// Induce returns the subgraph induced by the given vertex IDs: all specified
// vertices plus all edges and hyperedges whose endpoints lie entirely within
// the set.
func (g *memGraph) Induce(ids []identity.VertexID) (Subgraph, error) {
	idSet := make(map[identity.VertexID]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	vs := make([]Vertex, 0, len(ids))
	for _, id := range ids {
		v, ok := g.vertices[id]
		if !ok {
			return nil, fmt.Errorf("%w: %v", ErrVertexNotFound, id)
		}
		vs = append(vs, v)
	}

	var es []Edge
	for _, e := range g.edges {
		if idSet[e.From] && idSet[e.To] {
			es = append(es, e)
		}
	}

	var hs []Hyperedge
	for _, h := range g.hypers {
		if allIn(h.Inputs, idSet) && allIn(h.Outputs, idSet) {
			hs = append(hs, h)
		}
	}

	return &memSubgraph{ids: ids, verts: vs, edges: es, hypers: hs}, nil
}

// Query is a placeholder; no concrete query types are defined yet.
func (g *memGraph) Query(_ Query) (Subgraph, error) {
	return nil, ErrQueryUnsupported
}

// WithVertex returns a new graph containing all elements of g plus v.
// If a vertex with the same ID already exists it is replaced.
func (g *memGraph) WithVertex(v Vertex) (Graph, error) {
	nv := copyMap(g.vertices)
	nv[v.ID] = v
	return &memGraph{schema: g.schema, vertices: nv, edges: g.edges, hypers: g.hypers}, nil
}

// WithEdge returns a new graph containing all elements of g plus e.
// Returns an error if either endpoint is missing from the graph.
//
// Axiom: addE(G, e) = G if not (src(e) in vertexIDs(G) and dst(e) in vertexIDs(G)).
func (g *memGraph) WithEdge(e Edge) (Graph, error) {
	if _, ok := g.vertices[e.From]; !ok {
		return nil, fmt.Errorf("%w: source %v", ErrMissingEndpoint, e.From)
	}
	if _, ok := g.vertices[e.To]; !ok {
		return nil, fmt.Errorf("%w: destination %v", ErrMissingEndpoint, e.To)
	}
	ne := copyMap(g.edges)
	ne[e.ID] = e
	return &memGraph{schema: g.schema, vertices: g.vertices, edges: ne, hypers: g.hypers}, nil
}

// WithHyperedge returns a new graph containing all elements of g plus h.
// Returns an error if any input or output vertex is missing from the graph.
//
// Axiom: addH(G, h) = G if not ((inputs(h) union outputs(h)) subset vertexIDs(G)).
func (g *memGraph) WithHyperedge(h Hyperedge) (Graph, error) {
	for _, id := range h.Inputs {
		if _, ok := g.vertices[id]; !ok {
			return nil, fmt.Errorf("%w: input %v", ErrMissingEndpoint, id)
		}
	}
	for _, id := range h.Outputs {
		if _, ok := g.vertices[id]; !ok {
			return nil, fmt.Errorf("%w: output %v", ErrMissingEndpoint, id)
		}
	}
	nh := copyMap(g.hypers)
	nh[h.ID] = h
	return &memGraph{schema: g.schema, vertices: g.vertices, edges: g.edges, hypers: nh}, nil
}

// Validate checks well-formedness: referential integrity and admissibility of
// every edge and hyperedge against the schema.
//
// Axiom: wellFormed(G) => every edge satisfies admissibleEdge and every
// hyperedge satisfies admissibleHyperedge.
func (g *memGraph) Validate() error {
	for _, e := range g.edges {
		srcV, ok := g.vertices[e.From]
		if !ok {
			return fmt.Errorf("%w: edge references missing source %v", ErrNotWellFormed, e.From)
		}
		dstV, ok := g.vertices[e.To]
		if !ok {
			return fmt.Errorf("%w: edge references missing destination %v", ErrNotWellFormed, e.To)
		}
		if !g.schema.EdgeAllowed(srcV.Type, e.Type, dstV.Type) {
			return fmt.Errorf("%w: edge (%s -%s-> %s) not admissible",
				ErrNotWellFormed, srcV.Type, e.Type, dstV.Type)
		}
	}

	for _, h := range g.hypers {
		inputTypes := make([]ontology.VertexType, 0, len(h.Inputs))
		for _, id := range h.Inputs {
			v, ok := g.vertices[id]
			if !ok {
				return fmt.Errorf("%w: hyperedge references missing input %v", ErrNotWellFormed, id)
			}
			inputTypes = append(inputTypes, v.Type)
		}
		outputTypes := make([]ontology.VertexType, 0, len(h.Outputs))
		for _, id := range h.Outputs {
			v, ok := g.vertices[id]
			if !ok {
				return fmt.Errorf("%w: hyperedge references missing output %v", ErrNotWellFormed, id)
			}
			outputTypes = append(outputTypes, v.Type)
		}
		if !g.schema.HyperedgeAllowed(inputTypes, h.Type, outputTypes) {
			return fmt.Errorf("%w: hyperedge (%s) not admissible", ErrNotWellFormed, h.Type)
		}
	}

	return nil
}

// --- helpers ---

func allIn(ids []identity.VertexID, set map[identity.VertexID]bool) bool {
	for _, id := range ids {
		if !set[id] {
			return false
		}
	}
	return true
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	c := make(map[K]V, len(m)+1)
	for k, v := range m {
		c[k] = v
	}
	return c
}

// --- memSubgraph ---

type memSubgraph struct {
	ids    []identity.VertexID
	verts  []Vertex
	edges  []Edge
	hypers []Hyperedge
}

func (s *memSubgraph) VertexIDs() []identity.VertexID { return s.ids }
func (s *memSubgraph) Vertices() []Vertex             { return s.verts }
func (s *memSubgraph) Edges() []Edge                  { return s.edges }
func (s *memSubgraph) Hyperedges() []Hyperedge        { return s.hypers }
