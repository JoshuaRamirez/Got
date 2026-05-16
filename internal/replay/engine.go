package replay

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

// DefaultEngine replays a change capsule by checking it is replayable
// against the current graph and that the supplied environment binding
// matches the capsule's recorded environment.
//
// This is the minimum interpretation that satisfies UC-U14: it does not
// re-execute the rewrite, since the capsule does not carry the Rule. It
// confirms the structural preconditions for replay (Consumed and Produced
// vertices present) and the environment match. If both hold, the outcome
// is deterministic.
type DefaultEngine struct {
	revision revision.Engine
}

// NewEngine returns a default replay Engine backed by the supplied
// revision.Engine for the Replayable check.
func NewEngine(rev revision.Engine) *DefaultEngine {
	return &DefaultEngine{revision: rev}
}

func (e *DefaultEngine) Replay(ctx context.Context, g graph.Graph, c revision.ChangeCapsule, env verification.EnvironmentBinding) (Outcome, error) {
	if err := ctx.Err(); err != nil {
		return Outcome{}, err
	}
	if err := e.revision.Replayable(ctx, g, c); err != nil {
		return Outcome{}, err
	}
	// If the capsule recorded an explicit environment, it must match
	// the supplied binding. A zero-value Environment is treated as "any".
	if !isZeroVertexID(c.Environment) && c.Environment != env.ID {
		return Outcome{Deterministic: false},
			fmt.Errorf("%w: capsule environment %v != binding %v", ErrNonDeterministic, c.Environment, env.ID)
	}
	return Outcome{Deterministic: true}, nil
}

func isZeroVertexID(id identity.VertexID) bool {
	var zero identity.VertexID
	return id == zero
}
