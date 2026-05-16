package multiagent

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// authorshipEngine traces authorship via configured edge types. By default
// it follows ontology.AuthoredBy, which carries the agent → artifact
// relation in the canonical schema.
type authorshipEngine struct {
	authorEdges map[ontology.EdgeType]bool
}

// NewEngine returns an authorship engine that recognizes the given edge
// types as authorship. Call NewDefaultEngine for the standard configuration.
func NewEngine(authorEdges map[ontology.EdgeType]bool) Engine {
	m := make(map[ontology.EdgeType]bool, len(authorEdges))
	for k, v := range authorEdges {
		m[k] = v
	}
	return &authorshipEngine{authorEdges: m}
}

// NewDefaultEngine recognizes ontology.AuthoredBy and ontology.ApprovedBy
// as authorship/responsibility edges.
func NewDefaultEngine() Engine {
	return NewEngine(map[ontology.EdgeType]bool{
		ontology.AuthoredBy: true,
		ontology.ApprovedBy: true,
	})
}

// Authorship returns agent vertices with an authorship edge into target.
// AuthoredBy points agent → artifact, so authors are the source vertices
// of edges whose destination is target.
func (e *authorshipEngine) Authorship(ctx context.Context, g graph.Graph, target identity.VertexID) ([]identity.VertexID, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, ok := g.Vertex(target); !ok {
		return nil, fmt.Errorf("%w: %v", graph.ErrVertexNotFound, target)
	}

	seen := make(map[identity.VertexID]bool)
	var authors []identity.VertexID
	for _, edge := range g.Edges() {
		if !e.authorEdges[edge.Type] {
			continue
		}
		if edge.To != target {
			continue
		}
		if !seen[edge.From] {
			seen[edge.From] = true
			authors = append(authors, edge.From)
		}
	}
	return authors, nil
}

// ResponsibilityTrace walks the authorship chain transitively. From target,
// find authors; from each author, find their authors (delegation); continue
// until no new vertices are discovered. Returns the discovered path in
// BFS order.
func (e *authorshipEngine) ResponsibilityTrace(ctx context.Context, g graph.Graph, target identity.VertexID) (Responsibility, error) {
	if err := ctx.Err(); err != nil {
		return Responsibility{}, err
	}
	if _, ok := g.Vertex(target); !ok {
		return Responsibility{}, fmt.Errorf("%w: %v", graph.ErrVertexNotFound, target)
	}

	// Build reverse adjacency on authorship edges: dst → list of srcs.
	rev := make(map[identity.VertexID][]identity.VertexID)
	for _, edge := range g.Edges() {
		if !e.authorEdges[edge.Type] {
			continue
		}
		rev[edge.To] = append(rev[edge.To], edge.From)
	}

	seen := map[identity.VertexID]bool{target: true}
	queue := []identity.VertexID{target}
	var path []identity.VertexID

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return Responsibility{}, err
		}
		cur := queue[0]
		queue = queue[1:]
		for _, src := range rev[cur] {
			if !seen[src] {
				seen[src] = true
				path = append(path, src)
				queue = append(queue, src)
			}
		}
	}

	if len(path) == 0 {
		return Responsibility{}, fmt.Errorf("%w: target %v has no authorship edges",
			ErrNoAuthorship, target)
	}
	return Responsibility{Path: path}, nil
}
