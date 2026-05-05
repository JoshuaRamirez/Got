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
	"context"
	"errors"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// ErrNoAuthorship is returned when a vertex has no authorship edges.
var ErrNoAuthorship = errors.New("multiagent: no authorship found")

// Responsibility is a chain of accountability through the graph. Per
// docs/design-rules.md it is a struct (single-getter data holder).
type Responsibility struct {
	Path []identity.VertexID
}

// Engine computes authorship and responsibility information.
type Engine interface {
	// Authorship returns the agent vertices that authored the target vertex.
	Authorship(ctx context.Context, g graph.Graph, target identity.VertexID) ([]identity.VertexID, error)

	// ResponsibilityTrace returns the full responsibility chain for the target.
	ResponsibilityTrace(ctx context.Context, g graph.Graph, target identity.VertexID) (Responsibility, error)
}
