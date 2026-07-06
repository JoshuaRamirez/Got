package repo

import (
	"context"

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
