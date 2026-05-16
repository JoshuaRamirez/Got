package release

import (
	"context"
	"fmt"
	"sync"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// DefaultService binds release aliases via a namespace.Store and tracks
// historical bindings in an in-memory ledger keyed by (alias, version).
//
// The Service does not re-run the governance gate during Promote — it
// trusts that the supplied certificate already implies Sat, which the
// upstream verification.Engine.Certify enforces. If callers want extra
// gating they can compose Promote with their own gate.
type DefaultService struct {
	store  namespace.Store
	mu     sync.Mutex
	ledger map[aliasVersion]identity.VertexID
}

type aliasVersion struct {
	alias   namespace.Alias
	version string
}

// NewService returns a release Service backed by the supplied
// namespace.Store. The in-memory ledger starts empty.
func NewService(store namespace.Store) *DefaultService {
	return &DefaultService{
		store:  store,
		ledger: make(map[aliasVersion]identity.VertexID),
	}
}

// Promote binds the alias to a vertex derived from the frontier. The
// vertex chosen is the first VertexID in frontier.VertexIDs(); callers
// that want a deterministic witness should supply a frontier whose first
// element is the merge witness.
//
// The supplied certificate must target the supplied frontier — this is
// the only inline gate, since the cert already encodes the governance
// decision. The version under which this binding is recorded in the
// ledger is the hex-prefix of the bound vertex ID.
func (s *DefaultService) Promote(ctx context.Context, alias namespace.Alias, f projection.Frontier, c verification.Certificate, _ []governance.Policy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ids := f.VertexIDs()
	if len(ids) == 0 {
		return fmt.Errorf("%w: empty frontier", ErrPolicyGate)
	}
	if c == nil {
		return fmt.Errorf("%w: nil certificate", ErrPolicyGate)
	}
	if c.Target() == nil {
		return fmt.Errorf("%w: certificate has no target", ErrPolicyGate)
	}
	if !sameFrontier(c.Target(), f) {
		return fmt.Errorf("%w: certificate target does not match frontier", ErrPolicyGate)
	}
	target := ids[0]
	if err := s.store.BindAlias(ctx, alias, target); err != nil {
		return err
	}
	version := fmt.Sprintf("v-%x", [32]byte(target))[:10]
	s.mu.Lock()
	s.ledger[aliasVersion{alias: alias, version: version}] = target
	s.mu.Unlock()
	return nil
}

// Rollback rebinds the alias to a previously-recorded version.
func (s *DefaultService) Rollback(ctx context.Context, alias namespace.Alias, to string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	target, ok := s.ledger[aliasVersion{alias: alias, version: to}]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: alias=%q version=%q", ErrUnknownVersion, alias, to)
	}
	return s.store.BindAlias(ctx, alias, target)
}

// sameFrontier returns true if two Frontiers have the same VertexIDs in
// the same order. Frontier equality at the interface level is not
// specified, so this is a conservative check.
func sameFrontier(a, b projection.Frontier) bool {
	aIDs := a.VertexIDs()
	bIDs := b.VertexIDs()
	if len(aIDs) != len(bIDs) {
		return false
	}
	for i := range aIDs {
		if aIDs[i] != bIDs[i] {
			return false
		}
	}
	return true
}
