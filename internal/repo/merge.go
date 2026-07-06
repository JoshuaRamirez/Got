package repo

import (
	"context"
	"reflect"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
)

// MergeStates performs a semantic three-way merge of two committed graph states
// (left, right) against their common-ancestor state (base), using the
// composition engine's ancestor-relative merge (UC-U18). Unlike git's
// line-based merge, reconciliation is per vertex and per edge: a change made on
// only one side is taken, and only genuine same-target divergence surfaces as a
// typed conflict.
//
// On a clean merge it returns the merged graph and a populated MergeResult. On
// conflicts it returns a nil graph and a MergeResult whose Conflicts are
// non-empty. The base snapshot may be empty (unrelated histories), in which
// case every element is treated as an addition.
func (s *DefaultService) MergeStates(ctx context.Context, schema ontology.Schema, base, left, right graph.Snapshot) (graph.Graph, composition.MergeResult, error) {
	tw, ok := s.composition.(composition.ThreeWayMerger)
	if !ok {
		return nil, composition.MergeResult{}, ErrThreeWayUnsupported
	}

	ancestor, err := editedFrontierFromSnapshot(schema, base)
	if err != nil {
		return nil, composition.MergeResult{}, err
	}
	lf, err := editedFrontierFromSnapshot(schema, left)
	if err != nil {
		return nil, composition.MergeResult{}, err
	}
	rf, err := editedFrontierFromSnapshot(schema, right)
	if err != nil {
		return nil, composition.MergeResult{}, err
	}

	// The frontiers carry all content; the host graph is only a fallback and a
	// gate target, so an empty graph suffices (nil policies).
	host := graph.NewGraph(schema)
	mr, err := tw.MergeThreeWay(ctx, host, ancestor, lf, rf, nil)
	if err != nil {
		return nil, composition.MergeResult{}, err
	}
	if len(mr.Conflicts) > 0 {
		return nil, mr, nil
	}

	merged, err := graphFromEdited(schema, mr.Frontier)
	if err != nil {
		return nil, composition.MergeResult{}, err
	}
	return merged, mr, nil
}

// MergeStatesStrategy performs the same ancestor-relative three-way merge as
// MergeStates, but always produces a merged graph: on a genuine same-target
// conflict it takes the left (preferLeft == true, "ours") or right ("theirs")
// side rather than reporting a conflict. Non-conflicting changes from both
// sides still merge normally.
func (s *DefaultService) MergeStatesStrategy(schema ontology.Schema, base, left, right graph.Snapshot, preferLeft bool) (graph.Graph, error) {
	bv, lv, rv := indexVs(base), indexVs(left), indexVs(right)
	outV := make(map[string]graph.VertexSnapshot)
	for id := range unionKeys(bv, lv, rv) {
		b, inB := bv[id]
		l, inL := lv[id]
		r, inR := rv[id]
		if chosen, keep := resolveVertex(b, inB, l, inL, r, inR, preferLeft); keep {
			outV[id] = chosen
		}
	}

	be, le, re := indexEs(base), indexEs(left), indexEs(right)
	outE := make(map[string]graph.EdgeSnapshot)
	for id := range unionKeysE(be, le, re) {
		b, inB := be[id]
		l, inL := le[id]
		r, inR := re[id]
		if chosen, keep := resolveEdge(b, inB, l, inL, r, inR, preferLeft); keep {
			outE[id] = chosen
		}
	}

	var snap graph.Snapshot
	for _, v := range outV {
		snap.Vertices = append(snap.Vertices, v)
	}
	for _, e := range outE {
		if _, ok := outV[e.From]; !ok {
			continue
		}
		if _, ok := outV[e.To]; !ok {
			continue
		}
		snap.Edges = append(snap.Edges, e)
	}
	return snap.Build(schema)
}

func resolveVertex(b graph.VertexSnapshot, inB bool, l graph.VertexSnapshot, inL bool, r graph.VertexSnapshot, inR bool, preferLeft bool) (graph.VertexSnapshot, bool) {
	pick := func() graph.VertexSnapshot {
		if preferLeft {
			return l
		}
		return r
	}
	switch {
	case inL && inR && inB:
		switch {
		case vSnapEqual(l, r):
			return l, true
		case vSnapEqual(l, b):
			return r, true // only right changed
		case vSnapEqual(r, b):
			return l, true // only left changed
		default:
			return pick(), true // both changed → strategy
		}
	case !inB && inL && inR:
		if vSnapEqual(l, r) {
			return l, true
		}
		return pick(), true
	case !inB && inL && !inR:
		return l, true
	case !inB && !inL && inR:
		return r, true
	case inB && inL && !inR: // right deleted
		if vSnapEqual(l, b) {
			return graph.VertexSnapshot{}, false // honor deletion
		}
		if preferLeft {
			return l, true // keep our modification
		}
		return graph.VertexSnapshot{}, false // theirs deleted → honor
	case inB && !inL && inR: // left deleted
		if vSnapEqual(r, b) {
			return graph.VertexSnapshot{}, false
		}
		if preferLeft {
			return graph.VertexSnapshot{}, false // we deleted → honor
		}
		return r, true
	default: // deleted both / absent
		return graph.VertexSnapshot{}, false
	}
}

