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
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// Witness attests that a named capability has emerged.
type Witness interface {
	Name() string
}

// Engine checks for emergent capabilities.
type Engine interface {
	// Emerges returns true (with a witness) if the given frontier, policies,
	// and certificates jointly produce a named capability.
	Emerges(g graph.Graph, f projection.Frontier, ps []governance.Policy, cs []verification.Certificate) (bool, Witness, error)
}
