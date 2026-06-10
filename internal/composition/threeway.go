package composition

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/projection"
)

// ThreeWayMerger is the optional capability a composition Engine may satisfy
// to perform an ancestor-relative (three-way) merge. It is kept off the core
// Engine interface so the two-way contract stays minimal; callers that need
// three-way semantics type-assert to ThreeWayMerger (or use *DefaultEngine
// directly).
type ThreeWayMerger interface {
	MergeThreeWay(ctx context.Context, g graph.Graph, ancestor, left, right projection.Frontier, ps []governance.Policy) (MergeResult, error)
}

var _ ThreeWayMerger = (*DefaultEngine)(nil)

// MergeThreeWay merges two divergent frontiers (left, right) relative to their
// common ancestor. Unlike the two-way Merge — which is a plain set-union —
// three-way merge reconciles each vertex against the ancestor so that a change
// made on only one side is taken automatically and a deletion on one side is
// not silently undone by the other side.
//
// Per-vertex decision (over the union of the three frontiers' vertex IDs):
//
//   - present all three, both sides equal            → take it
//   - present all three, only one side changed       → take the changed side
//   - present all three, both sides changed, differ  → modify/modify conflict
//   - absent from ancestor, added one side only       → take the addition
//   - absent from ancestor, added both, equal         → take it
//   - absent from ancestor, added both, differ        → add/add conflict
//   - in ancestor, deleted one side, other unchanged  → honor the deletion
//   - in ancestor, deleted one side, other modified   → modify/delete conflict
//   - in ancestor, deleted both sides                 → omit
//
// Content (modify detection) comes from the projection.Edited capability when
// the frontiers satisfy it. Plain ID frontiers all read the same content from
// g, so no modify/modify divergence is visible and the merge degrades to a
// presence-only three-way: additions union and deletions are honored, but no
// content conflicts arise.
//
// When any conflict is found the result carries the typed conflicts and no
// merged frontier (the merged-xor-conflicted invariant). Otherwise the merged
// frontier is gated through governance and certified exactly like Merge.
func (e *DefaultEngine) MergeThreeWay(ctx context.Context, g graph.Graph, ancestor, left, right projection.Frontier, ps []governance.Policy) (MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return MergeResult{}, err
	}

	aSet := frontierIDSet(ancestor)
	lSet := frontierIDSet(left)
	rSet := frontierIDSet(right)

	// Deterministic iteration order over the union of all three ID sets.
	union := make(map[identity.VertexID]bool, len(aSet)+len(lSet)+len(rSet))
	for id := range aSet {
		union[id] = true
	}
	for id := range lSet {
		union[id] = true
	}
	for id := range rSet {
		union[id] = true
	}
	ids := make([]identity.VertexID, 0, len(union))
	for id := range union {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 })

	var conflicts []Conflict
	included := make([]identity.VertexID, 0, len(ids))
	chosen := make(map[identity.VertexID]graph.Vertex, len(ids))

	keep := func(id identity.VertexID, v graph.Vertex, has bool) {
		included = append(included, id)
		if has {
			chosen[id] = v
		}
	}

	for _, id := range ids {
		av, inA := contentOf(ancestor, g, id, aSet)
		lv, inL := contentOf(left, g, id, lSet)
		rv, inR := contentOf(right, g, id, rSet)

		switch {
		case inL && inR && inA:
			switch {
			case e.vertexContentEqual(lv, rv):
				keep(id, lv, true)
			case e.vertexContentEqual(lv, av):
				keep(id, rv, true) // only right changed
			case e.vertexContentEqual(rv, av):
				keep(id, lv, true) // only left changed
			default:
				conflicts = append(conflicts, e.classifyVertexConflict(id, lv, rv))
			}
		case !inA && inL && inR:
			if e.vertexContentEqual(lv, rv) {
				keep(id, lv, true)
			} else {
				conflicts = append(conflicts, e.classifyVertexConflict(id, lv, rv))
			}
		case !inA && inL && !inR:
			keep(id, lv, true) // left addition
		case !inA && !inL && inR:
			keep(id, rv, true) // right addition
		case inA && inL && !inR:
			if e.vertexContentEqual(lv, av) {
				// left unchanged, right deleted → honor deletion (omit)
			} else {
				conflicts = append(conflicts, modifyDeleteConflict(id, "left"))
			}
		case inA && !inL && inR:
			if e.vertexContentEqual(rv, av) {
				// right unchanged, left deleted → honor deletion (omit)
			} else {
				conflicts = append(conflicts, modifyDeleteConflict(id, "right"))
			}
		case inA && !inL && !inR:
			// deleted on both sides → omit
		}
	}

	if len(conflicts) > 0 {
		return MergeResult{Conflicts: conflicts}, nil
	}

	merged := projection.NewEditedFrontier(included)
	for id, v := range chosen {
		merged.Vertices[id] = v
	}

	decision, obligations, err := e.governance.Check(ctx, g, merged, ps)
	if err != nil {
		return MergeResult{}, err
	}
	if decision != governance.Sat {
		return MergeResult{
			Conflicts: []Conflict{policyConflict{
				kind:        Policy,
				boundary:    included,
				obligations: obligations,
			}},
		}, nil
	}

	cert, err := e.verification.Certify(ctx, g, merged, nil, ps)
	if err != nil {
		return MergeResult{}, fmt.Errorf("composition: %w", err)
	}

	return MergeResult{
		Frontier:    merged,
		Witness:     MergeWitness{ID: deterministicWitnessID(included)},
		Certificate: cert,
	}, nil
}

