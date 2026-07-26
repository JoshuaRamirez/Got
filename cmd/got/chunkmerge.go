package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"strings"

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
// graph three-way engine, reassembles the merged file, and validity-gates it.
// It returns ok == false in any of three cases: the engine reports a chunk-level
// content conflict (both sides changed the same chunk differently); the chunk
// order cannot be three-way merged (both sides reordered incompatibly); or the
// reassembled file fails the structural-validity gate. In every such case the
// file is left for the file-level merge to flag.
func chunkMerge(path, base, left, right string) (string, bool) {
	ch := chunkerFor(path)
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

	// Reassemble. The *content* of each surviving chunk comes from the graph
	// merge (body); the *order* is a separate three-way merge of the chunk key
	// sequences, recomputed here from the ordered Split slices. Order is never
	// stored on a chunk vertex: repo.MergeStates compares whole attribute maps,
	// so an order attribute would turn an independent reorder into a false
	// content conflict.
	seq, ok := mergeChunkOrder(bC, lC, rC, body)
	if !ok {
		return "", false // incompatible two-sided reorder → leave to file-level merge
	}
	if strings.HasSuffix(path, ".go") {
		// Keep decomposed import chunks contiguous and correctly bracketed:
		// generic ordering would append a newly-added spec after the block's
		// tail (outside the parens). Regroup them as head, specs, tail.
		seq = groupImportChunks(seq)
	}
	ordered := make([]chunk, 0, len(seq))
	for _, k := range seq {
		ordered = append(ordered, chunk{Key: k, Content: body[k]})
	}
	mergedContent := ch.Join(ordered)

	// Structural-validity gate: refuse to auto-produce a merged file that is not
	// structurally sound (e.g. two funcs cleanly merged from different chunks
	// that collide at package scope). Refusing here leaves the file for the
	// file-level merge to flag, so the developer resolves it rather than
	// committing invalid code — a check git cannot make.
	if !mergedIsValid(path, mergedContent) {
		return "", false
	}
	return mergedContent, true
}

// mergeChunkOrder three-way merges the *sequence* of chunk keys (independent of
// their content, which the graph merge already reconciled into `body`). It
// returns the merged key order over the surviving chunks, and ok == false only
// when both sides reordered the shared chunks into different sequences.
//
// The rule: if neither side reordered the shared chunks, emit base order then
// each side's additions (identical to the pre-order-merge behavior, so
// disjoint-edit results are unchanged). If exactly one side reordered, that
// side's full sequence is the authority (preserving both its reorder and the
// placement of its own additions) and the other side's additions are appended.
// If both reordered identically, accept; if differently, conflict.
func mergeChunkOrder(base, left, right []chunk, body map[string]string) ([]string, bool) {
	bKeys, lKeys, rKeys := keysOf(base), keysOf(left), keysOf(right)
	baseSet := setOf(bKeys)

	lReordered := sideReordered(bKeys, lKeys, baseSet)
	rReordered := sideReordered(bKeys, rKeys, baseSet)

	var authority []string
	otherAdds := [][]string{}
	switch {
	case !lReordered && !rReordered:
		authority, otherAdds = bKeys, [][]string{lKeys, rKeys}
	case lReordered && !rReordered:
		authority, otherAdds = lKeys, [][]string{rKeys}
	case rReordered && !lReordered:
		authority, otherAdds = rKeys, [][]string{lKeys}
	default: // both reordered
		if !equalSeq(commonOrder(lKeys, baseSet), commonOrder(rKeys, baseSet)) {
			return nil, false
		}
		authority, otherAdds = lKeys, [][]string{rKeys}
	}

	var out []string
	seen := make(map[string]bool)
	push := func(k string) {
		if seen[k] {
			return
		}
		if _, alive := body[k]; !alive {
			return // dropped by the content merge (deletion honored)
		}
		out = append(out, k)
		seen[k] = true
	}
	for _, k := range authority {
		push(k)
	}
	for _, seq := range otherAdds {
		for _, k := range seq {
			if !baseSet[k] {
				push(k) // only that side's own additions; shared keys came from authority
			}
		}
	}
	return out, true
}

// sideReordered reports whether a side changed the relative order of the chunks
// it shares with the base (additions and deletions do not count as reorder).
func sideReordered(bKeys, sideKeys []string, baseSet map[string]bool) bool {
	sideSet := setOf(sideKeys)
	var baseCommon []string
	for _, k := range bKeys {
		if sideSet[k] {
			baseCommon = append(baseCommon, k)
		}
	}
	return !equalSeq(baseCommon, commonOrder(sideKeys, baseSet))
}

// commonOrder is the subsequence of keys that are present in the base set.
func commonOrder(keys []string, baseSet map[string]bool) []string {
	var out []string
	for _, k := range keys {
		if baseSet[k] {
			out = append(out, k)
		}
	}
	return out
}

func keysOf(chunks []chunk) []string {
	out := make([]string, len(chunks))
	for i, c := range chunks {
		out[i] = c.Key
	}
	return out
}

func setOf(keys []string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func equalSeq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// groupImportChunks relocates decomposed import chunks (keys "import.head",
// "import:<path>", "import.tail") into one contiguous, correctly-ordered block
// — head(s), then specs, then tail(s) — positioned where the first import chunk
// appeared. Without this, an import spec added on one side (an "addition") would
// be appended by the generic reassembly after the block's closing paren,
// producing unparseable Go. Multiple import blocks (rare) collapse into one.
func groupImportChunks(seq []string) []string {
	firstAt := -1
	var heads, specs, tails, rest []string
	for _, k := range seq {
		switch {
		case strings.HasPrefix(k, "import.head"):
			if firstAt < 0 {
				firstAt = len(rest)
			}
			heads = append(heads, k)
		case strings.HasPrefix(k, "import.tail"):
			if firstAt < 0 {
				firstAt = len(rest)
			}
			tails = append(tails, k)
		case strings.HasPrefix(k, "import:"):
			if firstAt < 0 {
				firstAt = len(rest)
			}
			specs = append(specs, k)
		default:
			rest = append(rest, k)
		}
	}
	if firstAt < 0 {
		return seq // no import chunks
	}
	group := make([]string, 0, len(heads)+len(specs)+len(tails))
	group = append(group, heads...)
	group = append(group, specs...)
	group = append(group, tails...)

	out := make([]string, 0, len(seq))
	out = append(out, rest[:firstAt]...)
	out = append(out, group...)
	out = append(out, rest[firstAt:]...)
	return out
}

// chunkerFor selects the language-aware chunker for a path, falling back to the
// language-agnostic block chunker.
func chunkerFor(path string) chunker {
	if strings.HasSuffix(path, ".go") {
		return newGoChunker()
	}
	return newBlockChunker()
}

// mergedIsValid runs the language-specific validity gate for a path. Non-Go
// files have no gate (any text is "valid"), so they pass.
func mergedIsValid(path, content string) bool {
	if strings.HasSuffix(path, ".go") {
		return goValidityOK(content)
	}
	return true
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
