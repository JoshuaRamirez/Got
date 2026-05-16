package governance

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/projection"
)

// defaultEngine aggregates per-policy decisions via the standard
// three-valued rule: any Unsat → aggregate Unsat; otherwise any Unknown
// → aggregate Unknown; otherwise Sat. Obligations are concatenated.
type defaultEngine struct{}

// NewEngine returns a default governance Engine.
func NewEngine() Engine {
	return defaultEngine{}
}

func (defaultEngine) Check(ctx context.Context, g graph.Graph, f projection.Frontier, ps []Policy) (Decision, []Obligation, error) {
	if err := ctx.Err(); err != nil {
		return Unsat, nil, err
	}
	if len(ps) == 0 {
		return Sat, nil, nil
	}

	aggregate := Sat
	var obligations []Obligation
	for _, p := range ps {
		if err := ctx.Err(); err != nil {
			return Unsat, nil, err
		}
		d, obs, err := p.Check(g, f)
		if err != nil {
			return Unsat, nil, fmt.Errorf("governance: policy %q check failed: %w", p.Name(), err)
		}
		obligations = append(obligations, obs...)
		switch {
		case d == Unsat:
			aggregate = Unsat
		case d == Unknown && aggregate != Unsat:
			aggregate = Unknown
		}
	}
	return aggregate, obligations, nil
}

func (e defaultEngine) GateRelease(ctx context.Context, g graph.Graph, f projection.Frontier, ps []Policy) (bool, []Obligation, error) {
	decision, obligations, err := e.Check(ctx, g, f, ps)
	if err != nil {
		return false, obligations, err
	}
	if decision == Sat && len(obligations) == 0 {
		return true, nil, nil
	}
	return false, obligations, nil
}
