package projection

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// defaultEngine is a thin pass-through Engine that delegates to the
// supplied Selector or Spec and wraps the result in a Frontier or View.
type defaultEngine struct{}

// NewEngine returns a default projection Engine.
func NewEngine() Engine {
	return defaultEngine{}
}

func (defaultEngine) Select(ctx context.Context, g graph.Graph, s Selector) (Frontier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids, err := s.Frontier(g)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSelector, err)
	}
	for _, id := range ids {
		if _, ok := g.Vertex(id); !ok {
			return nil, fmt.Errorf("%w: selector returned vertex not in graph: %v",
				ErrInvalidSelector, id)
		}
	}
	return &frontier{ids: ids}, nil
}

func (defaultEngine) Project(ctx context.Context, g graph.Graph, s Spec) (View, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sub, err := s.Apply(g)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSelector, err)
	}
	return &view{sub: sub}, nil
}

// frontier is the default Frontier implementation: an ordered slice of IDs.
type frontier struct {
	ids []identity.VertexID
}

func (f *frontier) VertexIDs() []identity.VertexID { return f.ids }

// view is the default View implementation: a wrapped subgraph.
type view struct {
	sub graph.Subgraph
}

func (v *view) Subgraph() graph.Subgraph { return v.sub }

// IDsSelector is a Selector that returns a pre-baked list of vertex IDs.
// Useful in tests and as the bottom of selector composition.
type IDsSelector struct {
	IDs []identity.VertexID
}

// Frontier returns the IDs unchanged.
func (s IDsSelector) Frontier(graph.Graph) ([]identity.VertexID, error) {
	return s.IDs, nil
}

// InduceSpec is a Spec that produces the subgraph induced by a fixed
// vertex set via graph.Graph.Induce.
type InduceSpec struct {
	IDs []identity.VertexID
}

// Apply delegates to Graph.Induce.
func (s InduceSpec) Apply(g graph.Graph) (graph.Subgraph, error) {
	return g.Induce(s.IDs)
}
