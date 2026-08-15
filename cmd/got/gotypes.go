package main

import (
	"encoding/base64"
	"go/ast"
	"go/build"
	"go/build/constraint"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"runtime"
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
// Scope is deliberately honest and bounded to a *within-package* net, not a
// whole-program build. It tolerates (never refuses on): parse errors (the
// structural gate owns syntax), imports it cannot resolve (the sandbox lacks the
// module's own internal deps), unused-import/variable warnings, and the
// undefined-name cascade that follows an unresolved dot import. It respects
// build constraints (so mutually-exclusive platform files are not judged as
// duplicates) and partitions a directory by declared package (so an external
// `_test` package does not mask the production package). It returns ok == true
// whenever it cannot make a confident negative judgement.
func semanticGateOK(merged graph.Graph, base graph.Snapshot) (bool, string) {
	mergedFiles := goOnly(fileContents(merged))
	baseFiles := goOnly(fileContentByPath(base))

	// A directory is "changed" if any .go file in it was added, edited, or
	// deleted — deletions included, so removing a helper used elsewhere is gated.
	changed := make(map[string]bool)
	for p, c := range mergedFiles {
		if bc, ok := baseFiles[p]; !ok || bc != c {
			changed[filepath.Dir(p)] = true
		}
	}
	for p := range baseFiles {
		if _, ok := mergedFiles[p]; !ok {
			changed[filepath.Dir(p)] = true
		}
	}

	byDir := filesByDir(mergedFiles)
	dirs := sortedKeys(changed)
	for _, dir := range dirs {
		if ok, detail := typeCheckDir(dir, byDir[dir]); !ok {
			return false, detail
		}
	}
	return true, ""
}

// typeCheckDir filters a directory's files to those the host build context would
// compile, partitions them by declared package, and type-checks each package.
func typeCheckDir(dir string, byPath map[string]string) (bool, string) {
	fset := token.NewFileSet()
	pkgs := make(map[string][]*ast.File)
	pkgOrder := []string{}
	for _, p := range sortedKeys(byPath) {
		if !buildMatchesHost(p, byPath[p]) {
			continue // constrained to another platform; not part of this build
		}
		f, err := parser.ParseFile(fset, p, byPath[p], parser.ParseComments)
		if err != nil {
			return true, "" // not valid Go syntax — the structural gate owns this
		}
		name := f.Name.Name
		if _, seen := pkgs[name]; !seen {
			pkgOrder = append(pkgOrder, name)
		}
		pkgs[name] = append(pkgs[name], f)
	}
	sort.Strings(pkgOrder)
	for _, name := range pkgOrder {
		if ok, detail := typeCheckPackage(dir, fset, pkgs[name]); !ok {
			return false, detail
		}
	}
	return true, ""
}

// typeCheckPackage type-checks one package's files and classifies the errors.
// Redeclarations are always fatal; undefined names are fatal unless they are the
// cascade from an unresolved dot import.
func typeCheckPackage(dir string, fset *token.FileSet, files []*ast.File) (bool, string) {
	if len(files) == 0 {
		return true, ""
	}
	// Paths of dot imports (`import . "x"`), which inject names — if one of these
	// specifically fails to resolve, undefined-name errors are its cascade and
	// must be tolerated. A *different* failed import must NOT license tolerating
	// undefined names.
	dotPaths := make(map[string]bool)
	for _, f := range files {
		for _, imp := range f.Imports {
			if imp.Name != nil && imp.Name.Name == "." {
				dotPaths[importPath(imp)] = true
			}
		}
	}

	var msgs []string
	cfg := &types.Config{
		Importer:                 importer.Default(),
		DisableUnusedImportCheck: true,
		Error: func(err error) {
			if te, ok := err.(types.Error); ok {
				msgs = append(msgs, te.Msg)
			}
		},
	}
	_, _ = cfg.Check(dir, fset, files, nil)

	dotImportFailed := false
	for _, m := range msgs {
		if p, ok := couldNotImportPath(m); ok && dotPaths[p] {
			dotImportFailed = true
			break
		}
	}
	for _, m := range msgs {
		if strings.Contains(m, "redeclared") {
			return false, m
		}
	}
	for _, m := range msgs {
		if strings.Contains(m, "undefined:") || strings.Contains(m, "undeclared name") {
			if dotImportFailed {
				continue // names injected by an unresolvable dot import; tolerate
			}
			return false, m
		}
	}
	return true, ""
}

// couldNotImportPath extracts the import path from a go/types
// "could not import <path> (...)" error message.
func couldNotImportPath(msg string) (string, bool) {
	const pfx = "could not import "
	i := strings.Index(msg, pfx)
	if i < 0 {
		return "", false
	}
	rest := msg[i+len(pfx):]
	if j := strings.Index(rest, " ("); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest), true
}

