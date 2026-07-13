package main

import "testing"

// Split then Join must reproduce the input exactly, for any text.
func TestBlockChunkerRoundTrip(t *testing.T) {
	ch := newBlockChunker()
	cases := []string{
		"",
		"one line no newline",
		"a\nb\nc\n",
		"package main\n\nfunc F() int {\n\treturn 1\n}\n",
		"nested {\n  inner {\n    x\n  }\n}\ntail\n",
		"trailing\nblank\n\n",
	}
	for _, in := range cases {
		if got := ch.Join(ch.Split(in)); got != in {
			t.Fatalf("round-trip mismatch:\n in=%q\nout=%q", in, got)
		}
	}
}

// A brace block is one chunk; standalone lines are their own chunks.
func TestBlockChunkerGroupsBraceBlocks(t *testing.T) {
	ch := newBlockChunker()
	src := "package main\n\nfunc F() int {\n\treturn 1\n}\n\nfunc G() int {\n\treturn 2\n}\n"
	chunks := ch.Split(src)
	var funcs int
	for _, c := range chunks {
		body := c.Content
		if len(body) > 4 && body[:4] == "func" {
			funcs++
			// The whole function body is in one chunk.
			if !containsAll(body, "func", "return", "}") {
				t.Fatalf("function chunk not whole: %q", body)
			}
		}
	}
	if funcs != 2 {
		t.Fatalf("expected 2 function chunks, got %d", funcs)
	}
}

// A chunk's key is stable across edits to its body — so a three-way merge sees
// "same chunk, edited", not delete+add.
func TestBlockChunkerKeyStableAcrossBodyEdit(t *testing.T) {
	ch := newBlockChunker()
	before := ch.Split("func F() int {\n\treturn 1\n}\n")
	after := ch.Split("func F() int {\n\treturn 999\n}\n")
	if len(before) != 1 || len(after) != 1 {
		t.Fatalf("expected single chunks, got %d and %d", len(before), len(after))
	}
	if before[0].Key != after[0].Key {
		t.Fatalf("key changed on body edit: %q vs %q", before[0].Key, after[0].Key)
	}
}

// Distinct blocks get distinct keys; duplicate signatures are disambiguated.
func TestBlockChunkerDistinctKeys(t *testing.T) {
	ch := newBlockChunker()
	chunks := ch.Split("x := 1\nx := 1\ny := 2\n")
	keys := map[string]bool{}
	for _, c := range chunks {
		if keys[c.Key] {
			t.Fatalf("duplicate key %q", c.Key)
		}
		keys[c.Key] = true
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 distinct keys, got %d", len(keys))
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
