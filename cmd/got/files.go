package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/repo"
)

// File vertices carry a real file's bytes into the graph so ordinary commits,
// branches, merges, and checkouts version source code — not just abstract
// nodes. A file is an Artifact vertex named by its repository-relative path
// (so its content-addressed id is stable across edits at that path), with its
// bytes base64-encoded under fileContentAttr and its permission bits under
// fileModeAttr. The presence of filePathAttr is what marks a vertex as a file.
const (
	filePathAttr    = "file.path"
	fileContentAttr = "file.content"
	fileModeAttr    = "file.mode"
)

// cmdAdd ingests one or more files or directories into the working graph as
// file vertices (git's `add`). Directories are walked recursively; the repo
// state directory and any nested VCS metadata are skipped.
func cmdAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "add: expected <path>...")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	paths, err := collectFiles(fs.Args())
	if err != nil {
		fmt.Fprintf(stderr, "add: %v\n", err)
		return 1
	}
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "add: no files matched")
		return 1
	}

	g := state.Graph()
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintf(stderr, "add: %v\n", err)
			return 1
		}
		content, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "add: %v\n", err)
			return 1
		}
		rel := filepath.ToSlash(filepath.Clean(p))
		attrs := graph.AttrMap{
			nameAttr:        rel,
			filePathAttr:    rel,
			fileContentAttr: base64.StdEncoding.EncodeToString(content),
			fileModeAttr:    fmt.Sprintf("%o", info.Mode().Perm()),
		}
		g, err = g.WithVertex(graph.Vertex{ID: vid(rel), Type: ontology.Artifact, Attrs: attrs})
		if err != nil {
			fmt.Fprintf(stderr, "add: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "added %s (%d bytes)\n", rel, len(content))
	}
	if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
		fmt.Fprintf(stderr, "add: %v\n", err)
		return 1
	}
	return 0
}

// cmdExtract writes every file vertex in the working graph back to disk under
// the target directory (default "."). This is the checkout-to-worktree step:
// after `got checkout <branch>` rebuilds the working graph, `extract` renders
// that state as real files.
func cmdExtract(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dir := "."
	if fs.NArg() == 1 {
		dir = fs.Arg(0)
	} else if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "extract: expected an optional <dir>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	files := fileVertices(state.Graph())
	if len(files) == 0 {
		fmt.Fprintln(stdout, "no file vertices to extract")
		return 0
	}
	// Deterministic order for stable output.
	rels := make([]string, 0, len(files))
	for rel := range files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	for _, rel := range rels {
		v := files[rel]
		b64, _ := v.Attrs[fileContentAttr].(string)
		content, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			fmt.Fprintf(stderr, "extract: %s: %v\n", rel, err)
			return 1
		}
		dest, err := safeJoin(dir, rel)
		if err != nil {
			fmt.Fprintf(stderr, "extract: %v\n", err)
			return 1
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			fmt.Fprintf(stderr, "extract: %v\n", err)
			return 1
		}
		mode := os.FileMode(0o644)
		if ms, ok := v.Attrs[fileModeAttr].(string); ok {
			if parsed, err := strconv.ParseUint(ms, 8, 32); err == nil {
				mode = os.FileMode(parsed)
			}
		}
		if err := os.WriteFile(dest, content, mode); err != nil {
			fmt.Fprintf(stderr, "extract: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "extracted %s (%d bytes)\n", rel, len(content))
	}
	return 0
}

// fileVertices returns the working graph's file vertices keyed by their
// repository-relative path.
func fileVertices(g graph.Graph) map[string]graph.Vertex {
	out := make(map[string]graph.Vertex)
	for _, v := range g.Vertices() {
		if p, ok := v.Attrs[filePathAttr].(string); ok {
			out[p] = v
		}
	}
	return out
}

// collectFiles expands the given paths into a sorted, de-duplicated list of
// regular files, walking directories and skipping VCS metadata.
func collectFiles(args []string) ([]string, error) {
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		c := filepath.Clean(p)
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	skipDir := stateDir()
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			add(arg)
			continue
		}
		err = filepath.WalkDir(arg, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				base := d.Name()
				if base == ".git" || base == skipDir || p == skipDir {
					return filepath.SkipDir
				}
				return nil
			}
			add(p)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

// safeJoin joins dir and a repo-relative path, refusing paths that would escape
// dir (absolute, or traversing above it).
func safeJoin(dir, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe path %q", rel)
	}
	return filepath.Join(dir, clean), nil
}