// --- build-constraint filtering ---

// buildMatchesHost reports whether a file is compiled for the host GOOS/GOARCH,
// honoring both the filename `_GOOS_GOARCH` suffix convention and a `//go:build`
// constraint line. Used so mutually-exclusive platform files (e.g. foo_linux.go
// and foo_windows.go) are not type-checked together and mis-reported as
// duplicate declarations.
func buildMatchesHost(path, content string) bool {
	if !filenameMatchesHost(path) {
		return false
	}
	if expr := buildExpr(content); expr != nil {
		return expr.Eval(hostSatisfiesTag)
	}
	return true
}

func filenameMatchesHost(path string) bool {
	name := strings.TrimSuffix(filepath.Base(path), ".go")
	name = strings.TrimSuffix(name, "_test")
	parts := strings.Split(name, "_")
	n := len(parts)
	// _GOOS_GOARCH (needs a prefix component before the two tags)
	if n >= 3 && knownOS[parts[n-2]] && knownArch[parts[n-1]] {
		return parts[n-2] == runtime.GOOS && parts[n-1] == runtime.GOARCH
	}
	// _GOARCH or _GOOS (needs a prefix component)
	if n >= 2 && knownArch[parts[n-1]] {
		return parts[n-1] == runtime.GOARCH
	}
	if n >= 2 && knownOS[parts[n-1]] {
		return parts[n-1] == runtime.GOOS
	}
	return true
}

// buildExpr extracts a //go:build constraint from a file's leading comments.
func buildExpr(content string) constraint.Expr {
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "//") {
			if constraint.IsGoBuild(t) {
				if e, err := constraint.Parse(t); err == nil {
					return e
				}
			}
			continue
		}
		break // first non-comment, non-blank line ends the constraint zone
	}
	return nil
}

// hostSatisfiesTag evaluates a build tag against the host build context. It uses
// build.Default so goN.M release tags and cgo track the real toolchain rather
// than a hand-rolled approximation. Unknown tags are unsatisfied.
func hostSatisfiesTag(tag string) bool {
	switch tag {
	case runtime.GOOS, runtime.GOARCH, "gc":
		return true
	case "cgo":
		return build.Default.CgoEnabled
	case "unix":
		return unixGOOS[runtime.GOOS]
	}
	for _, rt := range build.Default.ReleaseTags {
		if tag == rt {
			return true
		}
	}
	return false
}

var knownOS = set("aix", "android", "darwin", "dragonfly", "freebsd", "hurd",
	"illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9",
	"solaris", "wasip1", "windows", "zos")

var knownArch = set("386", "amd64", "amd64p32", "arm", "arm64", "arm64be",
	"armbe", "loong64", "mips", "mips64", "mips64le", "mips64p32", "mips64p32le",
	"mipsle", "ppc", "ppc64", "ppc64le", "riscv", "riscv64", "s390", "s390x",
	"sparc", "sparc64", "wasm")

var unixGOOS = set("aix", "android", "darwin", "dragonfly", "freebsd", "hurd",
	"illumos", "ios", "linux", "netbsd", "openbsd", "solaris")

func set(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// --- helpers ---

func goOnly(byPath map[string]string) map[string]string {
	out := make(map[string]string)
	for p, c := range byPath {
		if strings.HasSuffix(p, ".go") {
			out[p] = c
		}
	}
	return out
}

func filesByDir(byPath map[string]string) map[string]map[string]string {
	out := make(map[string]map[string]string)
	for p, content := range byPath {
		dir := filepath.Dir(p)
		if out[dir] == nil {
			out[dir] = make(map[string]string)
		}
		out[dir][p] = content
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
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
