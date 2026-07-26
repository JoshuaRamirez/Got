package main

import "testing"

// mergeChunkOrder is the three-way merge of the chunk key sequence. body holds
// the surviving keys (values are irrelevant to ordering).
func TestMergeChunkOrder(t *testing.T) {
	ch := func(keys ...string) []chunk {
		out := make([]chunk, len(keys))
		for i, k := range keys {
			out[i] = chunk{Key: k}
		}
		return out
	}
	survive := func(keys ...string) map[string]string {
		m := map[string]string{}
		for _, k := range keys {
			m[k] = ""
		}
		return m
	}
	eq := func(got []string, ok bool, wantOK bool, want ...string) {
		t.Helper()
		if ok != wantOK {
			t.Fatalf("ok=%v want %v (got seq %v)", ok, wantOK, got)
		}
		if !ok {
			return
		}
		if len(got) != len(want) {
			t.Fatalf("seq=%v want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("seq=%v want %v", got, want)
			}
		}
	}

	// No reorder + one-sided additions → base order then left adds then right adds
	// (identical to the pre-order-merge behavior).
	seq, ok := mergeChunkOrder(ch("A", "B"), ch("A", "B", "L"), ch("A", "B", "R"), survive("A", "B", "L", "R"))
	eq(seq, ok, true, "A", "B", "L", "R")

	// One side reordered → that side's order is authority.
	seq, ok = mergeChunkOrder(ch("A", "B"), ch("B", "A"), ch("A", "B"), survive("A", "B"))
	eq(seq, ok, true, "B", "A")

	// One side reordered AND added; the other only edited → reorder + add preserved.
	seq, ok = mergeChunkOrder(ch("A", "B"), ch("B", "A", "L"), ch("A", "B"), survive("A", "B", "L"))
	eq(seq, ok, true, "B", "A", "L")

	// Both reordered incompatibly → conflict (ok == false).
	seq, ok = mergeChunkOrder(ch("A", "B", "C"), ch("C", "B", "A"), ch("B", "A", "C"), survive("A", "B", "C"))
	eq(seq, ok, false)

	// Both reordered identically → accept.
	seq, ok = mergeChunkOrder(ch("A", "B", "C"), ch("C", "A", "B"), ch("C", "A", "B"), survive("A", "B", "C"))
	eq(seq, ok, true, "C", "A", "B")

	// A deletion honored by the content merge simply drops out of the sequence.
	seq, ok = mergeChunkOrder(ch("A", "B"), ch("B", "A"), ch("A"), survive("A"))
	eq(seq, ok, true, "A")
}
