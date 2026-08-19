package main

import (
	"encoding/base64"
	"encoding/hex"
	"path"
	"sort"
	"strings"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
)

// Rename similarity is shared/max(n,m) ≥ 3/5. Conservative relative to git's
// default 50%. Compared as integers (cross-multiply) so equal rationals with
// different denominators still tie. UC-U44 adds a 1/10 basename bonus, so a
// same-basename pair meets 3/5 once content is ≥ 1/2.
const (
	renameSimNum           = 3
	renameSimDen           = 5
	renameBasenameBonusNum = 1
	renameBasenameBonusDen = 10
)

// renameLCSMaxWork is the maximum n×m line-count product for which rename
// scoring runs a linear-memory LCS. Larger pairs use bag-of-lines overlap
// instead, so a tens-of-thousands-of-lines candidate cannot allocate an
// (n+1)×(m+1) matrix. Diff3's merge-time lcsMatch is unchanged.
const renameLCSMaxWork = 2_000_000

// reconcileRenames is the UC-U43/U44/U45 path-identity pre-pass. After
// same-path chunk/diff3 reconciliation, it pairs a base path that
// disappeared on one side with a path that side added, when the contents
// are similar enough to be the same file (UC-U44: basename-weighted, then
// directory-as-a-unit; UC-U45: flatten / one-level re-nest). A unique
// match runs the ordinary content merge (structural, then diff3, then hunk
// refinement) and rewrites both sides so the merged file sits at the new
// path. Low-similarity delete+add pairs, combined-score ties, and
// ambiguous directory splits are left alone — never silently joined.
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

		// Mode is a three-way attribute merge against the base and both
		// sides (dest if that side renamed, else the old path). A one-sided
		// chmod on the non-renaming side must survive; a two-sided mode
		// conflict leaves the rename unapplied.
		mode, ok := mergeMode(
			fileModeAt(base, old),
			sideMode(left, old, dest, lOK),
			sideMode(right, old, dest, rOK),
		)
		if !ok {
			continue
		}

		applyRenameResult(&leftOut, old, dest, merged, mode, lOK)
		applyRenameResult(&rightOut, old, dest, merged, mode, rOK)
	}
	return leftOut, rightOut
}

// matchRenames returns oldPath → newPath for delete+add pairs that are each
// other's unique best match at or above the 3/5 combined-similarity
// threshold (content plus a same-basename bonus). Unmatched files in a
// unique-majority directory move, flatten, or one-level re-nest are then
// paired by the recovered path mapping.
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
	for _, d := range deleted {
		p, ok := uniqueBest(d, base[d], added, func(p string) string { return side[p] })
		if ok {
			bestNew[d] = p
		}
	}
	bestOld := make(map[string]string, len(added))
	for _, a := range added {
		p, ok := uniqueBest(a, side[a], deleted, func(p string) string { return base[p] })
		if ok {
			bestOld[a] = p
		}
	}

	out := make(map[string]string)
	for d, a := range bestNew {
		if bestOld[a] == d {
			out[d] = a
		}
	}
	for d, a := range matchDirRenames(base, side, deleted, added, out) {
		out[d] = a
	}
	return out
}

// simFrac is a similarity shared/maxLen, compared as a rational so 3/5 and
// 6/10 are a tie.
type simFrac struct {
	shared, maxLen int
}

