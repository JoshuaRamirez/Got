package main

import "strings"

// This file provides a line-level three-way merge (diff3) used as a fallback
// when the structural chunk merge cannot cleanly reconcile a file. It gives Got
// at least git's line-based merge on every file and language — closing the cases
// where the structural chunker is coarser than a textual merge (e.g. a statement
// inserted on one side while another is edited) — while the structural and
// semantic gates still validate the result.
//
// UC-U43 refines a remaining conflict region with pairwise hunks so an
// insertion immediately next to the other side's edit merges when the hunks
// do not share a base line. Overlapping edits still conflict.

// diff3 three-way merges line slices against a common base. It returns the merged
// lines and ok == true when every change region is non-conflicting (one side
// changed, or both changed identically); on any overlapping divergent change it
// returns ok == false, exactly like a textual merge conflict.
func diff3(base, left, right []string) ([]string, bool) {
	ml := lcsMatch(base, left)
	mr := lcsMatch(base, right)

	// Anchors are base lines matched to both sides (stable in all three); LCS
	// matches are monotonic, so these are already in order. Sentinels bound the
	// head and tail gaps.
	anchors := [][3]int{{-1, -1, -1}}
	for bi := 0; bi < len(base); bi++ {
		li, okl := ml[bi]
		ri, okr := mr[bi]
		if okl && okr {
			anchors = append(anchors, [3]int{bi, li, ri})
		}
	}
	anchors = append(anchors, [3]int{len(base), len(left), len(right)})

	var out []string
	for k := 0; k+1 < len(anchors); k++ {
		lo, hi := anchors[k], anchors[k+1]
		baseGap := base[lo[0]+1 : hi[0]]
		leftGap := left[lo[1]+1 : hi[1]]
		rightGap := right[lo[2]+1 : hi[2]]

		res, ok := resolveGap(baseGap, leftGap, rightGap)
		if !ok {
			return nil, false
		}
		out = append(out, res...)

		// Emit the trailing anchor's line (identical in all three), except for
		// the final sentinel which has none.
		if k+1 < len(anchors)-1 {
			out = append(out, base[hi[0]])
		}
	}
	return out, true
}

// resolveGap reconciles one region between two anchors: if a side is unchanged
// from base, take the other; if both changed identically, take it; otherwise
// refine the region by pairwise hunks so adjacent but non-overlapping edits
// (an insertion immediately next to the other side's change) can still merge.
func resolveGap(base, left, right []string) ([]string, bool) {
	switch {
	case equalLines(left, base):
		return right, true
	case equalLines(right, base):
		return left, true
	case equalLines(left, right):
		return left, true
	default:
		return refineGap(base, left, right)
	}
}

// A hunk is one contiguous pairwise edit: base[a1:a2] is replaced by side[s1:s2].
// An insertion is an empty base range [i,i); a deletion is an empty side range.
type hunk struct {
	a1, a2 int
	s1, s2 int
}

// refineGap merges a coarse diff3 conflict region by applying pairwise hunks
// whose base ranges do not overlap. Overlapping hunks still conflict unless
// both sides made the identical replacement. This is the UC-U43 minimal-diff
// refinement: never silently join two edits of the same base line.
func refineGap(base, left, right []string) ([]string, bool) {
	return mergeHunks(base, left, right, diffHunks(base, left), diffHunks(base, right))
}

// diffHunks returns the LCS-based edit hunks that turn base into side.
func diffHunks(base, side []string) []hunk {
	match := lcsMatch(base, side)
	var hunks []hunk
	bi, si := 0, 0
	for bi < len(base) || si < len(side) {
		if s, ok := match[bi]; ok && s == si {
			bi++
			si++
			continue
		}
		a1, s1 := bi, si
		for bi < len(base) {
			if _, ok := match[bi]; ok {
				break
			}
			bi++
		}
		s2 := len(side)
		if bi < len(base) {
			s2 = match[bi]
		}
		si = s2
		if a1 != bi || s1 != si {
			hunks = append(hunks, hunk{a1: a1, a2: bi, s1: s1, s2: si})
		}
	}
	return hunks
}

// mergeHunks walks base, emitting unchanged spans and applying non-overlapping
// left/right hunks. An insertion at index i does not overlap a replacement of
// [i,j); two different insertions at the same index do.
func mergeHunks(base, left, right []string, lh, rh []hunk) ([]string, bool) {
	var out []string
	li, ri, pos := 0, 0, 0
	for li < len(lh) || ri < len(rh) {
		var l, r *hunk
		if li < len(lh) {
			l = &lh[li]
		}
		if ri < len(rh) {
			r = &rh[ri]
		}
		next := len(base)
		if l != nil && l.a1 < next {
			next = l.a1
		}
		if r != nil && r.a1 < next {
			next = r.a1
		}
		out = append(out, base[pos:next]...)
		pos = next

		switch {
		case l != nil && r != nil && hunksOverlap(*l, *r):
			if l.a1 == r.a1 && l.a2 == r.a2 && equalLines(left[l.s1:l.s2], right[r.s1:r.s2]) {
				out = append(out, left[l.s1:l.s2]...)
				pos = l.a2
				li++
				ri++
				continue
			}
			return nil, false
		case l != nil && r != nil && l.a1 == r.a1:
			// Same index, no overlap: one side inserts, the other replaces.
			if l.a1 == l.a2 {
				out = append(out, left[l.s1:l.s2]...)
				li++
				continue
			}
			out = append(out, right[r.s1:r.s2]...)
			ri++
			continue
		case l != nil && (r == nil || l.a1 < r.a1):
			out = append(out, left[l.s1:l.s2]...)
			pos = l.a2
			li++
		default:
			out = append(out, right[r.s1:r.s2]...)
			pos = r.a2
			ri++
		}
	}
	out = append(out, base[pos:]...)
	return out, true
}

// hunksOverlap reports whether two half-open base ranges share a line, or are
// competing insertions at the same index.
func hunksOverlap(a, b hunk) bool {
	if a.a1 == a.a2 && b.a1 == b.a2 {
		return a.a1 == b.a1
	}
	return a.a1 < b.a2 && b.a1 < a.a2
}

// lcsMatch returns, for each index in a, the index in b it aligns to under a
// longest common subsequence (only matched indices are present).
func lcsMatch(a, b []string) map[int]int {
	n, m := len(a), len(b)
	// dp[i][j] = LCS length of a[i:], b[j:].
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	match := make(map[int]int)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			match[i] = j
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			i++
		default:
			j++
		}
	}
	return match
}

func equalLines(a, b []string) bool {
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

// diff3Merge line-merges base/left/right and, if clean, validates the result
// with the same language gate the structural merge uses. It is the textual
// fallback: it never produces invalid code (the gate refuses that), and the
// whole-package semantic gate still runs on the assembled merge afterward.
func diff3Merge(path, base, left, right string) (string, bool) {
	merged, ok := diff3(
		splitLinesKeepEOL(base),
		splitLinesKeepEOL(left),
		splitLinesKeepEOL(right),
	)
	if !ok {
		return "", false
	}
	out := strings.Join(merged, "")
	if !mergedIsValid(path, out) {
		return "", false
	}
	return out, true
}
