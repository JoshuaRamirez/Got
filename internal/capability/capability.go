// Package capability detects emergent capabilities from the interplay of
// governance, verification, and projection.
//
// A capability "emerges" when a frontier, under a given policy set and with
// supporting certificates, satisfies conditions that no single component
// independently guarantees.
//
// Imports: internal/graph, internal/governance, internal/verification, internal/projection.
// Must not import: composition, realization, or repo.
package capability

import (
	"context"
	"errors"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// ErrNoEmergence indicates no capability emerged for the inputs supplied.
var ErrNoEmergence = errors.New("capability: no emergence")

// Witness attests that a named capability has emerged. Per
// docs/design-rules.md it is a struct (single-getter data holder).
type Witness struct {
	Name string
}

// Engine checks for emergent capabilities.
type Engine interface {
	// Emerges returns true (with a witness) if the given frontier, policies,
	// and certificates jointly produce a named capability.
	Emerges(ctx context.Context, g graph.Graph, f projection.Frontier, ps []governance.Policy, cs []verification.Certificate) (bool, Witness, error)
}
