package main

import (
	"bytes"
	"sort"
	"strings"
	"testing"
)

// Bounds so a seed run (and a short local -fuzz) cannot hang or allocate
// gigabytes. renameLCSMaxWork is unchanged; these caps keep n×m well under it.
const (
	fuzzMaxFiles   = 16
	fuzzMaxPathLen = 128
	fuzzMaxContent = 1024
	fuzzMaxLines   = 64
)

// FuzzMatchRenamesInvariants asserts existing matcher behavior on random
// trees. Seeds are the UC-U43–U46 fixtures from rename_test.go, including
// competing same-basename dests so leftovers reach matchFlattenRenest.
// Copy detection stays out: a path present on both sides is never paired.
func FuzzMatchRenamesInvariants(f *testing.F) {
	for _, fx := range renameFuzzFixtures() {
		f.Add(encodeRenameTree(fx.base), encodeRenameTree(fx.side))
	}

	f.Fuzz(func(t *testing.T, baseEnc, sideEnc []byte) {
		assertMatchRenameInvariants(t, decodeRenameTree(baseEnc), decodeRenameTree(sideEnc))
	})
}

// FuzzPathPrefixesTerminates asserts pathPrefixes walks parents in finite
// time. Absolute Unix paths must reach "/" at most once; pathParent(p)==p
// (including "/") is the stop, not a loop.
func FuzzPathPrefixesTerminates(f *testing.F) {
	for _, p := range []string{
		"/pkg/a.txt",
		"/",
		"/pkg",
		"pkg/a.txt",
		"pkg/sub/b.txt",
		"vendor/lib/pkg/a.txt",
		"",
		".",
	} {
		f.Add(p)
	}

	f.Fuzz(func(t *testing.T, p string) {
		if len(p) > fuzzMaxPathLen {
			p = p[:fuzzMaxPathLen]
		}
		got := pathPrefixes(p)
		want := walkPathPrefixes(p)
		if !stringSlicesEqual(got, want) {
			t.Fatalf("pathPrefixes(%q) = %v, independent walk = %v", p, got, want)
		}
		slash := 0
		cur := p
		for i, pref := range got {
			if pref == "" {
				t.Fatalf("empty prefix at %d in %v", i, got)
			}
			if pref == cur {
				t.Fatalf("self-parent included: path=%q prefixes=%v", p, got)
			}
			if pathParent(cur) != pref {
				t.Fatalf("step %d: pathParent(%q)=%q, got %q", i, cur, pathParent(cur), pref)
			}
			if pref == "/" {
				slash++
			}
			cur = pref
		}
		if slash > 1 {
			t.Fatalf("must include / at most once, path=%q prefixes=%v", p, got)
		}
		if next := pathParent(cur); next != "" && next != cur {
			t.Fatalf("stopped early: last=%q parent=%q path=%q", cur, next, p)
		}
		if p == "/" && len(got) != 0 {
			t.Fatalf("pathPrefixes(/) must be empty (parent==self), got %v", got)
		}
	})
}

// TestRenameFuzzSeedCorpus keeps the seed encode/decode lossless for the
// UC fixtures and checks that competing-dest splits actually miss per-file
// matching (so leftovers hit matchFlattenRenest).
func TestRenameFuzzSeedCorpus(t *testing.T) {
	for _, fx := range renameFuzzFixtures() {
		base := decodeRenameTree(encodeRenameTree(fx.base))
		side := decodeRenameTree(encodeRenameTree(fx.side))
		if !mapsEqual(base, fx.base) || !mapsEqual(side, fx.side) {
			t.Fatalf("%s: encode/decode lost the fixture\nbase=%v\nside=%v", fx.name, base, side)
		}
		if fx.competingDests {
			deleted, added := fileDeleteAdds(base, side)
			taken := matchPerFile(base, side, deleted, added)
			if len(taken) != 0 {
				t.Fatalf("%s: per-file uniqueBest must refuse (competing dests), taken=%v", fx.name, taken)
			}
		}
		assertMatchRenameInvariants(t, base, side)
	}
}

type renameFuzzFixture struct {
	name           string
	base, side     map[string]string
	competingDests bool
}

