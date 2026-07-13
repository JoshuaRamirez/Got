package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// goChunker is the tier-4 (language-aware) chunker for Go, slotting behind the
// same `chunker` interface as blockChunker. It parses the file and chunks it at
// top-level declaration boundaries, keyed by the *symbol* each declaration
// introduces (func name, receiver.method, type/var/const name) rather than by a
// text signature. That gives two things the block chunker cannot:
//
//   - Body-edit and reformat immunity: a chunk's identity is its symbol, so
//     editing or reindenting a function's body keeps it aligned across a merge.
//   - Rename alignment: a declaration is the same chunk wherever it moves in the
//     file, because position is not part of its key.
//
// Content is spliced from the original source bytes at declaration offsets — it
// is never reprinted — so Split→Join reproduces the file verbatim (unlike
// go/printer, which would reformat). On a parse error it falls back to the
// block chunker, so it is always safe to use.
type goChunker struct{}

func newGoChunker() goChunker { return goChunker{} }

func (goChunker) Split(content string) []chunk {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return newBlockChunker().Split(content) // not valid Go; degrade gracefully
	}
	src := content

	// A cut is the byte offset where a top-level declaration begins (including
	// its doc comment). The region before the first cut is the preamble
	// (package clause). Chunks tile [cut_i, cut_{i+1}) so Join is lossless.
	type cut struct {
		off int
		key string
	}
	var cuts []cut
	for _, d := range f.Decls {
		start := d.Pos()
		if doc := declDoc(d); doc != nil {
			start = doc.Pos()
		}
		cuts = append(cuts, cut{off: fset.Position(start).Offset, key: declKey(d)})
	}
	sort.SliceStable(cuts, func(i, j int) bool { return cuts[i].off < cuts[j].off })

	// Boundaries, made strictly increasing and bounded by the source length.
	type seg struct {
		start int
		key   string
	}
	segs := []seg{{start: 0, key: "preamble"}}
	last := 0
	for _, c := range cuts {
		if c.off <= last || c.off > len(src) {
			continue // defensive: skip non-monotonic or out-of-range offsets
		}
		segs = append(segs, seg{start: c.off, key: c.key})
		last = c.off
	}

	// Disambiguate duplicate keys (e.g. two var blocks) by occurrence.
	seen := make(map[string]int)
	chunks := make([]chunk, 0, len(segs))
	for i, s := range segs {
		end := len(src)
		if i+1 < len(segs) {
			end = segs[i+1].start
		}
		if s.start >= end {
			continue
		}
		n := seen[s.key]
		seen[s.key] = n + 1
		chunks = append(chunks, chunk{
			Key:     fmt.Sprintf("%s#%d", s.key, n),
			Content: src[s.start:end],
		})
	}
	return chunks
}

func (goChunker) Join(chunks []chunk) string {
	var b strings.Builder
	for _, c := range chunks {
		b.WriteString(c.Content)
	}
	return b.String()
}

// declKey is the stable symbol identity of a top-level declaration.
func declKey(d ast.Decl) string {
	switch x := d.(type) {
	case *ast.FuncDecl:
		if x.Recv != nil {
			return "method:" + recvTypeName(x) + "." + x.Name.Name
		}
		return "func:" + x.Name.Name
	case *ast.GenDecl:
		switch x.Tok {
		case token.IMPORT:
			return "import"
		case token.TYPE:
			if n := firstTypeName(x); n != "" {
				return "type:" + n
			}
		case token.VAR:
			if n := firstValueName(x); n != "" {
				return "var:" + n
			}
		case token.CONST:
			if n := firstValueName(x); n != "" {
				return "const:" + n
			}
		}
	}
	return "decl"
}

func declDoc(d ast.Decl) *ast.CommentGroup {
	switch x := d.(type) {
	case *ast.FuncDecl:
		return x.Doc
	case *ast.GenDecl:
		return x.Doc
	}
	return nil
}

// recvTypeName extracts the base type name of a method receiver, unwrapping
// pointer and generic (T[...]) receivers.
func recvTypeName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	return baseTypeName(fd.Recv.List[0].Type)
}

func baseTypeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.IndexExpr:
		return baseTypeName(t.X)
	case *ast.IndexListExpr:
		return baseTypeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func firstTypeName(g *ast.GenDecl) string {
	for _, s := range g.Specs {
		if ts, ok := s.(*ast.TypeSpec); ok {
			return ts.Name.Name
		}
	}
	return ""
}

func firstValueName(g *ast.GenDecl) string {
	for _, s := range g.Specs {
		if vs, ok := s.(*ast.ValueSpec); ok && len(vs.Names) > 0 {
			return vs.Names[0].Name
		}
	}
	return ""
}

// goValidityOK reports whether merged Go source is structurally sound: it parses
// AND declares no top-level symbol twice. This is the structural-validity gate —
// a whole-result check the per-chunk merge cannot make, and one git cannot make
// at all because it has no parser. A merge that produces two `func New` (or a
// `const X` colliding with a `var X`) parses as text but is invalid Go; the gate
// refuses to auto-produce it.
func goValidityOK(content string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, 0)
	if err != nil {
		return false
	}
	seen := make(map[string]bool)
	for _, d := range f.Decls {
		for _, name := range topLevelNames(d) {
			if name == "_" {
				continue
			}
			if seen[name] {
				return false // redeclared at package scope
			}
			seen[name] = true
		}
	}
	return true
}

// topLevelNames returns the package-scope identifiers a declaration introduces.
// Methods are namespaced by receiver type, since distinct types may share a
// method name without colliding.
func topLevelNames(d ast.Decl) []string {
	switch x := d.(type) {
	case *ast.FuncDecl:
		if x.Recv != nil {
			return []string{"method:" + recvTypeName(x) + "." + x.Name.Name}
		}
		return []string{x.Name.Name}
	case *ast.GenDecl:
		if x.Tok == token.IMPORT {
			return nil
		}
		var out []string
		for _, s := range x.Specs {
			switch sp := s.(type) {
			case *ast.TypeSpec:
				out = append(out, sp.Name.Name)
			case *ast.ValueSpec:
				for _, n := range sp.Names {
					out = append(out, n.Name)
				}
			}
		}
		return out
	}
	return nil
}
