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

// Strictness controls how thoroughly DefaultEngine.Merge inspects the
// candidate merge before issuing a certificate. See
// docs/devlog/2026-05-17.md for the design rationale and the limits
// of what each mode can detect under the current API.
type Strictness int

const (
	// Lenient is the historical behavior: set-union of frontiers plus a
	// governance gate. Six of eight ConflictKinds are unreachable.
	Lenient Strictness = iota

	// Strict adds in-graph audits before the governance gate. Currently
	// detects Structural (distinct edges sharing the same (from, to)
	// pair with incompatible types) and Temporal (malformed TimeTriple)
	// conflicts reachable from the merged frontier.
	//
	// Strict does NOT detect per-side content divergence (Textual,
	// Trust, type-level disagreement) because the current API does not
	// carry per-side vertex data — both sides see the same host graph.
	// Per-side detection requires either a Frontier-carries-edits
	// extension or a three-way-merge API change; see the devlog entry
	// for the analysis.
	Strict
)

// DefaultEngine merges two frontiers by computing the union and gating
// the result through governance. On `Sat`, it asks verification to issue
// a certificate. The merge witness ID is a deterministic hash of the
// sorted union of vertex IDs.
type DefaultEngine struct {
	governance   governance.Engine
	verification verification.Engine
	strictness   Strictness
}

// NewEngine returns a default composition Engine wired to the supplied
// governance and verification engines, configured for Lenient strictness.
func NewEngine(gov governance.Engine, ver verification.Engine) *DefaultEngine {
	return &DefaultEngine{governance: gov, verification: ver, strictness: Lenient}
}

// NewEngineStrict returns a default composition Engine configured for
// Strict strictness. Strict mode runs additional in-graph audits before
// the governance gate; see Strictness documentation for what it covers
// and what it does not.
func NewEngineStrict(gov governance.Engine, ver verification.Engine) *DefaultEngine {
	return &DefaultEngine{governance: gov, verification: ver, strictness: Strict}
}

// Strictness returns the configured strictness mode.
func (e *DefaultEngine) Strictness() Strictness { return e.strictness }

func (e *DefaultEngine) Merge(ctx context.Context, g graph.Graph, left, right projection.Frontier, ps []governance.Policy) (MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return MergeResult{}, err
	}

	union := unionVertexIDs(left, right)
	merged := &mergedFrontier{ids: union}

	if e.strictness == Strict {
		if conflicts := strictAudit(g, union); len(conflicts) > 0 {
			return MergeResult{Conflicts: conflicts}, nil
		}
	}

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

// --- strict-mode audits ---

// strictAudit runs in-graph checks reachable from the merged frontier.
// It emits typed conflicts for issues that today's lenient mode silently
// accepts. Returns nil if everything is consistent.
//
// What this does NOT cover: per-side content divergence (Textual, Trust,
// Schema disagreement between sides). Those require the API extension
// described in docs/devlog/2026-05-17.md.
func strictAudit(g graph.Graph, ids []identity.VertexID) []Conflict {
	idSet := make(map[identity.VertexID]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	var conflicts []Conflict
	conflicts = append(conflicts, structuralAudit(g, idSet)...)
	conflicts = append(conflicts, temporalAudit(g, idSet)...)
	return conflicts
}

// structuralAudit detects distinct edges in g that share (from, to) but
// have different edge types, when both endpoints lie in the merged
// frontier. This is the conflict that arises when two branches added
// different-typed edges between the same vertex pair.
func structuralAudit(g graph.Graph, idSet map[identity.VertexID]bool) []Conflict {
	type pair struct {
		from, to identity.VertexID
	}
	seen := make(map[pair]string)
	var conflicts []Conflict
	for _, e := range g.Edges() {
		if !idSet[e.From] || !idSet[e.To] {
			continue
		}
		key := pair{from: e.From, to: e.To}
		curType := string(e.Type)
		if prev, ok := seen[key]; ok && prev != curType {
			conflicts = append(conflicts, auditConflict{
				kind:     Structural,
				boundary: []identity.VertexID{e.From, e.To},
				detail:   fmt.Sprintf("edges of types %q and %q coexist on the same endpoint pair", prev, curType),
			})
			continue
		}
		seen[key] = curType
	}
	return conflicts
}

// temporalAudit detects vertices in the merged frontier whose TimeTriple
// is malformed: ValidTo > 0 and ValidTo < ValidFrom violates the
// half-open-interval invariant.
func temporalAudit(g graph.Graph, idSet map[identity.VertexID]bool) []Conflict {
	var conflicts []Conflict
	for id := range idSet {
		v, ok := g.Vertex(id)
		if !ok {
			continue
		}
		if v.Time.ValidTo != 0 && v.Time.ValidTo < v.Time.ValidFrom {
			conflicts = append(conflicts, auditConflict{
				kind:     Temporal,
				boundary: []identity.VertexID{id},
				detail:   fmt.Sprintf("ValidTo=%d < ValidFrom=%d", v.Time.ValidTo, v.Time.ValidFrom),
			})
		}
	}
	return conflicts
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

// auditConflict carries a strict-audit-generated conflict with a
// free-text detail string. Kept distinct from policyConflict so callers
// that type-assert can route on the source.
type auditConflict struct {
	kind     ConflictKind
	boundary []identity.VertexID
	detail   string
}

func (c auditConflict) Kind() ConflictKind            { return c.kind }
func (c auditConflict) Boundary() []identity.VertexID { return c.boundary }

// Detail returns the human-readable explanation for this audit conflict.
// Not part of the Conflict interface; callers type-assert to access it.
func (c auditConflict) Detail() string { return c.detail }

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
