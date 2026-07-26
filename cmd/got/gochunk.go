package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
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
//   - Move alignment: a declaration is the same chunk wherever it moves in the
//     file, because position is not part of its key. (A *rename* changes the
//     symbol and therefore the key, so it does not align — it looks like a
//     delete plus an add; rename detection is not attempted.)
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
		// A parenthesized import block is decomposed into head "import (",
		// one chunk per spec, and tail ")", so two branches adding different
		// imports become independent chunk additions that the graph engine
		// unions instead of conflicting on the whole block.
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT && g.Lparen.IsValid() {
			head := d.Pos()
			if g.Doc != nil {
				head = g.Doc.Pos()
			}
			cuts = append(cuts, cut{off: fset.Position(head).Offset, key: "import.head"})
			for _, s := range g.Specs {
				is := s.(*ast.ImportSpec)
				at := is.Pos()
				if is.Doc != nil {
					at = is.Doc.Pos()
				}
				cuts = append(cuts, cut{
					off: lineStartOffset(src, fset.Position(at).Offset),
					key: "import:" + importPath(is),
				})
			}
			cuts = append(cuts, cut{
				off: lineStartOffset(src, fset.Position(g.Rparen).Offset),
				key: "import.tail",
			})
			continue
		}
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
			// Only reached for a non-parenthesized single import (the
			// parenthesized case is expanded in Split). Key by its path.
			for _, s := range x.Specs {
				if is, ok := s.(*ast.ImportSpec); ok {
					return "import:" + importPath(is)
				}
			}
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

// importPath returns the unquoted import path of a spec (raw value on error).
func importPath(is *ast.ImportSpec) string {
	if p, err := strconv.Unquote(is.Path.Value); err == nil {
		return p
	}
	return strings.Trim(is.Path.Value, `"`)
}

// importLocalNames returns the file-scope names an import declaration binds:
// the explicit alias when present, otherwise the package-name heuristic
// (path.Base). Blank ("_") and dot (".") imports bind no checkable name and are
// skipped. These names participate in the validity gate because an import name
// shares the file block with package-scope declarations — e.g. `import "fmt"`
// collides with `var fmt = 1`.
func importLocalNames(g *ast.GenDecl) []string {
	var out []string
	for _, s := range g.Specs {
		is, ok := s.(*ast.ImportSpec)
		if !ok {
			continue
		}
		if is.Name != nil {
			if n := is.Name.Name; n != "_" && n != "." {
				out = append(out, n)
			}
			continue
		}
		// No alias: the bound name is the package's own name. path.Base is a
		// heuristic (usually correct; e.g. gopkg.in/yaml.v2 → "yaml.v2" is
		// wrong) — the go/types gate (UC-U40) reads the real package name.
		out = append(out, path.Base(importPath(is)))
	}
	return out
}

// lineStartOffset returns the offset of the start of the line containing off,
// so a chunk boundary at a spec captures that spec's leading indentation.
func lineStartOffset(src string, off int) int {
	if off > len(src) {
		off = len(src)
	}
	i := off
	for i > 0 && src[i-1] != '\n' {
		i--
	}
	return i
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
			return importLocalNames(x)
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
