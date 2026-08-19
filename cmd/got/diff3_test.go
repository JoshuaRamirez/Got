package main

import (
	"strings"
	"testing"
)

func TestDiff3(t *testing.T) {
	sp := splitLinesKeepEOL
	merge := func(base, left, right string) (string, bool) {
		m, ok := diff3(sp(base), sp(left), sp(right))
		return strings.Join(m, ""), ok
	}

	// Disjoint edits (different regions) merge, preserving order.
	if got, ok := merge("a\nb\nc\nd\ne\n", "A\nb\nc\nd\ne\n", "a\nb\nc\nd\nE\n"); !ok || got != "A\nb\nc\nd\nE\n" {
		t.Fatalf("disjoint edits: ok=%v got=%q", ok, got)
	}
	// One side unchanged → take the other.
	if got, ok := merge("a\nb\n", "a\nb\n", "a\nB\n"); !ok || got != "a\nB\n" {
		t.Fatalf("one-sided: ok=%v got=%q", ok, got)
	}
	// Both make the identical change → accept once.
	if got, ok := merge("a\nb\n", "a\nX\n", "a\nX\n"); !ok || got != "a\nX\n" {
		t.Fatalf("same change: ok=%v got=%q", ok, got)
	}
	// Both change the same line differently → conflict.
	if _, ok := merge("a\n", "b\n", "c\n"); ok {
		t.Fatal("divergent same-line change must conflict")
	}
	// Insertion far from the other side's edit merges.
	if got, ok := merge("a\nb\nc\nd\n", "a\nINS\nb\nc\nd\n", "a\nb\nc\nD\n"); !ok || got != "a\nINS\nb\nc\nD\n" {
		t.Fatalf("insert + distant edit: ok=%v got=%q", ok, got)
	}
	// UC-U43: insertion immediately next to the other side's edit — coarse
	// diff3 lumps these into one gap; hunk refinement must merge them.
	if got, ok := merge("a\nb\nc\n", "a\nINS\nb\nc\n", "a\nB\nc\n"); !ok || got != "a\nINS\nB\nc\n" {
		t.Fatalf("adjacent insert vs edit: ok=%v got=%q", ok, got)
	}
	if got, ok := merge("a\nb\n", "A\nb\n", "a\nINS\nb\n"); !ok || got != "A\nINS\nb\n" {
		t.Fatalf("adjacent edit vs insert: ok=%v got=%q", ok, got)
	}
	// Same-point divergent inserts still conflict.
	if _, ok := merge("a\nc\n", "a\nX\nc\n", "a\nY\nc\n"); ok {
		t.Fatal("divergent inserts at the same point must conflict")
	}
	// Deletion on one side, unchanged on the other → deletion applied.
	if got, ok := merge("a\nb\nc\n", "a\nc\n", "a\nb\nc\n"); !ok || got != "a\nc\n" {
		t.Fatalf("one-sided delete: ok=%v got=%q", ok, got)
	}
	// diff3Merge gates the result: an invalid-Go line merge is refused.
	if _, ok := diff3Merge("m.go", "package p\n\nfunc A() int { return 1 }\n",
		"package p\n\nfunc A() int { return 1 }\nfunc A() int { return 2 }\n",
		"package p\n\nfunc A() int { return 1 }\n"); ok {
		// left introduced a duplicate A; even though it line-merges, the gate refuses.
		t.Fatal("diff3Merge must reject invalid-Go results")
	}
}