func resolveEdge(b graph.EdgeSnapshot, inB bool, l graph.EdgeSnapshot, inL bool, r graph.EdgeSnapshot, inR bool, preferLeft bool) (graph.EdgeSnapshot, bool) {
	pick := func() graph.EdgeSnapshot {
		if preferLeft {
			return l
		}
		return r
	}
	switch {
	case inL && inR && inB:
		switch {
		case eSnapEqual(l, r):
			return l, true
		case eSnapEqual(l, b):
			return r, true
		case eSnapEqual(r, b):
			return l, true
		default:
			return pick(), true
		}
	case !inB && inL && inR:
		if eSnapEqual(l, r) {
			return l, true
		}
		return pick(), true
	case !inB && inL && !inR:
		return l, true
	case !inB && !inL && inR:
		return r, true
	case inB && inL && !inR:
		if eSnapEqual(l, b) {
			return graph.EdgeSnapshot{}, false
		}
		if preferLeft {
			return l, true
		}
		return graph.EdgeSnapshot{}, false
	case inB && !inL && inR:
		if eSnapEqual(r, b) {
			return graph.EdgeSnapshot{}, false
		}
		if preferLeft {
			return graph.EdgeSnapshot{}, false
		}
		return r, true
	default:
		return graph.EdgeSnapshot{}, false
	}
}

func indexVs(s graph.Snapshot) map[string]graph.VertexSnapshot {
	m := make(map[string]graph.VertexSnapshot, len(s.Vertices))
	for _, v := range s.Vertices {
		m[v.ID] = v
	}
	return m
}

func indexEs(s graph.Snapshot) map[string]graph.EdgeSnapshot {
	m := make(map[string]graph.EdgeSnapshot, len(s.Edges))
	for _, e := range s.Edges {
		m[e.ID] = e
	}
	return m
}

func unionKeys(ms ...map[string]graph.VertexSnapshot) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range ms {
		for k := range m {
			out[k] = struct{}{}
		}
	}
	return out
}

func unionKeysE(ms ...map[string]graph.EdgeSnapshot) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range ms {
		for k := range m {
			out[k] = struct{}{}
		}
	}
	return out
}

func vSnapEqual(a, b graph.VertexSnapshot) bool {
	return a.Type == b.Type && a.Time == b.Time && a.Trust == b.Trust && reflect.DeepEqual(a.Attrs, b.Attrs)
}

func eSnapEqual(a, b graph.EdgeSnapshot) bool {
	return a.Type == b.Type && a.From == b.From && a.To == b.To && reflect.DeepEqual(a.Attrs, b.Attrs)
}

// editedFrontierFromSnapshot builds an Edited frontier carrying every vertex
// and edge of a snapshot, so the three-way merge sees full per-side content.
func editedFrontierFromSnapshot(schema ontology.Schema, snap graph.Snapshot) (*projection.EditedFrontier, error) {
	g, err := snap.Build(schema)
	if err != nil {
		return nil, err
	}
	f := projection.NewEditedFrontier(g.VertexIDs())
	for _, v := range g.Vertices() {
		f.Vertices[v.ID] = v
	}
	for _, e := range g.Edges() {
		f.Edges[e.ID] = e
	}
	return f, nil
}

// graphFromEdited reconstructs a graph from a merged Edited frontier, keeping
// only edges whose endpoints survived the merge, and validates the result.
func graphFromEdited(schema ontology.Schema, f projection.Frontier) (graph.Graph, error) {
	ed, ok := f.(*projection.EditedFrontier)
	if !ok {
		// No per-side content (should not happen for MergeThreeWay results);
		// fall back to an empty graph.
		return graph.NewGraph(schema), nil
	}
	b := graph.NewBuilder(schema)
	present := make(map[string]bool, len(ed.Vertices))
	for _, v := range ed.Vertices {
		b.AddVertex(v)
		present[string(v.ID[:])] = true
	}
	for _, e := range ed.Edges {
		if !present[string(e.From[:])] || !present[string(e.To[:])] {
			continue // endpoint didn't survive; drop the edge
		}
		if err := b.AddEdge(e); err != nil {
			return nil, err
		}
	}
	g := b.Build()
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}
