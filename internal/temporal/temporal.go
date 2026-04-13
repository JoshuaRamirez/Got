// Package temporal provides time-interval queries over graph vertices.
//
// It extracts validity intervals from vertex time triples and checks temporal
// freshness relative to a reference time.
//
// Imports: internal/graph, internal/identity.
// Must not import: revision or any higher orchestration package.
package temporal

import (
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// Interval is a half-open time range [From, To).
type Interval struct {
	From int64
	To   int64
}

// Engine answers temporal queries about graph vertices.
type Engine interface {
	// Validity returns the validity interval of the identified vertex.
	Validity(g graph.Graph, id identity.VertexID) (Interval, error)

	// Fresh returns true if the vertex's validity interval contains 'now'.
	Fresh(g graph.Graph, id identity.VertexID, now int64) (bool, error)
}
