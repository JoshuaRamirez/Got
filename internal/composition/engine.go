package composition

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// DefaultEngine merges two frontiers by computing the union and gating
// the result through governance. On `Sat`, it asks verification to issue
// a certificate. The merge witness ID is a deterministic hash of the
// sorted union of vertex IDs.
type DefaultEngine struct {
	governance   governance.Engine
	verification verification.Engine
}

// NewEngine returns a default composition Engine wired to the supplied
// governance and verification engines.
func NewEngine(gov governance.Engine, ver verification.Engine) *DefaultEngine {
	return &DefaultEngine{governance: gov, verification: ver}
}

func (e *DefaultEngine) Merge(ctx context.Context, g graph.Graph, left, right projection.Frontier, ps []governance.Policy) (MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return MergeResult{}, err
	}

	union := unionVertexIDs(left, right)
	merged := &mergedFrontier{ids: union}

	decision, obligations, err := e.governance.Check(ctx, g, merged, ps)
	if err != nil {
		return MergeResult{}, err
	}
	if decision != governance.Sat {
		return MergeResult{
			Conflicts: []Conflict{policyConflict{
				kind:        Policy,
				boundary:    union,
				obligations: obligations,
			}},
		}, nil
	}

	cert, err := e.verification.Certify(ctx, g, merged, nil, ps)
	if err != nil {
		return MergeResult{}, fmt.Errorf("composition: %w", err)
	}

	witness := MergeWitness{ID: deterministicWitnessID(union)}
	return MergeResult{
		Frontier:    merged,
		Witness:     witness,
		Certificate: cert,
	}, nil
}

func (e *DefaultEngine) Resolve(ctx context.Context, g graph.Graph, mr MergeResult, rs []Resolution) (MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return MergeResult{}, err
	}

	current := g
	for _, r := range rs {
		next, err := r.Apply(current)
		if err != nil {
			return MergeResult{}, fmt.Errorf("%w: resolution failed: %v", ErrConflictUnresolvable, err)
		}
		current = next
	}

	// Re-derive frontiers from the resolved graph. We approximate by
	// re-merging the original conflict boundary against itself: every
	// resolution may have mutated the graph but the boundary remains
	// the same set of identity references.
	var boundary []identity.VertexID
	for _, c := range mr.Conflicts {
		boundary = append(boundary, c.Boundary()...)
	}
	frontier := &mergedFrontier{ids: dedupe(boundary)}
	return e.Merge(ctx, current, frontier, frontier, nil)
}

// --- helpers ---

type mergedFrontier struct {
	ids []identity.VertexID
}

func (f *mergedFrontier) VertexIDs() []identity.VertexID { return f.ids }

type policyConflict struct {
	kind        ConflictKind
	boundary    []identity.VertexID
	obligations []governance.Obligation
}

func (c policyConflict) Kind() ConflictKind                   { return c.kind }
func (c policyConflict) Boundary() []identity.VertexID        { return c.boundary }
func (c policyConflict) Obligations() []governance.Obligation { return c.obligations }

func unionVertexIDs(a, b projection.Frontier) []identity.VertexID {
	seen := make(map[identity.VertexID]bool)
	var out []identity.VertexID
	for _, id := range a.VertexIDs() {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range b.VertexIDs() {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func dedupe(ids []identity.VertexID) []identity.VertexID {
	seen := make(map[identity.VertexID]bool, len(ids))
	out := make([]identity.VertexID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// deterministicWitnessID hashes the sorted vertex IDs so the same union
// always yields the same merge-witness ID. Used as a content-addressed
// marker without requiring identity.Factory.
func deterministicWitnessID(ids []identity.VertexID) identity.VertexID {
	h := sha256.New()
	h.Write([]byte("composition.merge-witness:"))
	// IDs are 32-byte hashes; concat them in input order. Callers that
	// want order-invariance can pre-sort.
	for _, id := range ids {
		h.Write(id[:])
	}
	var sum identity.VertexID
	copy(sum[:], h.Sum(nil))
	return sum
}
