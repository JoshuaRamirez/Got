package temporal

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// defaultEngine answers temporal queries by reading the vertex's TimeTriple.
type defaultEngine struct{}

// NewEngine returns a default temporal Engine.
func NewEngine() Engine {
	return defaultEngine{}
}

func (defaultEngine) Validity(ctx context.Context, g graph.Graph, id identity.VertexID) (Interval, error) {
	if err := ctx.Err(); err != nil {
		return Interval{}, err
	}
	v, ok := g.Vertex(id)
	if !ok {
		return Interval{}, fmt.Errorf("%w: %v", ErrUnknownVertex, id)
	}
	from, to := v.Time.ValidFrom, v.Time.ValidTo
	if to != 0 && to < from {
		return Interval{}, fmt.Errorf("temporal: malformed time triple on %v: ValidTo(%d) < ValidFrom(%d)",
			id, to, from)
	}
	return Interval{From: from, To: to}, nil
}

func (e defaultEngine) Fresh(ctx context.Context, g graph.Graph, id identity.VertexID, now int64) (bool, error) {
	iv, err := e.Validity(ctx, g, id)
	if err != nil {
		return false, err
	}
	// ValidTo == 0 is treated as "indefinite" — no upper bound.
	if iv.To == 0 {
		return now >= iv.From, nil
	}
	return now >= iv.From && now < iv.To, nil
}