// cmpSim compares a and b as rationals (shared/maxLen). Returns 1 if a is
// strictly better, -1 if b is, 0 if they tie.
func cmpSim(a, b simFrac) int {
	left := int64(a.shared) * int64(b.maxLen)
	right := int64(b.shared) * int64(a.maxLen)
	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func meetsRenameThreshold(s simFrac) bool {
	if s.maxLen <= 0 {
		return false
	}
	return int64(s.shared)*int64(renameSimDen) >= int64(s.maxLen)*int64(renameSimNum)
}

// renameScore is content similarity plus a 1/10 bonus when the paths share
// a basename. Uncapped so a same-basename 1.0 beats a different-basename
// 1.0 rather than tying at the cap. Combined with the 3/5 threshold, a
// same-basename pair still needs content ≥ 1/2 — a low-similarity
// same-name pair cannot sneak through on the bonus alone.
func renameScore(oldPath, newPath string, content simFrac) simFrac {
	if content.maxLen <= 0 {
		return content
	}
	if path.Base(oldPath) != path.Base(newPath) {
		return content
	}
	shared := renameBasenameBonusDen*content.shared + renameBasenameBonusNum*content.maxLen
	maxLen := renameBasenameBonusDen * content.maxLen
	return simFrac{shared, maxLen}
}

// uniqueBest returns the unique highest-scoring candidate at or above the
// rename threshold (content plus basename bonus). A tie for first place is
// not a match.
func uniqueBest(oldPath, target string, cands []string, content func(string) string) (string, bool) {
	var best string
	var bestScore simFrac
	tied := false
	found := false
	for _, c := range cands {
		s := pairScore(oldPath, c, target, content(c))
		if !meetsRenameThreshold(s) {
			continue
		}
		if !found {
			best, bestScore, tied, found = c, s, false, true
			continue
		}
		switch cmpSim(s, bestScore) {
		case 1:
			best, bestScore, tied = c, s, false
		case 0:
			tied = true
		}
	}
	if !found || tied {
		return "", false
	}
	return best, true
}

// matchDirRenames pairs unmatched delete+add files that follow a unique
// majority directory move (UC-U44: immediate parent → dest parent), a
// flatten (common source prefix dropped), or a one-level re-nest (one
// extra dest parent on every dest). Root-level files are ignored. A
// destination already taken by per-file matching, or proposed by two
// unmatched olds, is refused. Each proposed pair still has to meet the
// combined similarity gate.
func matchDirRenames(base, side map[string]string, deleted, added []string, taken map[string]string) map[string]string {
	usedDest := make(map[string]bool, len(taken))
	matchedOld := make(map[string]bool, len(taken))
	for d, a := range taken {
		usedDest[a] = true
		matchedOld[d] = true
	}

	srcCount := map[string]int{}
	for _, d := range deleted {
		if p := pathParent(d); p != "" {
			srcCount[p]++
		}
	}

	votes := map[string]map[string]int{}
	addVote := func(src, dst string) {
		if src == "" || dst == "" || src == dst {
			return
		}
		m := votes[src]
		if m == nil {
			m = map[string]int{}
			votes[src] = m
		}
		m[dst]++
	}

	for _, d := range deleted {
		src := pathParent(d)
		if src == "" {
			continue
		}
		if a, ok := taken[d]; ok {
			addVote(src, pathParent(a))
			continue
		}
		var hit string
		n := 0
		b := path.Base(d)
		for _, a := range added {
			if usedDest[a] || path.Base(a) != b {
				continue
			}
			n++
			hit = a
			if n > 1 {
				break
			}
		}
		if n == 1 {
			addVote(src, pathParent(hit))
		}
	}

	mapping := map[string]string{}
	for src, n := range srcCount {
		if n < 2 {
			continue
		}
		best, bestN := "", 0
		tied := false
		for dst, c := range votes[src] {
			if c > bestN {
				best, bestN, tied = dst, c, false
			} else if c == bestN {
				tied = true
			}
		}
		if tied || best == "" || bestN*2 <= n {
			continue
		}
		mapping[src] = best
	}

	proposed := map[string][]string{} // dest → olds
	for _, d := range deleted {
		if matchedOld[d] {
			continue
		}
		dstDir, ok := mapping[pathParent(d)]
		if !ok {
			continue
		}
		dest := path.Join(dstDir, path.Base(d))
		if _, ok := side[dest]; !ok {
			continue
		}
		if _, inBase := base[dest]; inBase {
			continue
		}
		if usedDest[dest] {
			continue
		}
		s := pairScore(d, dest, base[d], side[dest])
		if !meetsRenameThreshold(s) {
			continue
		}
		proposed[dest] = append(proposed[dest], d)
	}

	out := map[string]string{}
	for dest, olds := range proposed {
		if len(olds) != 1 {
			continue
		}
		out[olds[0]] = dest
	}

	takenAll := make(map[string]string, len(taken)+len(out))
	for d, a := range taken {
		takenAll[d] = a
	}
	for d, a := range out {
		takenAll[d] = a
	}
	for d, a := range matchFlattenRenest(base, side, deleted, added, takenAll) {
		out[d] = a
	}
	return out
}

const (
	flattenXform = "flatten"
	renestXform  = "renest:"
)

// matchFlattenRenest pairs unmatched delete+add files that follow a unique
// flatten (dest is the path with a common source prefix dropped) or a
// one-level re-nest (dest is one extra parent plus the original path).
// Votes come from already-matched pairs and from unmatched files with a
// unique unused basename among adds. Unique strict majority only; a split
// does not invent a winner. Root-only trees and single-file prefixes stay
// on per-file matching.
func matchFlattenRenest(base, side map[string]string, deleted, added []string, taken map[string]string) map[string]string {
	usedDest := make(map[string]bool, len(taken))
	matchedOld := make(map[string]bool, len(taken))
	for d, a := range taken {
		usedDest[a] = true
		matchedOld[d] = true
	}

	prefixCount := map[string]int{}
	for _, d := range deleted {
		for _, pref := range pathPrefixes(d) {
			prefixCount[pref]++
		}
	}

	votes := map[string]map[string]int{}
	addVote := func(pref, key string) {
		if pref == "" || key == "" {
			return
		}
		m := votes[pref]
		if m == nil {
			m = map[string]int{}
			votes[pref] = m
		}
		m[key]++
	}

	for _, d := range deleted {
		dest, ok := voteDest(d, taken, added, usedDest)
		if !ok {
			continue
		}
		if pref, ok := flattenPrefix(d, dest); ok {
			addVote(pref, flattenXform)
		}
		if extra, ok := oneLevelRenestExtra(d, dest); ok {
			for _, pref := range pathPrefixes(d) {
				addVote(pref, renestXform+extra)
			}
		}
	}

	type prefXform struct {
		pref, key string
	}
	var mappings []prefXform
	for pref, n := range prefixCount {
		if n < 2 {
			continue
		}
		best, bestN := "", 0
		tied := false
		for key, c := range votes[pref] {
			if c > bestN {
				best, bestN, tied = key, c, false
			} else if c == bestN {
				tied = true
			}
		}
		if tied || best == "" || bestN*2 <= n {
			continue
		}
		mappings = append(mappings, prefXform{pref, best})
	}
	sort.Slice(mappings, func(i, j int) bool {
		if len(mappings[i].pref) != len(mappings[j].pref) {
			return len(mappings[i].pref) > len(mappings[j].pref)
		}
		return mappings[i].pref < mappings[j].pref
	})

	out := map[string]string{}
	for _, m := range mappings {
		proposed := map[string][]string{}
		for _, d := range deleted {
			if matchedOld[d] {
				continue
			}
			if _, ok := relToPrefix(d, m.pref); !ok {
				continue
			}
			dest, ok := applyPrefixXform(d, m.pref, m.key)
			if !ok || dest == "" {
				continue
			}
			if _, ok := side[dest]; !ok {
				continue
			}
			if _, inBase := base[dest]; inBase {
				continue
			}
			if usedDest[dest] {
				continue
			}
			s := pairScore(d, dest, base[d], side[dest])
			if !meetsRenameThreshold(s) {
				continue
			}
			proposed[dest] = append(proposed[dest], d)
		}
		for dest, olds := range proposed {
			if len(olds) != 1 {
				continue
			}
			out[olds[0]] = dest
			usedDest[dest] = true
			matchedOld[olds[0]] = true
		}
	}
	return out
}

func voteDest(d string, taken map[string]string, added []string, usedDest map[string]bool) (string, bool) {
	if a, ok := taken[d]; ok {
		return a, true
	}
	var hit string
	n := 0
	b := path.Base(d)
	for _, a := range added {
		if usedDest[a] || path.Base(a) != b {
			continue
		}
		n++
		hit = a
		if n > 1 {
			return "", false
		}
	}
	if n == 1 {
		return hit, true
	}
	return "", false
}

func pathPrefixes(p string) []string {
	var out []string
	for {
		p = pathParent(p)
		if p == "" {
			break
		}
		out = append(out, p)
	}
	return out
}

func relToPrefix(p, prefix string) (string, bool) {
	if prefix == "" {
		return p, p != ""
	}
	pref := prefix + "/"
	if !strings.HasPrefix(p, pref) {
		return "", false
	}
	rel := p[len(pref):]
	if rel == "" {
		return "", false
	}
	return rel, true
}

// flattenPrefix reports the source prefix dropped when dest is old with
// that prefix removed. The prefix must be non-empty (root-only files are
// not a flatten).
func flattenPrefix(old, dest string) (string, bool) {
	if dest == "" || dest == old {
		return "", false
	}
	suffix := "/" + dest
	if !strings.HasSuffix(old, suffix) {
		return "", false
	}
	prefix := old[:len(old)-len(suffix)]
	if prefix == "" {
		return "", false
	}
	return prefix, true
}

// oneLevelRenestExtra reports the single extra parent when dest is that
// parent plus the original path. Two or more extra parents are out of
// scope.
func oneLevelRenestExtra(old, dest string) (string, bool) {
	if old == "" || dest == "" {
		return "", false
	}
	suffix := "/" + old
	if !strings.HasSuffix(dest, suffix) {
		return "", false
	}
	extra := dest[:len(dest)-len(suffix)]
	if extra == "" || strings.Contains(extra, "/") {
		return "", false
	}
	return extra, true
}

func applyPrefixXform(old, pref, key string) (string, bool) {
	switch {
	case key == flattenXform:
		return relToPrefix(old, pref)
	case strings.HasPrefix(key, renestXform):
		extra := strings.TrimPrefix(key, renestXform)
		if extra == "" || strings.Contains(extra, "/") {
			return "", false
		}
		return path.Join(extra, old), true
	default:
		return "", false
	}
}

func pathParent(p string) string {
	d := path.Dir(p)
	if d == "." {
		return ""
	}
	return d
}

// pairScore is the path-aware rename score: content similarity with an
// LCS floor of 1/2 when basenames match (so the 1/10 bonus can still
// reach 3/5) and 3/5 when they differ (so a different-basename pair that
// cannot meet the threshold skips LCS).
func pairScore(oldPath, newPath, a, b string) simFrac {
	num, den := renameSimNum, renameSimDen
	if path.Base(oldPath) == path.Base(newPath) {
		num, den = 1, 2
	}
	return renameScore(oldPath, newPath, lineSimFracFloor(a, b, num, den))
}

// lineSimFrac is shared/max(n,m) for rename scoring. Empty files score 0.
// Path-unaware callers use the 1/2 floor (loosest; same as a basename
// pair). Scoring goes through pairScore instead.
func lineSimFrac(a, b string) simFrac {
	return lineSimFracFloor(a, b, 1, 2)
}

// lineSimFracFloor skips LCS/bag-overlap when even a perfect overlap
// cannot meet floorNum/floorDen. Pairs whose n×m product exceeds
// renameLCSMaxWork use bag-of-lines overlap (linear memory) instead of
// LCS, so scoring cannot allocate gigabytes.
func lineSimFracFloor(a, b string, floorNum, floorDen int) simFrac {
	la, lb := splitLinesKeepEOL(a), splitLinesKeepEOL(b)
	n, m := len(la), len(lb)
	if n == 0 || m == 0 {
		return simFrac{0, 1}
	}
	maxLen, minLen := n, m
	if m > n {
		maxLen, minLen = m, n
	}
	if floorDen <= 0 || int64(minLen)*int64(floorDen) < int64(maxLen)*int64(floorNum) {
		return simFrac{0, maxLen}
	}
	var shared int
	if m > 0 && n > renameLCSMaxWork/m {
		shared = lineBagOverlap(la, lb)
	} else {
		shared = lcsLen(la, lb)
	}
	return simFrac{shared, maxLen}
}

// lineSimilarity is the floating form of lineSimFrac, kept for tests.
func lineSimilarity(a, b string) float64 {
	s := lineSimFrac(a, b)
	return float64(s.shared) / float64(s.maxLen)
}

// lcsLen is the LCS length of two line slices using two rows (O(min(n,m))
// memory). Used only for rename scoring; diff3 still uses lcsMatch.
func lcsLen(a, b []string) int {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return 0
	}
	if n < m {
		a, b = b, a
		n, m = m, n
	}
	prev := make([]int, m+1)
	cur := make([]int, m+1)
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				cur[j] = prev[j-1] + 1
			} else if prev[j] >= cur[j-1] {
				cur[j] = prev[j]
			} else {
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
	}
	return prev[m]
}

