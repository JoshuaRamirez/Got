// Package replay re-executes recorded change capsules against a graph to
// verify deterministic reproducibility.
//
// The capsule returned by revision.Engine.Apply is the canonical replay input.
// Replay checks whether re-executing that capsule in a given environment
// produces a deterministic outcome.
//
// Imports: internal/revision, internal/verification, internal/graph.
package replay

import (
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

// Outcome reports whether a replayed capsule produced a deterministic result.
type Outcome interface {
	Deterministic() bool
}

// Engine replays change capsules.
type Engine interface {
	// Replay re-executes the change capsule c against graph g in the given
	// environment and reports whether the outcome is deterministic.
	Replay(g graph.Graph, c revision.ChangeCapsule, env verification.EnvironmentBinding) (Outcome, error)
}
