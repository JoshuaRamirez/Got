// Package replay re-executes recorded change capsules against a graph to
// verify deterministic reproducibility.
//
// The capsule returned by revision.Engine.Apply is the canonical replay input.
// Replay checks whether re-executing that capsule in a given environment
// produces a deterministic outcome.
//
// Imports: internal/revision, internal/verification, internal/graph.
// Must not import: composition, realization, or repo.
package replay

import (
	"context"
	"errors"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

// ErrNonDeterministic indicates a replay produced a different outcome from
// the original execution.
var ErrNonDeterministic = errors.New("replay: non-deterministic outcome")

// Outcome reports whether a replayed capsule produced a deterministic result.
// Per docs/design-rules.md it is a struct (single-getter data holder).
type Outcome struct {
	Deterministic bool
}

// Engine replays change capsules.
type Engine interface {
	// Replay re-executes the change capsule c against graph g in the given
	// environment and reports whether the outcome is deterministic.
	Replay(ctx context.Context, g graph.Graph, c revision.ChangeCapsule, env verification.EnvironmentBinding) (Outcome, error)
}