func renameFuzzFixtures() []renameFuzzFixture {
	ident := "l1\nl2\nl3\nl4\nl5\n"
	body := "a\nb\nc\nd\ne\n"
	ten := tenLineBody()
	half := halfChangedBody()
	unrel := "z0\nz1\nz2\nz3\nz4\nz5\nz6\nz7\nz8\nz9\n"
	return []renameFuzzFixture{
		// UC-U43
		{name: "u43-high-sim", base: map[string]string{"old.txt": body}, side: map[string]string{"new.txt": "a\nb\nc\nd\nE\n"}},
		{name: "u43-unrelated", base: map[string]string{"old.txt": body}, side: map[string]string{"other.txt": "zz\nyy\nxx\nww\nvv\n"}},
		{name: "u43-content-tie", base: map[string]string{"old.txt": body}, side: map[string]string{"a.txt": body, "b.txt": body}},
		{name: "u43-empty", base: map[string]string{"old.txt": ""}, side: map[string]string{"new.txt": ""}},
		{name: "u43-copy-stays-add", base: map[string]string{"keep.txt": body}, side: map[string]string{"keep.txt": body, "copy.txt": body}},
		{name: "u43-rational-tie", base: map[string]string{"old.txt": "a\nb\nc\nd\ne\nf\n"}, side: map[string]string{
			"wide.txt": "a\nb\nc\nd\ne\nf\nX\nY\nZ\n",
			"tall.txt": "a\nb\nc\nd\nP\nQ\n",
		}},
		{name: "u43-unchanged", base: map[string]string{"a.txt": ident}, side: map[string]string{"a.txt": ident}},
		{name: "u43-empty-trees", base: map[string]string{}, side: map[string]string{}},

		// UC-U44 basename + directory
		{name: "u44-basename-breaks-tie", base: map[string]string{"src/foo.go": body}, side: map[string]string{"pkg/foo.go": body, "pkg/bar.go": body}},
		{name: "u44-two-same-basename", base: map[string]string{"src/foo.go": body}, side: map[string]string{"pkg/foo.go": body, "other/foo.go": body}, competingDests: true},
		{name: "u44-basename-near-miss", base: map[string]string{"src/foo.go": ten}, side: map[string]string{"pkg/foo.go": half}},
		{name: "u44-diff-basename-half", base: map[string]string{"src/foo.go": ten}, side: map[string]string{"pkg/bar.go": half}},
		{name: "u44-low-sim-same-basename", base: map[string]string{"src/foo.go": ten}, side: map[string]string{"pkg/foo.go": unrel}},
		{name: "u44-dir-move", base: map[string]string{"pkg/a.txt": ident, "pkg/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"lib/a.txt": ident, "lib/b.txt": ident, "lib/c.txt": ident}},
		{name: "u44-dir-move-competing", base: map[string]string{"pkg/a.txt": ident, "pkg/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"lib/a.txt": ident, "lib/b.txt": ident, "lib/c.txt": ident, "extra/a.txt": ident}},
		{name: "u44-dir-split", base: map[string]string{"pkg/a.txt": ident, "pkg/b.txt": ident, "pkg/c.txt": ident, "pkg/d.txt": ident}, side: map[string]string{"d1/a.txt": ident, "d1/b.txt": ident, "d2/a.txt": ident, "d2/c.txt": ident, "d2/d.txt": ident}},

		// UC-U45 flatten / re-nest (competing dests so leftovers reach flatten)
		{name: "u45-flatten-competing", base: map[string]string{"pkg/a.txt": ident, "pkg/sub/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"a.txt": ident, "sub/b.txt": ident, "c.txt": ident, "extra/a.txt": ident}},
		{name: "u45-renest-competing", base: map[string]string{"pkg/a.txt": ident, "pkg/sub/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"lib/pkg/a.txt": ident, "lib/pkg/sub/b.txt": ident, "lib/pkg/c.txt": ident, "extra/a.txt": ident}},
		{name: "u45-flatten-renest-split", competingDests: true,
			base: map[string]string{"pkg/a.txt": ident, "pkg/sub/b.txt": ident, "pkg/c.txt": ident, "pkg/d.txt": ident},
			side: map[string]string{
				"a.txt": ident, "extra/a.txt": ident,
				"sub/b.txt": ident, "extra/b.txt": ident,
				"lib/pkg/c.txt": ident, "extra/c.txt": ident,
				"lib/pkg/d.txt": ident, "extra/d.txt": ident,
			}},
		{name: "u45-flatten-low-sim", base: map[string]string{"pkg/a.txt": ten, "pkg/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"a.txt": unrel, "b.txt": ident, "c.txt": ident}},

		// UC-U46 prefix-replace / N-level (competing dests so leftovers reach flatten)
		{name: "u46-nlevel-competing", base: map[string]string{"pkg/a.txt": ident, "pkg/sub/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"vendor/lib/pkg/a.txt": ident, "vendor/lib/pkg/sub/b.txt": ident, "vendor/lib/pkg/c.txt": ident, "extra/a.txt": ident}},
		{name: "u46-prefix-replace-competing", base: map[string]string{"pkg/a.txt": ident, "pkg/sub/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"lib/a.txt": ident, "lib/sub/b.txt": ident, "lib/c.txt": ident, "extra/a.txt": ident}},
		{name: "u46-prefix-nlevel-split", competingDests: true,
			base: map[string]string{"pkg/a.txt": ident, "pkg/sub/b.txt": ident, "pkg/c.txt": ident, "pkg/d.txt": ident},
			side: map[string]string{
				"lib/a.txt": ident, "extra/a.txt": ident,
				"lib/sub/b.txt": ident, "extra/b.txt": ident,
				"vendor/lib/pkg/c.txt": ident, "extra/c.txt": ident,
				"vendor/lib/pkg/d.txt": ident, "extra/d.txt": ident,
			}},
		{name: "u46-prefix-low-sim", base: map[string]string{"pkg/a.txt": ten, "pkg/b.txt": ident, "pkg/c.txt": ident}, side: map[string]string{"lib/a.txt": unrel, "lib/b.txt": ident, "lib/c.txt": ident}},
	}
}

func assertMatchRenameInvariants(t *testing.T, base, side map[string]string) {
	t.Helper()
	got := matchRenames(base, side)
	deleted, added := fileDeleteAdds(base, side)
	taken := matchPerFile(base, side, deleted, added)

	stayed := make(map[string]bool, len(base))
	for p := range base {
		if _, ok := side[p]; ok {
			stayed[p] = true
		}
	}

	seenDest := make(map[string]string, len(got))
	for old, neu := range got {
		if stayed[old] || stayed[neu] {
			t.Fatalf("path that remains on the renaming side was paired: %q → %q", old, neu)
		}
		if _, ok := base[old]; !ok {
			t.Fatalf("rename src %q is not a base path", old)
		}
		if _, ok := side[neu]; !ok {
			t.Fatalf("rename dest %q is not a side path", neu)
		}
		if _, ok := side[old]; ok {
			t.Fatalf("rename src %q is still on the side (copy)", old)
		}
		if _, ok := base[neu]; ok {
			t.Fatalf("rename dest %q is still in base", neu)
		}
		if prev, ok := seenDest[neu]; ok {
			t.Fatalf("two srcs %q and %q map to %q", prev, old, neu)
		}
		seenDest[neu] = old
		if fileIsEmpty(base[old]) || fileIsEmpty(side[neu]) {
			t.Fatalf("empty files are never matched: %q → %q", old, neu)
		}
		if s := pairScore(old, neu, base[old], side[neu]); !meetsRenameThreshold(s) {
			t.Fatalf("pair below combined 3/5 gate: %q → %q score=%+v", old, neu, s)
		}
	}

	for _, d := range deleted {
		if !combinedScoreTied(d, base[d], added, func(p string) string { return side[p] }) {
			continue
		}
		if dest, ok := taken[d]; ok {
			t.Fatalf("combined-score tie must refuse a unique per-file match, got %q → %q", d, dest)
		}
	}

	if hasUniqueStrictMajorityMap(deleted, added, taken) {
		return
	}
	for _, d := range deleted {
		if _, ok := taken[d]; ok {
			continue
		}
		if dest, ok := got[d]; ok {
			t.Fatalf("no unique strict majority, leftover must stay unmatched: %q → %q", d, dest)
		}
	}
}

func fileIsEmpty(s string) bool {
	return len(splitLinesKeepEOL(s)) == 0
}

// combinedScoreTied reports a uniqueBest tie: two or more candidates share
// the best combined score at or above the 3/5 gate.
func combinedScoreTied(oldPath, target string, cands []string, content func(string) string) bool {
	var best simFrac
	found := false
	ties := 0
	for _, c := range cands {
		s := pairScore(oldPath, c, target, content(c))
		if !meetsRenameThreshold(s) {
			continue
		}
		if !found {
			best, found, ties = s, true, 1
			continue
		}
		switch cmpSim(s, best) {
		case 1:
			best, ties = s, 1
		case 0:
			ties++
		}
	}
	return found && ties > 1
}

// hasUniqueStrictMajorityMap is an independent majority check over the
// same votes matchDirRenames / matchFlattenRenest use (taken pair or
// unique unused same-basename add). If nothing has a unique strict
// majority, leftovers must stay unmatched.
func hasUniqueStrictMajorityMap(deleted, added []string, taken map[string]string) bool {
	usedDest := make(map[string]bool, len(taken))
	for _, a := range taken {
		usedDest[a] = true
	}
	byBase := uniqueUnusedByBase(added, usedDest)

	srcCount := map[string]int{}
	dirVotes := map[string]map[string]int{}
	prefixCount := map[string]int{}
	xformVotes := map[string]map[string]int{}
	addVote := func(votes map[string]map[string]int, key, choice string) {
		if key == "" || choice == "" {
			return
		}
		m := votes[key]
		if m == nil {
			m = map[string]int{}
			votes[key] = m
		}
		m[choice]++
	}

	for _, d := range deleted {
		if p := pathParent(d); p != "" {
			srcCount[p]++
		}
		for _, pref := range pathPrefixes(d) {
			prefixCount[pref]++
		}
		dest, ok := voteDest(d, taken, byBase)
		if !ok {
			continue
		}
		src, dst := pathParent(d), pathParent(dest)
		if src != "" && dst != "" && src != dst {
			addVote(dirVotes, src, dst)
		}
		oldP, newP, ok := stripAdd(d, dest)
		if !ok {
			continue
		}
		if oldP != "" {
			addVote(xformVotes, oldP, replaceXform+newP)
			continue
		}
		if newP == "" {
			continue
		}
		for _, pref := range pathPrefixes(d) {
			addVote(xformVotes, pref, renestXform+newP)
		}
	}
	return uniqueStrictMajority(srcCount, dirVotes) || uniqueStrictMajority(prefixCount, xformVotes)
}

func uniqueStrictMajority(count map[string]int, votes map[string]map[string]int) bool {
	for key, n := range count {
		if n < 2 {
			continue
		}
		best, bestN := "", 0
		tied := false
		for choice, c := range votes[key] {
			if c > bestN {
				best, bestN, tied = choice, c, false
			} else if c == bestN {
				tied = true
			}
		}
		if !tied && best != "" && bestN*2 > n {
			return true
		}
	}
	return false
}

// walkPathPrefixes is an independent parent walk with a hard cap so a
// pathParent(p)==p bug cannot loop.
func walkPathPrefixes(p string) []string {
	var out []string
	for i := 0; i < len(p)+2; i++ {
		next := pathParent(p)
		if next == "" || next == p {
			return out
		}
		out = append(out, next)
		p = next
	}
	return append(out, "<unterminated>")
}

func encodeRenameTree(m map[string]string) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteByte(0)
		buf.WriteString(m[k])
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

func decodeRenameTree(enc []byte) map[string]string {
	parts := bytes.Split(enc, []byte{0})
	out := make(map[string]string)
	for i := 0; i+1 < len(parts) && len(out) < fuzzMaxFiles; i += 2 {
		p := string(parts[i])
		c := string(parts[i+1])
		if p == "" {
			continue
		}
		if len(p) > fuzzMaxPathLen {
			p = p[:fuzzMaxPathLen]
		}
		c = boundFuzzContent(c)
		if _, exists := out[p]; exists {
			continue
		}
		out[p] = c
	}
	return out
}

func boundFuzzContent(c string) string {
	if len(c) > fuzzMaxContent {
		c = c[:fuzzMaxContent]
	}
	lines := splitLinesKeepEOL(c)
	if len(lines) > fuzzMaxLines {
		return strings.Join(lines[:fuzzMaxLines], "")
	}
	return c
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func stringSlicesEqual(a, b []string) bool {
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
