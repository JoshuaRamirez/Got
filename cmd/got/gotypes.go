package main

import (
	"encoding/base64"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joshuaramirez/got/internal/graph"
)

// semanticGateOK type-checks the merged result and reports whether it is free of
// the semantic breakages a structural, per-file gate cannot see: a symbol
// redeclared across two files of a package, or a reference to a symbol the other
// branch deleted (an undefined name). It is the graph-VCS analogue of "does the
// merge still compile" — a check git cannot make because it has no type system.
//
// Scope is deliberately honest and bounded:
//   - It only type-checks packages that a changed file belongs to (cheap, and it
//     never judges code the merge did not touch).
//   - It tolerates everything that is not an in-package redeclaration or
//     undefined name: parse errors (the structural gate owns syntax), imports it
//     cannot resolve (the sandbox lacks the module's own internal deps), and
//     unused-import/variable warnings. So it is a *within-package* correctness
//     net, not a whole-program build.
//
// It returns ok == true (with empty detail) whenever it cannot make a confident
// negative judgement, so it never blocks a merge on its own uncertainty.
func semanticGateOK(merged graph.Graph, base graph.Snapshot) (bool, string) {
	files := goFilesByDir(fileContents(merged))
	baseContent := fileContentByPath(base)

	// Only gate directories that actually changed in this merge.
	dirs := make([]string, 0, len(files))
	for dir, byPath := range files {
		changed := false
		for p, content := range byPath {
			if baseContent[p] != content {
				changed = true
				break
			}
		}
		if changed {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		if ok, detail := typeCheckPackage(dir, files[dir]); !ok {
			return false, detail
		}
	}
	return true, ""
}

// typeCheckPackage parses and type-checks all .go files of one directory
// together. It returns ok == false only on an in-package redeclaration or
// undefined-name error; parse failures and unresolved imports are tolerated.
func typeCheckPackage(dir string, byPath map[string]string) (bool, string) {
	fset := token.NewFileSet()
	var asts []*ast.File
	names := make([]string, 0, len(byPath))
	for p := range byPath {
		names = append(names, p)
	}
	sort.Strings(names) // deterministic
	for _, p := range names {
		f, err := parser.ParseFile(fset, p, byPath[p], 0)
		if err != nil {
			return true, "" // not valid Go syntax — the structural gate owns this
		}
		asts = append(asts, f)
	}
	if len(asts) == 0 {
		return true, ""
	}

	var fatal string
	cfg := &types.Config{
		Importer:                 importer.Default(),
		DisableUnusedImportCheck: true, // unused-import is not a merge hazard we judge
		Error: func(err error) {
			if fatal != "" {
				return
			}
			te, ok := err.(types.Error)
			if !ok {
				return
			}
			if msg := te.Msg; isMergeHazard(msg) {
				fatal = msg
			}
		},
	}
	// Check reports the first error as its return; we rely on cfg.Error to see
	// them all and to classify. Ignore the returned error.
	_, _ = cfg.Check(dir, fset, asts, nil)
	if fatal != "" {
		return false, fatal
	}
	return true, ""
}

// isMergeHazard classifies a go/types error message as one that indicates the
// merge produced structurally broken code (as opposed to an unresolved external
// import, which the sandbox cannot help). Kept to unambiguous phrases.
func isMergeHazard(msg string) bool {
	switch {
	case strings.Contains(msg, "redeclared"):
		return true
	case strings.Contains(msg, "undefined:"):
		return true
	case strings.Contains(msg, "undeclared name"):
		return true
	default:
		return false
	}
}

// goFilesByDir groups .go file contents by their directory (a package's files
// live in one directory).
func goFilesByDir(byPath map[string]string) map[string]map[string]string {
	out := make(map[string]map[string]string)
	for p, content := range byPath {
		if !strings.HasSuffix(p, ".go") {
			continue
		}
		dir := filepath.Dir(p)
		if out[dir] == nil {
			out[dir] = make(map[string]string)
		}
		out[dir][p] = content
	}
	return out
}

// fileContents decodes every file vertex in a graph to path→text.
func fileContents(g graph.Graph) map[string]string {
	out := make(map[string]string)
	for _, v := range g.Vertices() {
		p, ok := v.Attrs[filePathAttr].(string)
		if !ok {
			continue
		}
		if b64, ok := v.Attrs[fileContentAttr].(string); ok {
			if raw, err := base64.StdEncoding.DecodeString(b64); err == nil {
				out[p] = string(raw)
			}
		}
	}
	return out
}
