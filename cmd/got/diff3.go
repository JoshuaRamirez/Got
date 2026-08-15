package main

import "strings"

// This file provides a line-level three-way merge (diff3) used as a fallback
// when the structural chunk merge cannot cleanly reconcile a file. It gives Got
// at least git's line-based merge on every file and language — closing the cases
// where the structural chunker is coarser than a textual merge (e.g. a statement
// inserted on one side while another is edited) — while the structural and
// semantic gates still validate the result.

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
// from base, take the other; if both changed identically, take it; otherwise it
// is a genuine conflict.
func resolveGap(base, left, right []string) ([]string, bool) {
	switch {
	case equalLines(left, base):
		return right, true
	case equalLines(right, base):
		return left, true
	case equalLines(left, right):
		return left, true
	default:
		return nil, false
	}
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
