package main

import (
	"encoding/base64"
	"encoding/hex"
	"sort"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
)

// renameSimilarityThreshold is the minimum line-LCS similarity for a
// delete+add pair to be treated as a rename. Conservative relative to git's
// default 50%: a pair must share at least 60% of the larger file's lines
// under LCS, and be each side's unique best match.
const renameSimilarityThreshold = 0.60

// reconcileRenames is the UC-U43 path-identity pre-pass. After same-path
// chunk/diff3 reconciliation, it pairs a base path that disappeared on one
// side with a path that side added, when the contents are similar enough to
// be the same file. A unique match runs the ordinary content merge
// (structural, then diff3, then hunk refinement) and rewrites both sides so
// the merged file sits at the new path. Low-similarity delete+add pairs and
// ambiguous ties are left alone — never silently joined.
func reconcileRenames(base, left, right graph.Snapshot) (graph.Snapshot, graph.Snapshot) {
	bC := fileContentByPath(base)
	lC := fileContentByPath(left)
	rC := fileContentByPath(right)
	lRen := matchRenames(bC, lC)
	rRen := matchRenames(bC, rC)

	olds := make([]string, 0, len(lRen)+len(rRen))
	seen := make(map[string]bool)
	for old := range lRen {
		olds = append(olds, old)
		seen[old] = true
	}
	for old := range rRen {
		if !seen[old] {
			olds = append(olds, old)
		}
	}
	sort.Strings(olds)

	leftOut := cloneSnapshot(left)
	rightOut := cloneSnapshot(right)

	for _, old := range olds {
		lNew, lOK := lRen[old]
		rNew, rOK := rRen[old]

		var dest, lText, rText string
		switch {
		case lOK && rOK && lNew == rNew:
			dest, lText, rText = lNew, lC[lNew], rC[rNew]
		case lOK && !rOK:
			if _, exists := rC[lNew]; exists {
				continue // destination already a different file on the other side
			}
			if _, still := rC[old]; !still {
				continue // other side also dropped the old path without this rename
			}
			dest, lText, rText = lNew, lC[lNew], rC[old]
		case rOK && !lOK:
			if _, exists := lC[rNew]; exists {
				continue
			}
			if _, still := lC[old]; !still {
				continue
			}
			dest, lText, rText = rNew, lC[old], rC[rNew]
		default:
			continue // divergent destinations, or nothing to pair
		}

		bc, ok := bC[old]
		if !ok {
			continue
		}
		merged, ok := chunkMerge(dest, bc, lText, rText)
		if !ok {
			merged, ok = diff3Merge(dest, bc, lText, rText)
		}
		if !ok {
			continue // genuine content conflict; leave delete+add / modify-delete
		}

		mode := fileModeAt(leftOut, dest)
		if mode == "" {
			mode = fileModeAt(rightOut, dest)
		}
		if mode == "" {
			mode = fileModeAt(leftOut, old)
		}
		if mode == "" {
			mode = fileModeAt(rightOut, old)
		}

		applyRenameResult(&leftOut, old, dest, merged, mode, lOK)
		applyRenameResult(&rightOut, old, dest, merged, mode, rOK)
	}
	return leftOut, rightOut
}

// matchRenames returns oldPath → newPath for delete+add pairs that are each
// other's unique best match at or above renameSimilarityThreshold.
func matchRenames(base, side map[string]string) map[string]string {
	var deleted, added []string
	for p := range base {
		if _, ok := side[p]; !ok {
			deleted = append(deleted, p)
		}
	}
	for p := range side {
		if _, ok := base[p]; !ok {
			added = append(added, p)
		}
	}
	sort.Strings(deleted)
	sort.Strings(added)
	if len(deleted) == 0 || len(added) == 0 {
		return nil
	}

	bestNew := make(map[string]string, len(deleted))
	bestNewScore := make(map[string]float64, len(deleted))
	for _, d := range deleted {
		p, score, ok := uniqueBest(base[d], added, func(p string) string { return side[p] })
		if ok {
			bestNew[d] = p
			bestNewScore[d] = score
		}
	}
	bestOld := make(map[string]string, len(added))
	for _, a := range added {
		p, _, ok := uniqueBest(side[a], deleted, func(p string) string { return base[p] })
		if ok {
			bestOld[a] = p
		}
	}

	out := make(map[string]string)
	for d, a := range bestNew {
		if bestOld[a] == d && bestNewScore[d] >= renameSimilarityThreshold {
			out[d] = a
		}
	}
	return out
}

// uniqueBest returns the unique highest-scoring candidate at or above the
// rename threshold. A tie for first place is not a match.
func uniqueBest(target string, cands []string, content func(string) string) (string, float64, bool) {
	var best string
	var bestScore float64
	tied := false
	found := false
	for _, c := range cands {
		s := lineSimilarity(target, content(c))
		if s < renameSimilarityThreshold {
			continue
		}
		if !found || s > bestScore {
			best, bestScore, tied, found = c, s, false, true
			continue
		}
		if s == bestScore {
			tied = true
		}
	}
	if !found || tied {
		return "", 0, false
	}
	return best, bestScore, true
}

// lineSimilarity is LCS(lines) / max(n, m). Empty files score 0 so they never
// match as renames. A size-ratio fast reject skips the LCS when the threshold
// is unreachable.
func lineSimilarity(a, b string) float64 {
	la, lb := splitLinesKeepEOL(a), splitLinesKeepEOL(b)
	n, m := len(la), len(lb)
	if n == 0 || m == 0 {
		return 0
	}
	maxLen, minLen := n, m
	if m > n {
		maxLen, minLen = m, n
	}
	if float64(minLen) < renameSimilarityThreshold*float64(maxLen) {
		return 0
	}
	return float64(len(lcsMatch(la, lb))) / float64(maxLen)
}

func applyRenameResult(s *graph.Snapshot, old, dest, content, mode string, alreadyAtDest bool) {
	if !alreadyAtDest {
		removeFileAtPath(s, old)
	}
	setFileAtPath(s, dest, content, mode)
}

func fileModeAt(s graph.Snapshot, path string) string {
	idx := fileVertexIndex(s)
	i, ok := idx[path]
	if !ok {
		return ""
	}
	m, _ := s.Vertices[i].Attrs[fileModeAttr].(string)
	return m
}

func setFileAtPath(s *graph.Snapshot, path, content, mode string) {
	idx := fileVertexIndex(*s)
	if i, ok := idx[path]; ok {
		setContent(&s.Vertices[i], content)
		return
	}
	attrs := graph.AttrMap{
		nameAttr:        path,
		filePathAttr:    path,
		fileContentAttr: base64.StdEncoding.EncodeToString([]byte(content)),
	}
	if mode != "" {
		attrs[fileModeAttr] = mode
	}
	id := vid(path)
	s.Vertices = append(s.Vertices, graph.VertexSnapshot{
		ID:    hex.EncodeToString(id[:]),
		Type:  string(ontology.Artifact),
		Attrs: attrs,
	})
}

func removeFileAtPath(s *graph.Snapshot, path string) {
	idx := fileVertexIndex(*s)
	i, ok := idx[path]
	if !ok {
		return
	}
	s.Vertices = append(s.Vertices[:i], s.Vertices[i+1:]...)
}
