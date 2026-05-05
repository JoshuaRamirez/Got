// Package multiagent traces authorship and responsibility across multiple
// agents in the graph.
//
// It answers questions like "who authored this vertex?" and "what is the
// full responsibility chain for this artifact?"
//
// Imports: internal/graph, internal/identity.
// Must not import: revision or any higher orchestration package.
package multiagent

import (
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// Responsibility traces a chain of accountability through the graph.
type Responsibility interface {
	Path() []identity.VertexID
}

// Engine computes authorship and responsibility information.
type Engine interface {
	// Authorship returns the agent vertices that authored the target vertex.
	Authorship(g graph.Graph, target identity.VertexID) ([]identity.VertexID, error)

	// ResponsibilityTrace returns the full responsibility chain for the target.
	ResponsibilityTrace(g graph.Graph, target identity.VertexID) (Responsibility, error)
}
