package composition

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
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
	attrsEqual   AttrsEqualFunc
}

// AttrsEqualFunc decides whether two Attrs values for the same key are
// equivalent for merge purposes. Returns true when the values should be
// treated as agreeing (no Textual conflict); false when they disagree.
//
// The default predicate (DefaultAttrsEqual) does bitwise `==` with a
// recover/fmt.Sprint fallback for non-comparable types. Callers that
// need semantic equivalence (canonical JSON, set equality on slices,
// etc.) supply their own predicate via SetAttrsEqual.
type AttrsEqualFunc func(a, b any) bool

// DefaultAttrsEqual is the conservative default Attrs equivalence
// predicate. Comparable values use `==`; non-comparable types fall back
// to fmt.Sprint comparison. The recover wrapper makes the predicate
// total (no panics on non-comparable maps or slices).
func DefaultAttrsEqual(a, b any) bool {
	return attrsEqual(a, b)
}

// NewEngine returns a default composition Engine wired to the supplied
// governance and verification engines, configured for Lenient strictness
// with the default Attrs equality predicate.
func NewEngine(gov governance.Engine, ver verification.Engine) *DefaultEngine {
	return &DefaultEngine{
		governance:   gov,
		verification: ver,
		strictness:   Lenient,
		attrsEqual:   DefaultAttrsEqual,
	}
}

// NewEngineStrict returns a default composition Engine configured for
// Strict strictness with the default Attrs equality predicate. Strict
// mode runs additional in-graph audits before the governance gate; see
// Strictness documentation for what it covers and what it does not.
func NewEngineStrict(gov governance.Engine, ver verification.Engine) *DefaultEngine {
	return &DefaultEngine{
		governance:   gov,
		verification: ver,
		strictness:   Strict,
		attrsEqual:   DefaultAttrsEqual,
	}
}

// Strictness returns the configured strictness mode.
func (e *DefaultEngine) Strictness() Strictness { return e.strictness }

// SetAttrsEqual replaces the Attrs equivalence predicate used by the
// per-side audit. A nil function resets to DefaultAttrsEqual.
//
// Typical use: a domain that stores JSON-shaped values in Attrs would
// supply a canonical-JSON comparator so reordered-but-equivalent JSON
// does not surface as a Textual conflict.
func (e *DefaultEngine) SetAttrsEqual(f AttrsEqualFunc) {
	if f == nil {
		e.attrsEqual = DefaultAttrsEqual
		return
	}
	e.attrsEqual = f
}