// lineBagOverlap is the multiset intersection size of two line slices.
func lineBagOverlap(a, b []string) int {
	if len(a) > len(b) {
		a, b = b, a
	}
	count := make(map[string]int, len(a))
	for _, ln := range a {
		count[ln]++
	}
	n := 0
	for _, ln := range b {
		if count[ln] > 0 {
			count[ln]--
			n++
		}
	}
	return n
}

func applyRenameResult(s *graph.Snapshot, old, dest, content, mode string, alreadyAtDest bool) {
	oldVID, destVID := vid(old), vid(dest)
	oldID := hex.EncodeToString(oldVID[:])
	destID := hex.EncodeToString(destVID[:])
	if !alreadyAtDest {
		removeFileAtPath(s, old)
	}
	setFileAtPath(s, dest, content, mode)
	if oldID != destID {
		retargetIncident(s, oldID, destID)
	}
}

// mergeMode is the ordinary three-way rule on a scalar attribute: take the
// one-sided change, accept an identical two-sided change, conflict if both
// sides diverged.
func mergeMode(base, left, right string) (string, bool) {
	switch {
	case left == right:
		return left, true
	case left == base:
		return right, true
	case right == base:
		return left, true
	default:
		return "", false
	}
}

// sideMode is the mode that side presents for the file: the destination if
// it renamed, otherwise the old path.
func sideMode(s graph.Snapshot, old, dest string, renamed bool) string {
	if renamed {
		if m := fileModeAt(s, dest); m != "" {
			return m
		}
	}
	return fileModeAt(s, old)
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
		if mode != "" {
			if s.Vertices[i].Attrs == nil {
				s.Vertices[i].Attrs = graph.AttrMap{}
			}
			s.Vertices[i].Attrs[fileModeAttr] = mode
		}
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

// retargetIncident rewrites edges and hyperedges that still name fromID so
// they name toID. Dropping the old file vertex without this leaves dangling
// endpoints that Snapshot.Build/Validate reject.
func retargetIncident(s *graph.Snapshot, fromID, toID string) {
	if fromID == toID {
		return
	}
	seen := make(map[string]bool, len(s.Edges))
	edges := make([]graph.EdgeSnapshot, 0, len(s.Edges))
	for _, e := range s.Edges {
		if e.From == fromID {
			e.From = toID
		}
		if e.To == fromID {
			e.To = toID
		}
		key := e.Type + "\x00" + e.From + "\x00" + e.To
		if seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, e)
	}
	s.Edges = edges

	rewrite := func(ids []string) []string {
		out := make([]string, len(ids))
		for i, id := range ids {
			if id == fromID {
				out[i] = toID
			} else {
				out[i] = id
			}
		}
		return out
	}
	for i := range s.Hyperedges {
		s.Hyperedges[i].Inputs = rewrite(s.Hyperedges[i].Inputs)
		s.Hyperedges[i].Outputs = rewrite(s.Hyperedges[i].Outputs)
	}
}
