package graph

import (
	"fmt"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// Builder accumulates vertices, edges, and hyperedges in-place so that
// bulk graph construction is O(n) rather than the O(n²) of repeated
// WithVertex/WithEdge calls (which copy the whole map on every insert).
//
// Builder is single-writer: do not Add from multiple goroutines without
// external synchronization. Once Build is called the resulting Graph is
// the standard immutable value; subsequent Add* calls on the Builder
// continue to mutate the Builder's state but do not affect the already-
// returned Graph (Build snapshots the maps).
type Builder struct {
	schema   ontology.Schema
	vertices map[identity.VertexID]Vertex
	edges    map[identity.EdgeID]Edge
	hypers   map[identity.HyperedgeID]Hyperedge
}

// NewBuilder returns an empty Builder configured with the given schema.
func NewBuilder(schema ontology.Schema) *Builder {
	return &Builder{
		schema:   schema,
		vertices: make(map[identity.VertexID]Vertex),
		edges:    make(map[identity.EdgeID]Edge),
		hypers:   make(map[identity.HyperedgeID]Hyperedge),
	}
}

// AddVertex inserts v into the builder. Replaces any existing vertex
// with the same ID. Always returns nil; signature mirrors WithVertex
// for API symmetry.
func (b *Builder) AddVertex(v Vertex) error {
	b.vertices[v.ID] = v
	return nil
}

// AddEdge inserts e. Returns ErrMissingEndpoint if either endpoint is
// not present in the builder.
func (b *Builder) AddEdge(e Edge) error {
	if _, ok := b.vertices[e.From]; !ok {
		return fmt.Errorf("%w: source %v", ErrMissingEndpoint, e.From)
	}
	if _, ok := b.vertices[e.To]; !ok {
		return fmt.Errorf("%w: destination %v", ErrMissingEndpoint, e.To)
	}
	b.edges[e.ID] = e
	return nil
}

// AddHyperedge inserts h. Returns ErrMissingEndpoint if any input or
// output vertex is missing from the builder.
func (b *Builder) AddHyperedge(h Hyperedge) error {
	for _, id := range h.Inputs {
		if _, ok := b.vertices[id]; !ok {
			return fmt.Errorf("%w: input %v", ErrMissingEndpoint, id)
		}
	}
	for _, id := range h.Outputs {
		if _, ok := b.vertices[id]; !ok {
			return fmt.Errorf("%w: output %v", ErrMissingEndpoint, id)
		}
	}
	b.hypers[h.ID] = h
	return nil
}

// Build snapshots the builder's current state into a new immutable
// Graph value. The returned Graph is independent of subsequent
// modifications to the Builder.
func (b *Builder) Build() Graph {
	vs := make(map[identity.VertexID]Vertex, len(b.vertices))
	for k, v := range b.vertices {
		vs[k] = v
	}
	es := make(map[identity.EdgeID]Edge, len(b.edges))
	for k, v := range b.edges {
		es[k] = v
	}
	hs := make(map[identity.HyperedgeID]Hyperedge, len(b.hypers))
	for k, v := range b.hypers {
		hs[k] = v
	}
	return &memGraph{schema: b.schema, vertices: vs, edges: es, hypers: hs}
}
