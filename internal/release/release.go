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
	"context"
	"errors"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

var (
	// ErrPolicyGate indicates Promote was rejected because policy gating
	// did not pass for the supplied frontier and certificate.
	ErrPolicyGate = errors.New("release: policy gate rejected")

	// ErrUnknownVersion indicates Rollback was given a version string that
	// has no recorded binding.
	ErrUnknownVersion = errors.New("release: unknown version")
)

// Service manages release lifecycle operations.
type Service interface {
	// Promote binds a namespace alias to a frontier, gated by a certificate
	// and policy set.
	Promote(ctx context.Context, a namespace.Alias, f projection.Frontier, c verification.Certificate, ps []governance.Policy) error

	// Rollback reverts the alias to a previously named state identified by the
	// given version string.
	Rollback(ctx context.Context, a namespace.Alias, to string) error
}
