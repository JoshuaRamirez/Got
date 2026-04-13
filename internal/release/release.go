// Package release manages the promotion and rollback of named release aliases.
//
// A release binds a namespace alias to a certified frontier under governance
// constraints. Rollback reverts to a previously named state.
//
// Note: this package exposes a Service (not an Engine) because it orchestrates
// multiple subsystems (governance, verification, namespace) rather than
// implementing a single domain operation.
//
// Imports: internal/governance, internal/verification, internal/namespace, internal/projection.
// Must not import: composition, realization, or repo.
package release

import (
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// Service manages release lifecycle operations.
type Service interface {
	// Promote binds a namespace alias to a frontier, gated by a certificate
	// and policy set.
	Promote(a namespace.Alias, f projection.Frontier, c verification.Certificate, ps []governance.Policy) error

	// Rollback reverts the alias to a previously named state identified by the
	// given version string.
	Rollback(a namespace.Alias, to string) error
}
