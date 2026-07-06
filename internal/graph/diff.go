package graph

import "reflect"

// Structural diff between two graph snapshots. Because identity is
// content-addressed, elements are matched by ID: an ID present only in the new
// snapshot is Added, only in the old is Removed, and present in both but with
// differing content is Changed. (In this system a VertexID is sha256(name), so
// the same ID can carry different Type/Attrs across versions — hence Changed is
// meaningful, unlike a pure full-content hash where equal IDs imply equal
// content.) This is a semantic, structure-aware diff, not a line diff.

// VertexChange records a vertex whose content changed between snapshots.
type VertexChange struct {
	Old VertexSnapshot
	New VertexSnapshot
}

// EdgeChange records an edge whose content changed between snapshots.
type EdgeChange struct {
	Old EdgeSnapshot
	New EdgeSnapshot
}

// Delta is the structural difference from an old snapshot to a new one.
type Delta struct {
	AddedVertices   []VertexSnapshot
	RemovedVertices []VertexSnapshot
	ChangedVertices []VertexChange
	AddedEdges      []EdgeSnapshot
	RemovedEdges    []EdgeSnapshot
	ChangedEdges    []EdgeChange
}

// Empty reports whether the delta has no changes.
func (d Delta) Empty() bool {
	return len(d.AddedVertices) == 0 && len(d.RemovedVertices) == 0 && len(d.ChangedVertices) == 0 &&
		len(d.AddedEdges) == 0 && len(d.RemovedEdges) == 0 && len(d.ChangedEdges) == 0
}

// Diff computes the structural difference from old to new, matching elements
// by ID.
func Diff(old, new Snapshot) Delta {
	var d Delta

	oldV := indexVertices(old.Vertices)
	newV := indexVertices(new.Vertices)
	for id, nv := range newV {
		ov, ok := oldV[id]
		if !ok {
			d.AddedVertices = append(d.AddedVertices, nv)
			continue
		}
		if !vertexSnapEqual(ov, nv) {
			d.ChangedVertices = append(d.ChangedVertices, VertexChange{Old: ov, New: nv})
		}
	}
	for id, ov := range oldV {
		if _, ok := newV[id]; !ok {
			d.RemovedVertices = append(d.RemovedVertices, ov)
		}
	}

	oldE := indexEdges(old.Edges)
	newE := indexEdges(new.Edges)
	for id, ne := range newE {
		oe, ok := oldE[id]
		if !ok {
			d.AddedEdges = append(d.AddedEdges, ne)
			continue
		}
		if !edgeSnapEqual(oe, ne) {
			d.ChangedEdges = append(d.ChangedEdges, EdgeChange{Old: oe, New: ne})
		}
	}
	for id, oe := range oldE {
		if _, ok := newE[id]; !ok {
			d.RemovedEdges = append(d.RemovedEdges, oe)
		}
	}

	return d
}

func indexVertices(vs []VertexSnapshot) map[string]VertexSnapshot {
	m := make(map[string]VertexSnapshot, len(vs))
	for _, v := range vs {
		m[v.ID] = v
	}
	return m
}

func indexEdges(es []EdgeSnapshot) map[string]EdgeSnapshot {
	m := make(map[string]EdgeSnapshot, len(es))
	for _, e := range es {
		m[e.ID] = e
	}
	return m
}

func vertexSnapEqual(a, b VertexSnapshot) bool {
	return a.Type == b.Type && a.Time == b.Time && a.Trust == b.Trust && reflect.DeepEqual(a.Attrs, b.Attrs)
}

func edgeSnapEqual(a, b EdgeSnapshot) bool {
	return a.Type == b.Type && a.From == b.From && a.To == b.To && reflect.DeepEqual(a.Attrs, b.Attrs)
}
