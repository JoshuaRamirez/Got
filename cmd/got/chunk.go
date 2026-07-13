package main

import (
	"fmt"
	"strings"
)

// A chunk is a contiguous slice of a file that acts as an independently
// mergeable unit. The block chunker below is the tier-2 (language-agnostic)
// decomposition described in the design notes: it needs no parser, so it works
// on any text, and it is the seam a future language-aware (tree-sitter or
// go/ast) chunker would slot behind by satisfying the same contract.
//
// Key is a *stable* identity for the chunk within its file — stable across edits
// to the chunk's body, so a three-way merge aligns "the same chunk, edited" on
// both sides rather than seeing a delete+add. Content is the exact bytes,
// newline included, so Split→Join reproduces the file verbatim.
type chunk struct {
	Key     string
	Content string
}

// chunker decomposes a file into ordered chunks and reassembles them. Split then
// Join must be the identity on any input.
type chunker interface {
	Split(content string) []chunk
	Join(chunks []chunk) string
}

// blockChunker splits a file into top-level brace blocks and standalone lines.
// A run of lines is accumulated until the running brace depth returns to zero at
// a line boundary, so a `func F() { ... }` (or any `{...}` block) becomes one
// chunk while lines outside braces (package, imports, blanks) are one chunk each.
//
// Brace counting is deliberately naive — it does not exclude braces inside
// strings or comments. That is an honest limitation of the parser-free tier: it
// merges correctly whenever the two sides touch *different* top-level blocks,
// which is the case git most often gets wrong, and falls back to a whole-file
// conflict otherwise.
type blockChunker struct{}

func newBlockChunker() blockChunker { return blockChunker{} }

func (blockChunker) Split(content string) []chunk {
	if content == "" {
		return nil
	}
	lines := splitLinesKeepEOL(content)

	var raw []string // chunk bodies before key assignment
	var cur strings.Builder
	depth := 0
	for _, ln := range lines {
		cur.WriteString(ln)
		depth += braceDelta(ln)
		if depth <= 0 {
			depth = 0
			raw = append(raw, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 { // unbalanced tail: emit what remains
		raw = append(raw, cur.String())
	}

	// Assign stable keys: a chunk's signature is its first non-blank, trimmed
	// line. Duplicate signatures within one file are disambiguated by an
	// occurrence counter so distinct blocks never collapse to one vertex.
	seen := make(map[string]int)
	chunks := make([]chunk, 0, len(raw))
	for _, body := range raw {
		sig := signature(body)
		n := seen[sig]
		seen[sig] = n + 1
		chunks = append(chunks, chunk{Key: fmt.Sprintf("%s#%d", sig, n), Content: body})
	}
	return chunks
}

func (blockChunker) Join(chunks []chunk) string {
	var b strings.Builder
	for _, c := range chunks {
		b.WriteString(c.Content)
	}
	return b.String()
}

// signature is the first non-blank line of a chunk body, trimmed — the part that
// stays constant while the body is edited (e.g. a function's declaration line).
func signature(body string) string {
	for _, ln := range strings.Split(body, "\n") {
		t := strings.TrimSpace(ln)
		if t != "" {
			return t
		}
	}
	return "" // all-blank chunk
}

// braceDelta is the net change in brace nesting contributed by a line.
func braceDelta(line string) int {
	d := 0
	for _, r := range line {
		switch r {
		case '{':
			d++
		case '}':
			d--
		}
	}
	return d
}

// splitLinesKeepEOL breaks content into lines, each retaining its trailing "\n"
// (the final line keeps whatever it had), so concatenation is lossless.
func splitLinesKeepEOL(content string) []string {
	var out []string
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			out = append(out, content[start:i+1])
			start = i + 1
		}
	}
	if start < len(content) {
		out = append(out, content[start:])
	}
	return out
}