// frontierIDSet returns the set of vertex IDs a frontier reports. A nil
// frontier yields an empty set.
func frontierIDSet(f projection.Frontier) map[identity.VertexID]bool {
	s := make(map[identity.VertexID]bool)
	if f == nil {
		return s
	}
	for _, id := range f.VertexIDs() {
		s[id] = true
	}
	return s
}

// contentOf returns the vertex content a frontier carries for id, and whether
// the frontier contains id at all. Content comes from the projection.Edited
// edit map when available, then from the host graph g, then (as a last resort
// for a present-but-contentless id) a minimal vertex carrying only the ID.
func contentOf(f projection.Frontier, g graph.Graph, id identity.VertexID, set map[identity.VertexID]bool) (graph.Vertex, bool) {
	if !set[id] {
		return graph.Vertex{}, false
	}
	if ed, ok := f.(projection.Edited); ok {
		if v, ok := ed.VertexEdits()[id]; ok {
			return v, true
		}
	}
	if v, ok := g.Vertex(id); ok {
		return v, true
	}
	return graph.Vertex{ID: id}, true
}

// vertexContentEqual reports whether two vertices carry the same content:
// type, temporal triple, trust annotation, and attributes (compared with the
// engine's configured Attrs equivalence predicate).
func (e *DefaultEngine) vertexContentEqual(a, b graph.Vertex) bool {
	if a.Type != b.Type || a.Time != b.Time || a.Trust != b.Trust {
		return false
	}
	return attrsMapEqual(a.Attrs, b.Attrs, e.attrsEqual)
}

// attrsMapEqual compares two AttrMaps key-by-key using eq, treating nil and
// empty maps as equal.
func attrsMapEqual(a, b graph.AttrMap, eq AttrsEqualFunc) bool {
	if len(a) != len(b) {
		return false
	}
	if eq == nil {
		eq = DefaultAttrsEqual
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !eq(av, bv) {
			return false
		}
	}
	return true
}

// classifyVertexConflict builds a typed conflict for two vertices that differ,
// choosing the kind by the first differing dimension: type → Schema, trust →
// Trust, time → Temporal, otherwise an attribute disagreement → Textual.
func (e *DefaultEngine) classifyVertexConflict(id identity.VertexID, lv, rv graph.Vertex) Conflict {
	if lv.Type != rv.Type {
		return auditConflict{
			kind:     Schema,
			boundary: []identity.VertexID{id},
			detail:   fmt.Sprintf("type %q vs %q", lv.Type, rv.Type),
			payload:  SchemaPayload{Vertex: id, LeftType: lv.Type, RightType: rv.Type},
		}
	}
	if lv.Trust != rv.Trust {
		return auditConflict{
			kind:     Trust,
			boundary: []identity.VertexID{id},
			detail:   fmt.Sprintf("trust (%d, %q) vs (%d, %q)", lv.Trust.Score, lv.Trust.Class, rv.Trust.Score, rv.Trust.Class),
			payload:  TrustPayload{Vertex: id, Left: lv.Trust, Right: rv.Trust},
		}
	}
	if lv.Time != rv.Time {
		return auditConflict{
			kind:     Temporal,
			boundary: []identity.VertexID{id},
			detail:   fmt.Sprintf("time %+v vs %+v", lv.Time, rv.Time),
			payload:  TemporalPayload{Vertex: id, Left: lv.Time, Right: rv.Time},
		}
	}
	// Attribute disagreement: report the first differing key.
	if k, lval, rval, ok := firstAttrDiff(lv.Attrs, rv.Attrs, e.attrsEqual); ok {
		return auditConflict{
			kind:     Textual,
			boundary: []identity.VertexID{id},
			detail:   fmt.Sprintf("attr %q: %v vs %v", k, lval, rval),
			payload:  TextualPayload{Vertex: id, Key: k, Left: lval, Right: rval},
		}
	}
	// Fallback: the vertices compared unequal but no single dimension stands
	// out (e.g. attr key only present on one side). Report a generic Textual.
	return auditConflict{
		kind:     Textual,
		boundary: []identity.VertexID{id},
		detail:   "vertices differ",
	}
}

// firstAttrDiff returns the first key whose value disagrees between a and b
// (in either direction) under eq, plus the two values. ok is false when the
// two maps agree on every key.
func firstAttrDiff(a, b graph.AttrMap, eq AttrsEqualFunc) (string, any, any, bool) {
	if eq == nil {
		eq = DefaultAttrsEqual
	}
	// Stable order so the reported key is deterministic.
	keys := make([]string, 0, len(a)+len(b))
	seen := make(map[string]bool, len(a)+len(b))
	for k := range a {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		av, aok := a[k]
		bv, bok := b[k]
		if aok != bok || (aok && bok && !eq(av, bv)) {
			return k, av, bv, true
		}
	}
	return "", nil, nil, false
}

// modifyDeleteConflict builds a Structural conflict for the case where one
// side deleted a vertex and the other modified it relative to the ancestor.
// There is no auto-resolver for this kind; the caller must decide whether the
// modification or the deletion wins.
func modifyDeleteConflict(id identity.VertexID, modifiedSide string) Conflict {
	return auditConflict{
		kind:     Structural,
		boundary: []identity.VertexID{id},
		detail:   fmt.Sprintf("modify/delete: %s modified while the other side deleted %v", modifiedSide, id),
	}
}