func (e *DefaultEngine) Merge(ctx context.Context, g graph.Graph, left, right projection.Frontier, ps []governance.Policy) (MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return MergeResult{}, err
	}

	union := unionVertexIDs(left, right)
	merged := &mergedFrontier{ids: union}

	if e.strictness == Strict {
		var conflicts []Conflict
		conflicts = append(conflicts, strictAudit(g, union)...)
		conflicts = append(conflicts, perSideAudit(left, right, e.attrsEqual)...)
		if len(conflicts) > 0 {
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

// perSideAudit runs when both frontiers satisfy projection.Edited. It
// compares each side's view of the same vertex ID and emits typed
// conflicts for disagreement on type, attrs, time, or trust. Vertices
// present in only one side's edit map are not conflicts — they are
// additions, which are the normal merge case.
//
// The eq predicate decides Attrs equivalence. Callers control it via
// DefaultEngine.SetAttrsEqual; the default predicate is bitwise.
func perSideAudit(left, right projection.Frontier, eq AttrsEqualFunc) []Conflict {
	if eq == nil {
		eq = DefaultAttrsEqual
	}
	leftE, lok := left.(projection.Edited)
	rightE, rok := right.(projection.Edited)
	if !lok || !rok {
		return nil
	}
	leftV := leftE.VertexEdits()
	rightV := rightE.VertexEdits()
	leftEd := leftE.EdgeEdits()
	rightEd := rightE.EdgeEdits()

	var conflicts []Conflict

	// Stable iteration: walk the union of IDs in deterministic-ish
	// order via the leftV map first, then any right-only IDs (only
	// matters for tests; production callers should not rely on order).
	for id, lv := range leftV {
		rv, ok := rightV[id]
		if !ok {
			continue
		}
		if lv.Type != rv.Type {
			conflicts = append(conflicts, auditConflict{
				kind:     Schema,
				boundary: []identity.VertexID{id},
				detail:   fmt.Sprintf("type %q vs %q", lv.Type, rv.Type),
				payload:  SchemaPayload{Vertex: id, LeftType: lv.Type, RightType: rv.Type},
			})
		}
		if lv.Trust != rv.Trust {
			conflicts = append(conflicts, auditConflict{
				kind:     Trust,
				boundary: []identity.VertexID{id},
				detail:   fmt.Sprintf("trust (%d, %q) vs (%d, %q)", lv.Trust.Score, lv.Trust.Class, rv.Trust.Score, rv.Trust.Class),
				payload:  TrustPayload{Vertex: id, Left: lv.Trust, Right: rv.Trust},
			})
		}
		if lv.Time != rv.Time {
			conflicts = append(conflicts, auditConflict{
				kind:     Temporal,
				boundary: []identity.VertexID{id},
				detail:   fmt.Sprintf("time %+v vs %+v", lv.Time, rv.Time),
				payload:  TemporalPayload{Vertex: id, Left: lv.Time, Right: rv.Time},
			})
		}
		for k, lval := range lv.Attrs {
			if rval, ok := rv.Attrs[k]; ok && !eq(lval, rval) {
				conflicts = append(conflicts, attrConflict(id, lv.Type, rv.Type, k, lval, rval))
			}
		}
	}

	// Edge edits: same (from, to) but different types is a Structural
	// conflict surfaced at the per-side level too. Distinct IDs aren't
	// conflicts on their own.
	type pair struct{ from, to identity.VertexID }
	for _, le := range leftEd {
		for _, re := range rightEd {
			if le.From == re.From && le.To == re.To && le.Type != re.Type {
				conflicts = append(conflicts, auditConflict{
					kind:     Structural,
					boundary: []identity.VertexID{le.From, le.To},
					detail:   fmt.Sprintf("edge types %q vs %q on the same endpoints", le.Type, re.Type),
					payload: StructuralPayload{
						From: le.From, To: le.To,
						LeftType: le.Type, RightType: re.Type,
					},
				})
			}
		}
	}

	return conflicts
}

// attrConflict classifies a per-key Attrs disagreement on a vertex. When
// both sides agree on the vertex type and that type is semantically
// Evaluation or Capability, the conflict is tagged with the matching
// ConflictKind (and a typed payload) rather than the generic Textual —
// so a disagreement over an evaluation result or a capability definition
// surfaces as such. Type-divergent vertices already produce a Schema
// conflict elsewhere in the audit, so their attr diffs stay Textual.
func attrConflict(id identity.VertexID, lt, rt ontology.VertexType, key string, l, r any) Conflict {
	base := func(kind ConflictKind, payload any) auditConflict {
		return auditConflict{
			kind:     kind,
			boundary: []identity.VertexID{id},
			detail:   fmt.Sprintf("attr %q: %v vs %v", key, l, r),
			payload:  payload,
		}
	}
	if lt == rt {
		switch lt {
		case ontology.Evaluation:
			return base(Evaluation, EvaluationPayload{Vertex: id, Key: key, Left: l, Right: r})
		case ontology.Capability:
			return base(Capability, CapabilityPayload{Vertex: id, Key: key, Left: l, Right: r})
		}
	}
	return base(Textual, TextualPayload{Vertex: id, Key: key, Left: l, Right: r})
}

// attrsEqual is the conservative default equivalence for AttrMap values.
// Comparable values use ==; everything else uses fmt.Sprint comparison
// as a stand-in for canonical equality. Domain-specific predicates can
// be added later (see equivalence-predicates open decision in
// docs/devlog/2026-05-17.md).
func attrsEqual(a, b any) bool {
	type comparablePair struct{ a, b any }
	defer func() { _ = recover() }()
	// Try == first; recover from non-comparable panic via fmt fallback.
	if eq, ok := tryEqual(comparablePair{a: a, b: b}); ok {
		return eq
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

// tryEqual reports whether a == b is a valid Go comparison. The defer/
// recover in the caller covers the panic case; this helper returns ok=true
// when the types support direct comparison.
func tryEqual(p struct{ a, b any }) (eq bool, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			eq = false
			ok = false
		}
	}()
	return p.a == p.b, true
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
// free-text detail string and an optional typed payload. Kept distinct
// from policyConflict so callers that type-assert can route on the
// source.
type auditConflict struct {
	kind     ConflictKind
	boundary []identity.VertexID
	detail   string
	payload  any
}

func (c auditConflict) Kind() ConflictKind            { return c.kind }
func (c auditConflict) Boundary() []identity.VertexID { return c.boundary }

// Detail returns the human-readable explanation for this audit conflict.
// Not part of the Conflict interface; callers type-assert to access it.
func (c auditConflict) Detail() string { return c.detail }

// Payload satisfies the Payloaded interface. Audit conflicts emitted by
// the per-side audit attach a typed payload (one of SchemaPayload,
// TextualPayload, TrustPayload, TemporalPayload, StructuralPayload).
// Conflicts emitted by the in-graph audit may have a nil payload.
func (c auditConflict) Payload() any { return c.payload }

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
