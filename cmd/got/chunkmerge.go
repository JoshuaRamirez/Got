package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
)

// Chunk vertices are the in-memory representation a file is decomposed into for
// a chunk-level merge. They live only for the duration of one reconciliation —
// they are never persisted — so they reuse the Artifact type and carry their
// key and body as attributes.
const (
	chunkKeyAttr     = "chunk.key"
	chunkContentAttr = "chunk.content"
)

// reconcileFilesByChunk is a pre-pass over the three merge inputs. For every
// file changed on *both* sides relative to the base — the case the file-level
// merge would report as a conflict — it decomposes the three versions into
// chunks and runs them through the same three-way merge engine
// (repo.MergeStates, the composition pushout) at chunk granularity. If the
// chunk merge is clean (the two sides touched different chunks), it rewrites
// both sides' file content to the merged result, so the outer file-level merge
// now sees the two sides agree and no longer conflicts. Files whose chunk merge
// still conflicts are left untouched for the file-level merge to handle.
//
// This is the payoff of the graph model for chunk merges: no bespoke diff3 — the
// engine that reconciles vertices reconciles chunks, so two branches editing
// different functions in the same file merge automatically, where git's
// line-based merge reports a conflict.
func reconcileFilesByChunk(base, left, right graph.Snapshot) (graph.Snapshot, graph.Snapshot) {
	baseContent := fileContentByPath(base)
	leftOut := cloneSnapshot(left)
	rightOut := cloneSnapshot(right)
	leftIdx := fileVertexIndex(leftOut)
	rightIdx := fileVertexIndex(rightOut)

	for path, li := range leftIdx {
		ri, ok := rightIdx[path]
		if !ok {
			continue
		}
		bc, ok := baseContent[path]
		if !ok {
			continue // added on both sides; not a base-relative divergence we chunk
		}
		lc, ok := decodeContent(leftOut.Vertices[li])
		if !ok {
			continue
		}
		rc, ok := decodeContent(rightOut.Vertices[ri])
		if !ok {
			continue
		}
		// Only worth chunking when both sides diverged from base and disagree.
		if lc == bc || rc == bc || lc == rc {
			continue
		}
		merged, ok := chunkMerge(path, bc, lc, rc)
		if !ok {
			continue // chunk-level conflict; leave for the file-level merge
		}
		setContent(&leftOut.Vertices[li], merged)
		setContent(&rightOut.Vertices[ri], merged)
	}
	return leftOut, rightOut
}

// chunkMerge decomposes base/left/right into chunks, merges them through the
// graph three-way engine, and reassembles the merged file. It returns ok ==
// false when the engine reports a chunk-level conflict (both sides changed the
// same chunk differently).
func chunkMerge(path, base, left, right string) (string, bool) {
	ch := newBlockChunker()
	bC, lC, rC := ch.Split(base), ch.Split(left), ch.Split(right)

	merged, mr, err := newService().MergeStates(
		context.Background(), schema(),
		chunkSnapshot(path, bC), chunkSnapshot(path, lC), chunkSnapshot(path, rC),
	)
	if err != nil || len(mr.Conflicts) > 0 {
		return "", false
	}

	// Collect merged chunk bodies by key.
	body := make(map[string]string)
	for _, v := range merged.Vertices() {
		k, ok := v.Attrs[chunkKeyAttr].(string)
		if !ok {
			continue
		}
		c, _ := v.Attrs[chunkContentAttr].(string)
		body[k] = c
	}

	// Reassemble in base order, then any chunks added by a side (base first
	// keeps unchanged files verbatim; the clean disjoint-edit case is exact).
	var ordered []chunk
	emitted := make(map[string]bool)
	emit := func(k string) {
		if emitted[k] {
			return
		}
		if c, ok := body[k]; ok {
			ordered = append(ordered, chunk{Key: k, Content: c})
			emitted[k] = true
		}
	}
	for _, c := range bC {
		emit(c.Key)
	}
	for _, c := range lC {
		emit(c.Key)
	}
	for _, c := range rC {
		emit(c.Key)
	}
	return ch.Join(ordered), true
}

// chunkSnapshot builds a throwaway snapshot of one Artifact vertex per chunk,
// each identified by (path, chunk key) so the three-way merge aligns the same
// chunk across the three inputs.
func chunkSnapshot(path string, chunks []chunk) graph.Snapshot {
	var s graph.Snapshot
	for _, c := range chunks {
		id := vid(path + "\x00chunk\x00" + c.Key)
		s.Vertices = append(s.Vertices, graph.VertexSnapshot{
			ID:    hex.EncodeToString(id[:]),
			Type:  string(ontology.Artifact),
			Attrs: graph.AttrMap{chunkKeyAttr: c.Key, chunkContentAttr: c.Content},
		})
	}
	return s
}

// --- file-vertex helpers over snapshots ---

// fileContentByPath returns the decoded text of each file vertex in a snapshot.
func fileContentByPath(s graph.Snapshot) map[string]string {
	out := make(map[string]string)
	for _, v := range s.Vertices {
		p, ok := v.Attrs[filePathAttr].(string)
		if !ok {
			continue
		}
		if c, ok := decodeContent(v); ok {
			out[p] = c
		}
	}
	return out
}

// fileVertexIndex maps each file vertex's path to its index in s.Vertices.
func fileVertexIndex(s graph.Snapshot) map[string]int {
	out := make(map[string]int)
	for i, v := range s.Vertices {
		if p, ok := v.Attrs[filePathAttr].(string); ok {
			out[p] = i
		}
	}
	return out
}

func decodeContent(v graph.VertexSnapshot) (string, bool) {
	b64, ok := v.Attrs[fileContentAttr].(string)
	if !ok {
		return "", false
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func setContent(v *graph.VertexSnapshot, content string) {
	v.Attrs[fileContentAttr] = base64.StdEncoding.EncodeToString([]byte(content))
}

// cloneSnapshot deep-copies a snapshot's vertices and their attribute maps, so
// reconciliation can rewrite content without mutating the stored commit
// snapshots the inputs came from.
func cloneSnapshot(s graph.Snapshot) graph.Snapshot {
	out := graph.Snapshot{
		Edges:      append([]graph.EdgeSnapshot(nil), s.Edges...),
		Hyperedges: append([]graph.HyperedgeSnapshot(nil), s.Hyperedges...),
	}
	out.Vertices = make([]graph.VertexSnapshot, len(s.Vertices))
	for i, v := range s.Vertices {
		nv := v
		if v.Attrs != nil {
			na := make(graph.AttrMap, len(v.Attrs))
			for k, val := range v.Attrs {
				na[k] = val
			}
			nv.Attrs = na
		}
		out.Vertices[i] = nv
	}
	return out
}
