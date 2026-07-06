package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/history"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/provenance"
	"github.com/joshuaramirez/got/internal/realization"
	"github.com/joshuaramirez/got/internal/repo"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

// run is the testable entry point: it dispatches a command, writes output to
// stdout/stderr, and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "init":
		return cmdInit(rest, stdout, stderr)
	case "add-vertex":
		return cmdAddVertex(rest, stdout, stderr)
	case "add-edge":
		return cmdAddEdge(rest, stdout, stderr)
	case "bind":
		return cmdBind(rest, stdout, stderr)
	case "resolve":
		return cmdResolve(rest, stdout, stderr)
	case "branch":
		return cmdBranch(rest, stdout, stderr)
	case "branches":
		return cmdBranches(rest, stdout, stderr)
	case "branch-log":
		return cmdBranchLog(rest, stdout, stderr)
	case "commit":
		return cmdCommit(rest, stdout, stderr)
	case "log":
		return cmdLog(rest, stdout, stderr)
	case "diff":
		return cmdDiff(rest, stdout, stderr)
	case "checkout", "switch":
		return cmdCheckout(rest, stdout, stderr)
	case "status":
		return cmdStatus(rest, stdout, stderr)
	case "merge-base":
		return cmdMergeBase(rest, stdout, stderr)
	case "show":
		return cmdShow(rest, stdout, stderr)
	case "tag":
		return cmdTag(rest, stdout, stderr)
	case "tags":
		return cmdTag(nil, stdout, stderr)
	case "revert":
		return cmdRevert(rest, stdout, stderr)
	case "reset":
		return cmdReset(rest, stdout, stderr)
	case "restore":
		return cmdRestore(rest, stdout, stderr)
	case "blame":
		return cmdBlame(rest, stdout, stderr)
	case "cherry-pick":
		return cmdCherryPick(rest, stdout, stderr)
	case "amend":
		return cmdAmend(rest, stdout, stderr)
	case "stash":
		return cmdStash(rest, stdout, stderr)
	case "rebase":
		return cmdRebase(rest, stdout, stderr)
	case "list":
		return cmdList(rest, stdout, stderr)
	case "merge":
		return cmdMerge(rest, stdout, stderr)
	case "merge3":
		return cmdMerge3(rest, stdout, stderr)
	case "materialize":
		return cmdMaterialize(rest, stdout, stderr)
	case "trace":
		return cmdTrace(rest, stdout, stderr)
	case "cone":
		return cmdCone(rest, stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", cmd)
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `got — command-line shell over the repository library

usage:
  got init
  got add-vertex <name> --type <VertexType> [--attr k=v ...]
  got add-edge <name> --type <EdgeType> --from <v> --to <v>
  got bind <ref> <vertex>
  got resolve <ref>
  got branch <name> [--from <parent>] [--tip <vertex>] [--desc <text>]
  got branch -d <name> | got branch -m <old> <new>
  got branches
  got branch-log <name>
  got status
  got checkout <branch>   (alias: switch; --force to discard uncommitted changes)
  got commit -m <message> [--branch <name>] [--actor <name>]
  got log [<branch>] [--touching <name>]   (--touching filters to commits that changed a node)
  got blame <name>             (which commit introduced / last changed a node)
  got cherry-pick <commit-ish> (apply a commit's change onto the current branch)
  got amend [-m <message>]     (replace the last commit with the working state)
  got stash [push|pop|list]    (save/restore uncommitted working changes)
  got rebase <onto>            (replay the current branch's commits onto another)
  got show [<commit-ish>]      (commit metadata + diff vs parent; default HEAD)
  got tag <name> [<commit-ish>] | got tags
  got revert <commit-ish>      (new commit undoing the target)
  got reset [--hard] <commit-ish>   (move current branch tip; --hard resets working graph)
  got restore [<commit-ish>]        (reset working graph to a commit; default HEAD)
  got diff [<branch>]          (last commit vs its parent; default HEAD)
  got diff <branchA> <branchB> (two branch heads)
  got list vertices|edges
  got merge <branch>                     (semantic merge into the current branch)
  got merge-base <branchA> <branchB>     (nearest common commit)
  got merge3 <ancestor> <left> <right>   (low-level three-way frontier merge)
  got materialize                        (manifest bundle of the whole graph)
  got trace <from> <to>
  got cone <name>

state is a repository directory under $GOT_DIR (default .got):
graph.json (the graph) + namespace.json (the durable namespace).
`)
}

// newService wires the full engine stack into a repo.DefaultService, mirroring
// the wiring used elsewhere in the codebase.
func newService() *repo.DefaultService {
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, nil)
	return repo.NewService(
		composition.NewEngine(gov, ver),
		gov,
		projection.NewEngine(),
		realization.NewEngine(),
		revision.NewEngine(),
		ver,
	)
}

// --- commands ---

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if repoInitialized() {
		fmt.Fprintf(stdout, "repository already exists at %s\n", stateDir())
		return 0
	}
	// Load an empty State (creates a namespace FileStore) and save the empty
	// graph so graph.json exists and marks the repo initialized.
	state, err := repo.LoadState(stateDir(), schema())
	if err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 1
	}
	if err := saveState(state); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 1
	}
	if err := setHEAD("main"); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "initialized empty repository at %s (on branch main)\n", stateDir())
	return 0
}

func cmdStatus(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "status: takes no arguments")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "status: %v\n", err)
		return 1
	}
	branch := currentBranch()
	fmt.Fprintf(stdout, "On branch %s\n", branch)

	headSnap, hasCommit := headSnapshot(state, log, branch)
	if !hasCommit {
		fmt.Fprintln(stdout, "No commits yet.")
	}
	delta := graph.Diff(contentOnly(headSnap), contentOnly(graph.EncodeSnapshot(state.Graph())))
	if delta.Empty() {
		fmt.Fprintln(stdout, "nothing to commit, working graph clean")
		return 0
	}
	fmt.Fprintln(stdout, "Uncommitted changes:")
	printDelta(stdout, delta)
	return 0
}

func cmdCheckout(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("checkout", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var force, create bool
	fs.BoolVar(&force, "force", false, "discard uncommitted changes")
	fs.BoolVar(&create, "b", false, "create the branch at the current commit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "checkout: expected [-b] <branch>")
		return 2
	}
	target := fs.Arg(0)

	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "checkout: %v\n", err)
		return 1
	}
	ctx := context.Background()
	cur := currentBranch()

	if create {
		if branchExists(state, target) {
			fmt.Fprintf(stderr, "checkout: branch %q already exists\n", target)
			return 1
		}
		// New branch starts at the current branch's commit; working graph stays.
		if id, ok := state.Namespace().ResolveRef(ctx, commitRefName(cur)); ok {
			if err := state.Namespace().BindRef(ctx, commitRefName(target), id); err != nil {
				fmt.Fprintf(stderr, "checkout: %v\n", err)
				return 1
			}
		}
		if err := setHEAD(target); err != nil {
			fmt.Fprintf(stderr, "checkout: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "created and switched to branch %q\n", target)
		return 0
	}

	if !branchExists(state, target) {
		fmt.Fprintf(stderr, "checkout: no such branch %q (use -b to create)\n", target)
		return 1
	}

	// Safety: refuse to switch away from uncommitted content changes.
	curHead, _ := headSnapshot(state, log, cur)
	if !force && !graph.Diff(contentOnly(curHead), contentOnly(graph.EncodeSnapshot(state.Graph()))).Empty() {
		fmt.Fprintf(stderr, "checkout: uncommitted changes on %q; commit them or use --force\n", cur)
		return 1
	}

	// Update the working graph to the target branch's committed state.
	targetSnap, hasCommit := headSnapshot(state, log, target)
	var g graph.Graph
	if hasCommit {
		g, err = targetSnap.Build(schema())
		if err != nil {
			fmt.Fprintf(stderr, "checkout: %v\n", err)
			return 1
		}
	} else {
		g = graph.NewGraph(schema())
	}
	if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
		fmt.Fprintf(stderr, "checkout: %v\n", err)
		return 1
	}
	if err := setHEAD(target); err != nil {
		fmt.Fprintf(stderr, "checkout: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "switched to branch %q\n", target)
	return 0
}

// branchExists reports whether a branch is known — it has a commit pointer or
// a first-class BranchSelector vertex, or it is the current HEAD branch.
func branchExists(state repo.State, branch string) bool {
	if _, ok := state.Namespace().ResolveRef(context.Background(), commitRefName(branch)); ok {
		return true
	}
	if _, ok := state.Graph().Vertex(repo.BranchVID(branch)); ok {
		return true
	}
	return branch == currentBranch()
}

// contentOnly drops BranchSelector vertices (and edges touching them) from a
// snapshot, so first-class branch metadata does not count as versioned content
// in status/diff.
func contentOnly(s graph.Snapshot) graph.Snapshot {
	branchIDs := make(map[string]bool)
	var out graph.Snapshot
	for _, v := range s.Vertices {
		if v.Type == string(ontology.BranchSelector) {
			branchIDs[v.ID] = true
			continue
		}
		out.Vertices = append(out.Vertices, v)
	}
	for _, e := range s.Edges {
		if branchIDs[e.From] || branchIDs[e.To] {
			continue
		}
		out.Edges = append(out.Edges, e)
	}
	out.Hyperedges = s.Hyperedges
	return out
}

func cmdAddVertex(args []string, stdout, stderr io.Writer) int {
	name, rest, ok := splitName(args)
	if !ok {
		fmt.Fprintln(stderr, "add-vertex: expected '<name> --type <VertexType> [--attr k=v ...]'")
		return 2
	}
	fs := flag.NewFlagSet("add-vertex", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var typ string
	var attrs multiFlag
	fs.StringVar(&typ, "type", "", "vertex type (e.g. Artifact)")
	fs.Var(&attrs, "attr", "attribute key=value (repeatable)")
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "add-vertex: unexpected extra arguments")
		return 2
	}
	if typ == "" {
		fmt.Fprintln(stderr, "add-vertex: --type is required")
		return 2
	}
	if !schema().KnownVertexType(ontology.VertexType(typ)) {
		fmt.Fprintf(stderr, "add-vertex: unknown vertex type %q\n", typ)
		return 1
	}
	parsed, err := attrs.parse()
	if err != nil {
		fmt.Fprintf(stderr, "add-vertex: %v\n", err)
		return 2
	}

	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, ok := vertexNamed(state.Graph(), name); ok {
		fmt.Fprintf(stderr, "add-vertex: vertex %q already exists\n", name)
		return 1
	}

	newState, err := newService().Ingest(context.Background(), state, repo.VertexPayload{
		Vertices: []graph.Vertex{{ID: vid(name), Type: ontology.VertexType(typ), Attrs: withName(name, parsed)}},
	})
	if err != nil {
		fmt.Fprintf(stderr, "add-vertex: %v\n", err)
		return 1
	}
	if err := saveState(newState); err != nil {
		fmt.Fprintf(stderr, "add-vertex: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "added vertex %q (%s)\n", name, typ)
	return 0
}

func cmdAddEdge(args []string, stdout, stderr io.Writer) int {
	name, rest, ok := splitName(args)
	if !ok {
		fmt.Fprintln(stderr, "add-edge: expected '<name> --type <EdgeType> --from <v> --to <v>'")
		return 2
	}
	fs := flag.NewFlagSet("add-edge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var typ, from, to string
	fs.StringVar(&typ, "type", "", "edge type (e.g. derived_from)")
	fs.StringVar(&from, "from", "", "source vertex name")
	fs.StringVar(&to, "to", "", "destination vertex name")
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "add-edge: unexpected extra arguments")
		return 2
	}
	if typ == "" || from == "" || to == "" {
		fmt.Fprintln(stderr, "add-edge: --type, --from and --to are required")
		return 2
	}

	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, ok := vertexNamed(state.Graph(), from); !ok {
		fmt.Fprintf(stderr, "add-edge: unknown --from vertex %q\n", from)
		return 1
	}
	if _, ok := vertexNamed(state.Graph(), to); !ok {
		fmt.Fprintf(stderr, "add-edge: unknown --to vertex %q\n", to)
		return 1
	}

	newState, err := newService().Ingest(context.Background(), state, repo.EdgePayload{
		Edges: []graph.Edge{{ID: eid(name), Type: ontology.EdgeType(typ), From: vid(from), To: vid(to), Attrs: withName(name, nil)}},
	})
	if err != nil {
		fmt.Fprintf(stderr, "add-edge: %v\n", err)
		return 1
	}
	if err := saveState(newState); err != nil {
		fmt.Fprintf(stderr, "add-edge: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "added edge %q (%s -%s-> %s)\n", name, from, typ, to)
	return 0
}

func cmdBind(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "bind: expected <ref> <vertex>")
		return 2
	}
	ref, target := args[0], args[1]

	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// Branch checks the vertex exists and binds the ref into the durable
	// namespace FileStore (persisted immediately; graph is unchanged).
	if _, err := newService().Branch(context.Background(), state, refName(ref), vid(target)); err != nil {
		fmt.Fprintf(stderr, "bind: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "bound %q -> %q\n", ref, target)
	return 0
}

func cmdResolve(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "resolve: expected <ref>")
		return 2
	}
	ref := args[0]
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	id, ok := state.Namespace().ResolveRef(context.Background(), refName(ref))
	if !ok {
		fmt.Fprintf(stderr, "resolve: ref %q is unbound\n", ref)
		return 1
	}
	name := nameIndex(state.Graph())[id]
	fmt.Fprintf(stdout, "%s -> %s (%s)\n", ref, name, shortID(id[:]))
	return 0
}

func cmdBranch(args []string, stdout, stderr io.Writer) int {
	// Flag-first forms: `branch -d <name>` (delete), `branch -m <old> <new>`
	// (rename). Otherwise the name-first create form.
	if len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "-d", "--delete":
			return cmdBranchDelete(args[1:], stdout, stderr)
		case "-m", "--rename":
			return cmdBranchRename(args[1:], stdout, stderr)
		}
	}
	name, rest, ok := splitName(args)
	if !ok {
		fmt.Fprintln(stderr, "branch: expected '<name> [--from <parent>] [--tip <vertex>] [--desc <text>]', '-d <name>', or '-m <old> <new>'")
		return 2
	}
	fs := flag.NewFlagSet("branch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var from, tip, desc string
	fs.StringVar(&from, "from", "", "parent branch name")
	fs.StringVar(&tip, "tip", "", "vertex the branch initially points at")
	fs.StringVar(&desc, "desc", "", "branch description")
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "branch: unexpected extra arguments")
		return 2
	}

	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var tipID identity.VertexID
	if tip != "" {
		if _, ok := vertexNamed(state.Graph(), tip); !ok {
			fmt.Fprintf(stderr, "branch: unknown --tip vertex %q\n", tip)
			return 1
		}
		tipID = vid(tip)
	}
	meta := map[string]string{}
	if desc != "" {
		meta["desc"] = desc
	}

	newState, b, err := newService().CreateBranch(context.Background(), state, name, from, tipID, meta)
	if err != nil {
		fmt.Fprintf(stderr, "branch: %v\n", err)
		return 1
	}
	if err := saveState(newState); err != nil {
		fmt.Fprintf(stderr, "branch: %v\n", err)
		return 1
	}
	if b.Parent != "" {
		fmt.Fprintf(stdout, "created branch %q (forked from %q)\n", b.Name, b.Parent)
	} else {
		fmt.Fprintf(stdout, "created branch %q\n", b.Name)
	}
	return 0
}

func cmdBranchDelete(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "branch -d: expected <name>")
		return 2
	}
	name := args[0]
	if name == currentBranch() {
		fmt.Fprintf(stderr, "branch -d: cannot delete the current branch %q\n", name)
		return 1
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !branchExists(state, name) {
		fmt.Fprintf(stderr, "branch -d: no such branch %q\n", name)
		return 1
	}
	ctx := context.Background()
	if err := state.Namespace().DeleteRef(ctx, commitRefName(name)); err != nil {
		fmt.Fprintf(stderr, "branch -d: %v\n", err)
		return 1
	}
	// Remove the first-class branch vertex if present (invisible to content
	// diff/status, which exclude BranchSelector vertices).
	if _, ok := state.Graph().Vertex(repo.BranchVID(name)); ok {
		g, err := removeVertexAndEdges(state.Graph(), repo.BranchVID(name))
		if err != nil {
			fmt.Fprintf(stderr, "branch -d: %v\n", err)
			return 1
		}
		if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
			fmt.Fprintf(stderr, "branch -d: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "deleted branch %q\n", name)
	return 0
}

func cmdBranchRename(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "branch -m: expected <old> <new>")
		return 2
	}
	old, newName := args[0], args[1]
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !branchExists(state, old) {
		fmt.Fprintf(stderr, "branch -m: no such branch %q\n", old)
		return 1
	}
	if branchExists(state, newName) {
		fmt.Fprintf(stderr, "branch -m: branch %q already exists\n", newName)
		return 1
	}
	ctx := context.Background()
	if id, ok := state.Namespace().ResolveRef(ctx, commitRefName(old)); ok {
		if err := state.Namespace().BindRef(ctx, commitRefName(newName), id); err != nil {
			fmt.Fprintf(stderr, "branch -m: %v\n", err)
			return 1
		}
	}
	if err := state.Namespace().DeleteRef(ctx, commitRefName(old)); err != nil {
		fmt.Fprintf(stderr, "branch -m: %v\n", err)
		return 1
	}
	if currentBranch() == old {
		if err := setHEAD(newName); err != nil {
			fmt.Fprintf(stderr, "branch -m: %v\n", err)
			return 1
		}
	}
	// The first-class branch vertex (if any) is keyed by name; drop the old
	// one so `branches` no longer lists the old name. Re-create it under the
	// new name with `got branch <new>` if you want its metadata/lineage back.
	if _, ok := state.Graph().Vertex(repo.BranchVID(old)); ok {
		g, err := removeVertexAndEdges(state.Graph(), repo.BranchVID(old))
		if err != nil {
			fmt.Fprintf(stderr, "branch -m: %v\n", err)
			return 1
		}
		if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
			fmt.Fprintf(stderr, "branch -m: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "renamed branch %q -> %q\n", old, newName)
	return 0
}

// removeVertexAndEdges rebuilds g without the given vertex and any edge
// touching it, returning the validated result.
func removeVertexAndEdges(g graph.Graph, id identity.VertexID) (graph.Graph, error) {
	b := graph.NewBuilder(schema())
	for _, v := range g.Vertices() {
		if v.ID == id {
			continue
		}
		b.AddVertex(v)
	}
	for _, e := range g.Edges() {
		if e.From == id || e.To == id {
			continue
		}
		if err := b.AddEdge(e); err != nil {
			return nil, err
		}
	}
	out := b.Build()
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

func cmdBranches(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "branches: takes no arguments")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	branches, err := newService().Branches(context.Background(), state)
	if err != nil {
		fmt.Fprintf(stderr, "branches: %v\n", err)
		return 1
	}
	sort.Slice(branches, func(i, j int) bool { return branches[i].Name < branches[j].Name })
	for _, b := range branches {
		line := b.Name
		if b.Parent != "" {
			line += "\t(from " + b.Parent + ")"
		}
		if d := b.Attrs["desc"]; d != "" {
			line += "\t" + d
		}
		fmt.Fprintln(stdout, line)
	}
	return 0
}

func cmdBranchLog(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "branch-log: expected <name>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	lineage, err := newService().BranchLineage(context.Background(), state, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "branch-log: %v\n", err)
		return 1
	}
	names := make([]string, len(lineage))
	for i, b := range lineage {
		names[i] = b.Name
	}
	// child ← parent ← … ← root
	fmt.Fprintln(stdout, strings.Join(names, " <- "))
	return 0
}

func cmdCommit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var message, branch, actor string
	fs.StringVar(&message, "m", "", "commit message (required)")
	fs.StringVar(&branch, "branch", "", "branch to commit on (default: current)")
	fs.StringVar(&actor, "actor", "", "commit author")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if message == "" {
		fmt.Fprintln(stderr, "commit: -m <message> is required")
		return 2
	}
	if branch == "" {
		branch = currentBranch()
	}

	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "commit: %v\n", err)
		return 1
	}
	ctx := context.Background()

	var parents []history.CommitID
	if id, ok := state.Namespace().ResolveRef(ctx, commitRefName(branch)); ok {
		parents = []history.CommitID{commitFromVID(id)}
	}

	c, err := newService().Commit(ctx, state, log, message, actor, parents)
	if err != nil {
		fmt.Fprintf(stderr, "commit: %v\n", err)
		return 1
	}
	if err := saveHistory(log); err != nil {
		fmt.Fprintf(stderr, "commit: %v\n", err)
		return 1
	}
	// Advance the branch's commit pointer (persisted by the FileStore).
	if err := state.Namespace().BindRef(ctx, commitRefName(branch), vidFromCommit(c.ID)); err != nil {
		fmt.Fprintf(stderr, "commit: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "committed %s to %s: %s\n", shortID(c.ID[:]), branch, message)
	if n := len(c.Produced); n > 0 {
		fmt.Fprintf(stdout, "  +%d vertex(es)", n)
		if m := len(c.Consumed); m > 0 {
			fmt.Fprintf(stdout, " -%d", m)
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func cmdLog(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var touching string
	fs.StringVar(&touching, "touching", "", "only commits that added/changed/removed this node")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	branch := currentBranch()
	if fs.NArg() == 1 {
		branch = fs.Arg(0)
	} else if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "log: expected an optional <branch>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "log: %v\n", err)
		return 1
	}
	head, ok := state.Namespace().ResolveRef(context.Background(), commitRefName(branch))
	if !ok {
		fmt.Fprintf(stdout, "no commits on branch %q\n", branch)
		return 0
	}
	commits, err := log.Ancestors(commitFromVID(head))
	if err != nil {
		fmt.Fprintf(stderr, "log: %v\n", err)
		return 1
	}
	target := vid(touching)
	for _, c := range commits {
		if touching != "" && !commitTouches(log, c, target) {
			continue
		}
		author := c.Actor
		if author == "" {
			author = "-"
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", shortID(c.ID[:]), author, c.Message)
	}
	return 0
}

// commitTouches reports whether commit c added, removed, or changed the given
// vertex relative to its first parent.
func commitTouches(log *history.Log, c history.Commit, target identity.VertexID) bool {
	var parentSnap graph.Snapshot
	if len(c.Parents) > 0 {
		if p, ok := log.Get(c.Parents[0]); ok {
			parentSnap = p.Snapshot
		}
	}
	d := graph.Diff(parentSnap, c.Snapshot)
	th := hexOfVID(target)
	for _, v := range d.AddedVertices {
		if v.ID == th {
			return true
		}
	}
	for _, v := range d.RemovedVertices {
		if v.ID == th {
			return true
		}
	}
	for _, ch := range d.ChangedVertices {
		if ch.New.ID == th {
			return true
		}
	}
	return false
}

func hexOfVID(id identity.VertexID) string { return hex.EncodeToString(id[:]) }

// cmdBlame reports which commit introduced a node and which last changed it —
// per-node provenance, rather than git's per-line heuristic blame.
func cmdBlame(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "blame: expected <name>")
		return 2
	}
	name := args[0]
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "blame: %v\n", err)
		return 1
	}
	head, ok := state.Namespace().ResolveRef(context.Background(), commitRefName(currentBranch()))
	if !ok {
		fmt.Fprintln(stderr, "blame: current branch has no commits")
		return 1
	}
	anc, err := log.Ancestors(commitFromVID(head))
	if err != nil {
		fmt.Fprintf(stderr, "blame: %v\n", err)
		return 1
	}
	// Ancestors is newest-first; walk chronologically (oldest-first).
	target := hexOfVID(vid(name))
	var introduced, lastChanged *history.Commit
	var prev *graph.VertexSnapshot
	for i := len(anc) - 1; i >= 0; i-- {
		c := anc[i]
		cur, present := findVertexInSnap(c.Snapshot, target)
		if !present {
			continue
		}
		cc := c
		if introduced == nil {
			introduced = &cc
			lastChanged = &cc
			pv := cur
			prev = &pv
			continue
		}
		if prev == nil || !snapVertexEqual(*prev, cur) {
			lastChanged = &cc
			pv := cur
			prev = &pv
		}
	}
	if introduced == nil {
		fmt.Fprintf(stderr, "blame: %q not present in this branch's history\n", name)
		return 1
	}
	fmt.Fprintf(stdout, "node %s\n", name)
	fmt.Fprintf(stdout, "  introduced by  %s  %s\t%s\n", shortID(introduced.ID[:]), blameAuthor(*introduced), introduced.Message)
	fmt.Fprintf(stdout, "  last changed   %s  %s\t%s\n", shortID(lastChanged.ID[:]), blameAuthor(*lastChanged), lastChanged.Message)
	return 0
}

func blameAuthor(c history.Commit) string {
	if c.Actor == "" {
		return "-"
	}
	return c.Actor
}

func findVertexInSnap(s graph.Snapshot, hexID string) (graph.VertexSnapshot, bool) {
	for _, v := range s.Vertices {
		if v.ID == hexID {
			return v, true
		}
	}
	return graph.VertexSnapshot{}, false
}

func snapVertexEqual(a, b graph.VertexSnapshot) bool {
	if a.Type != b.Type || a.Time != b.Time || a.Trust != b.Trust {
		return false
	}
	if len(a.Attrs) != len(b.Attrs) {
		return false
	}
	for k, av := range a.Attrs {
		if bv, ok := b.Attrs[k]; !ok || bv != av {
			return false
		}
	}
	return true
}

func cmdDiff(args []string, stdout, stderr io.Writer) int {
	if len(args) > 2 {
		fmt.Fprintln(stderr, "diff: expected [<branch>] or <branchA> <branchB>")
		return 2
	}
	if len(args) == 0 {
		args = []string{currentBranch()}
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "diff: %v\n", err)
		return 1
	}
	ctx := context.Background()

	head := func(branch string) (history.Commit, bool) {
		id, ok := state.Namespace().ResolveRef(ctx, commitRefName(branch))
		if !ok {
			return history.Commit{}, false
		}
		return log.Get(commitFromVID(id))
	}

	var oldSnap, newSnap graph.Snapshot
	if len(args) == 1 {
		h, ok := head(args[0])
		if !ok {
			fmt.Fprintf(stderr, "diff: no commits on branch %q\n", args[0])
			return 1
		}
		newSnap = h.Snapshot
		if len(h.Parents) > 0 {
			if p, ok := log.Get(h.Parents[0]); ok {
				oldSnap = p.Snapshot
			}
		}
	} else {
		a, ok := head(args[0])
		b, ok2 := head(args[1])
		if !ok || !ok2 {
			fmt.Fprintln(stderr, "diff: both branches must have commits")
			return 1
		}
		oldSnap, newSnap = a.Snapshot, b.Snapshot
	}

	printDelta(stdout, graph.Diff(contentOnly(oldSnap), contentOnly(newSnap)))
	return 0
}

// snapName recovers the human name recorded on a snapshot element's attrs.
func snapName(attrs graph.AttrMap, fallbackHex string) string {
	if attrs != nil {
		if n, ok := attrs[nameAttr].(string); ok {
			return n
		}
	}
	if len(fallbackHex) > 12 {
		return fallbackHex[:12]
	}
	return fallbackHex
}

func printDelta(w io.Writer, d graph.Delta) {
	if d.Empty() {
		fmt.Fprintln(w, "no changes")
		return
	}
	for _, v := range d.AddedVertices {
		fmt.Fprintf(w, "+ vertex %s (%s)\n", snapName(v.Attrs, v.ID), v.Type)
	}
	for _, v := range d.RemovedVertices {
		fmt.Fprintf(w, "- vertex %s (%s)\n", snapName(v.Attrs, v.ID), v.Type)
	}
	for _, c := range d.ChangedVertices {
		fmt.Fprintf(w, "~ vertex %s: %s\n", snapName(c.New.Attrs, c.New.ID), describeVertexChange(c))
	}
	for _, e := range d.AddedEdges {
		fmt.Fprintf(w, "+ edge %s (%s)\n", snapName(e.Attrs, e.ID), e.Type)
	}
	for _, e := range d.RemovedEdges {
		fmt.Fprintf(w, "- edge %s (%s)\n", snapName(e.Attrs, e.ID), e.Type)
	}
	for _, c := range d.ChangedEdges {
		fmt.Fprintf(w, "~ edge %s (%s -> %s)\n", snapName(c.New.Attrs, c.New.ID), c.Old.Type, c.New.Type)
	}
}

func describeVertexChange(c graph.VertexChange) string {
	if c.Old.Type != c.New.Type {
		return fmt.Sprintf("type %s -> %s", c.Old.Type, c.New.Type)
	}
	// Report the first attr that differs (ignoring the reserved name attr).
	for k, nv := range c.New.Attrs {
		if k == nameAttr {
			continue
		}
		if ov, ok := c.Old.Attrs[k]; !ok || ov != nv {
			return fmt.Sprintf("attr %q: %v -> %v", k, c.Old.Attrs[k], nv)
		}
	}
	for k := range c.Old.Attrs {
		if k == nameAttr {
			continue
		}
		if _, ok := c.New.Attrs[k]; !ok {
			return fmt.Sprintf("attr %q removed", k)
		}
	}
	return "content changed"
}

func cmdList(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "vertices" && args[0] != "edges") {
		fmt.Fprintln(stderr, "list: expected 'vertices' or 'edges'")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	g := state.Graph()
	if args[0] == "vertices" {
		verts := make([]graph.Vertex, 0, len(g.Vertices()))
		for _, v := range g.Vertices() {
			if v.Type == ontology.BranchSelector {
				continue // branches are shown by `got branches`
			}
			verts = append(verts, v)
		}
		sort.Slice(verts, func(i, j int) bool { return nameOf(verts[i]) < nameOf(verts[j]) })
		for _, v := range verts {
			fmt.Fprintf(stdout, "%s\t%s\n", nameOf(v), v.Type)
		}
		return 0
	}
	idx := nameIndex(g)
	edges := append([]graph.Edge(nil), g.Edges()...)
	sort.Slice(edges, func(i, j int) bool { return edgeNameOf(edges[i]) < edgeNameOf(edges[j]) })
	for _, e := range edges {
		fmt.Fprintf(stdout, "%s\t%s -%s-> %s\n", edgeNameOf(e), idx[e.From], e.Type, idx[e.To])
	}
	return 0
}

// cmdMerge merges another branch into the current branch (HEAD) with a
// semantic three-way merge: fast-forward when possible, else a merge commit
// with two parents, else typed conflicts.
func cmdMerge(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("merge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var actor string
	fs.StringVar(&actor, "actor", "", "merge author")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "merge: expected <branch> (to merge into the current branch)")
		return 2
	}
	other := fs.Arg(0)
	cur := currentBranch()
	if other == cur {
		fmt.Fprintln(stderr, "merge: cannot merge a branch into itself")
		return 1
	}

	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	ctx := context.Background()

	curID, ok := state.Namespace().ResolveRef(ctx, commitRefName(cur))
	if !ok {
		fmt.Fprintf(stderr, "merge: current branch %q has no commits\n", cur)
		return 1
	}
	otherID, ok := state.Namespace().ResolveRef(ctx, commitRefName(other))
	if !ok {
		fmt.Fprintf(stderr, "merge: branch %q has no commits\n", other)
		return 1
	}
	curC, otherC := commitFromVID(curID), commitFromVID(otherID)
	if curC == otherC {
		fmt.Fprintln(stdout, "already up to date")
		return 0
	}

	base, hasBase := log.MergeBase(curC, otherC)
	if hasBase && base == otherC {
		fmt.Fprintln(stdout, "already up to date")
		return 0
	}
	if hasBase && base == curC {
		// Fast-forward: current has no unique commits.
		otherCommit, _ := log.Get(otherC)
		g, err := otherCommit.Snapshot.Build(schema())
		if err != nil {
			fmt.Fprintf(stderr, "merge: %v\n", err)
			return 1
		}
		if err := state.Namespace().BindRef(ctx, commitRefName(cur), otherID); err != nil {
			fmt.Fprintf(stderr, "merge: %v\n", err)
			return 1
		}
		if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
			fmt.Fprintf(stderr, "merge: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "fast-forward: %s now at %s\n", cur, shortID(otherID[:]))
		return 0
	}

	var baseSnap graph.Snapshot
	if hasBase {
		bc, _ := log.Get(base)
		baseSnap = bc.Snapshot
	}
	curCommit, _ := log.Get(curC)
	otherCommit, _ := log.Get(otherC)

	mergedGraph, mr, err := newService().MergeStates(ctx, schema(), baseSnap, curCommit.Snapshot, otherCommit.Snapshot)
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	if len(mr.Conflicts) > 0 {
		fmt.Fprintf(stdout, "merge aborted: %d conflict(s)\n", len(mr.Conflicts))
		for _, c := range mr.Conflicts {
			if d, ok := c.(interface{ Detail() string }); ok {
				fmt.Fprintf(stdout, "  %s: %s\n", c.Kind(), d.Detail())
			} else {
				fmt.Fprintf(stdout, "  %s\n", c.Kind())
			}
		}
		return 1
	}

	mergedState := repo.NewState(mergedGraph, state.Namespace())
	c, err := newService().Commit(ctx, mergedState, log, "merge "+other+" into "+cur, actor, []history.CommitID{curC, otherC})
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	if err := saveHistory(log); err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	if err := state.Namespace().BindRef(ctx, commitRefName(cur), vidFromCommit(c.ID)); err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	if err := saveState(mergedState); err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "merged %q into %q: %s\n", other, cur, shortID(c.ID[:]))
	return 0
}

func cmdShow(args []string, stdout, stderr io.Writer) int {
	ref := currentBranch()
	if len(args) == 1 {
		ref = args[0]
	} else if len(args) > 1 {
		fmt.Fprintln(stderr, "show: expected an optional <commit-ish>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "show: %v\n", err)
		return 1
	}
	cid, ok := resolveCommit(state, log, ref)
	if !ok {
		fmt.Fprintf(stderr, "show: unknown commit-ish %q\n", ref)
		return 1
	}
	c, _ := log.Get(cid)
	fmt.Fprintf(stdout, "commit %s\n", shortID(c.ID[:]))
	if len(c.Parents) > 1 {
		ps := make([]string, len(c.Parents))
		for i, p := range c.Parents {
			ps[i] = shortID(p[:])
		}
		fmt.Fprintf(stdout, "Merge: %s\n", strings.Join(ps, " "))
	}
	if c.Actor != "" {
		fmt.Fprintf(stdout, "Author: %s\n", c.Actor)
	}
	fmt.Fprintf(stdout, "\n    %s\n\n", c.Message)

	var parentSnap graph.Snapshot
	if len(c.Parents) > 0 {
		if p, ok := log.Get(c.Parents[0]); ok {
			parentSnap = p.Snapshot
		}
	}
	printDelta(stdout, graph.Diff(contentOnly(parentSnap), contentOnly(c.Snapshot)))
	return 0
}

func cmdTag(args []string, stdout, stderr io.Writer) int {
	if !repoInitialized() {
		fmt.Fprintln(stderr, "no repository; run 'got init'")
		return 1
	}
	tags, err := loadTags()
	if err != nil {
		fmt.Fprintf(stderr, "tag: %v\n", err)
		return 1
	}
	if len(args) == 0 {
		names := make([]string, 0, len(tags))
		for n := range tags {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if b, err := hex.DecodeString(tags[n]); err == nil {
				fmt.Fprintf(stdout, "%s\t%s\n", n, shortID(b))
			}
		}
		return 0
	}
	name := args[0]
	ref := currentBranch()
	if len(args) == 2 {
		ref = args[1]
	} else if len(args) > 2 {
		fmt.Fprintln(stderr, "tag: expected <name> [<commit-ish>]")
		return 2
	}
	if _, exists := tags[name]; exists {
		fmt.Fprintf(stderr, "tag: %q already exists\n", name)
		return 1
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "tag: %v\n", err)
		return 1
	}
	cid, ok := resolveCommit(state, log, ref)
	if !ok {
		fmt.Fprintf(stderr, "tag: unknown commit-ish %q\n", ref)
		return 1
	}
	tags[name] = hex.EncodeToString(cid[:])
	if err := saveTags(tags); err != nil {
		fmt.Fprintf(stderr, "tag: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "tagged %q -> %s\n", name, shortID(cid[:]))
	return 0
}

func cmdRevert(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "revert: expected <commit-ish>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "revert: %v\n", err)
		return 1
	}
	ctx := context.Background()
	cid, ok := resolveCommit(state, log, args[0])
	if !ok {
		fmt.Fprintf(stderr, "revert: unknown commit-ish %q\n", args[0])
		return 1
	}
	c, _ := log.Get(cid)

	var parentSnap graph.Snapshot
	if len(c.Parents) > 0 {
		if p, ok := log.Get(c.Parents[0]); ok {
			parentSnap = p.Snapshot
		}
	}
	// Reverse delta (c -> parent) applied to the current working state undoes c.
	reverse := graph.Diff(c.Snapshot, parentSnap)
	revertedSnap := applySnapDelta(graph.EncodeSnapshot(state.Graph()), reverse)
	reverted, err := revertedSnap.Build(schema())
	if err != nil {
		fmt.Fprintf(stderr, "revert: %v\n", err)
		return 1
	}

	branch := currentBranch()
	var parents []history.CommitID
	if id, ok := state.Namespace().ResolveRef(ctx, commitRefName(branch)); ok {
		parents = []history.CommitID{commitFromVID(id)}
	}
	revState := repo.NewState(reverted, state.Namespace())
	newC, err := newService().Commit(ctx, revState, log, "Revert: "+c.Message, "", parents)
	if err != nil {
		fmt.Fprintf(stderr, "revert: %v\n", err)
		return 1
	}
	if err := saveHistory(log); err != nil {
		fmt.Fprintf(stderr, "revert: %v\n", err)
		return 1
	}
	if err := state.Namespace().BindRef(ctx, commitRefName(branch), vidFromCommit(newC.ID)); err != nil {
		fmt.Fprintf(stderr, "revert: %v\n", err)
		return 1
	}
	if err := saveState(revState); err != nil {
		fmt.Fprintf(stderr, "revert: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "reverted %s: new commit %s\n", shortID(cid[:]), shortID(newC.ID[:]))
	return 0
}

func cmdReset(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reset", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var hard bool
	fs.BoolVar(&hard, "hard", false, "also reset the working graph to the target commit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "reset: expected [--hard] <commit-ish>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "reset: %v\n", err)
		return 1
	}
	ctx := context.Background()
	cid, ok := resolveCommit(state, log, fs.Arg(0))
	if !ok {
		fmt.Fprintf(stderr, "reset: unknown commit-ish %q\n", fs.Arg(0))
		return 1
	}
	branch := currentBranch()
	if err := state.Namespace().BindRef(ctx, commitRefName(branch), vidFromCommit(cid)); err != nil {
		fmt.Fprintf(stderr, "reset: %v\n", err)
		return 1
	}
	if hard {
		c, _ := log.Get(cid)
		g, err := c.Snapshot.Build(schema())
		if err != nil {
			fmt.Fprintf(stderr, "reset: %v\n", err)
			return 1
		}
		if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
			fmt.Fprintf(stderr, "reset: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "reset %s to %s (hard)\n", branch, shortID(cid[:]))
		return 0
	}
	fmt.Fprintf(stdout, "reset %s to %s (working graph kept)\n", branch, shortID(cid[:]))
	return 0
}

func cmdRestore(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "restore: expected an optional <commit-ish>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "restore: %v\n", err)
		return 1
	}
	var cid history.CommitID
	if len(args) == 1 {
		var ok bool
		cid, ok = resolveCommit(state, log, args[0])
		if !ok {
			fmt.Fprintf(stderr, "restore: unknown commit-ish %q\n", args[0])
			return 1
		}
	} else {
		id, ok := state.Namespace().ResolveRef(context.Background(), commitRefName(currentBranch()))
		if !ok {
			fmt.Fprintln(stderr, "restore: current branch has no commits")
			return 1
		}
		cid = commitFromVID(id)
	}
	c, _ := log.Get(cid)
	g, err := c.Snapshot.Build(schema())
	if err != nil {
		fmt.Fprintf(stderr, "restore: %v\n", err)
		return 1
	}
	if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
		fmt.Fprintf(stderr, "restore: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "restored working graph to %s\n", shortID(cid[:]))
	return 0
}

func cmdStash(args []string, stdout, stderr io.Writer) int {
	sub := "push"
	if len(args) == 1 {
		sub = args[0]
	} else if len(args) > 1 {
		fmt.Fprintln(stderr, "stash: expected push|pop|list")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "stash: %v\n", err)
		return 1
	}
	stashes, err := loadStashes()
	if err != nil {
		fmt.Fprintf(stderr, "stash: %v\n", err)
		return 1
	}
	branch := currentBranch()

	switch sub {
	case "push":
		headSnap, _ := headSnapshot(state, log, branch)
		cur := graph.EncodeSnapshot(state.Graph())
		if graph.Diff(contentOnly(headSnap), contentOnly(cur)).Empty() {
			fmt.Fprintln(stdout, "nothing to stash, working graph clean")
			return 0
		}
		stashes = append(stashes, stashEntry{Branch: branch, Snapshot: cur})
		if err := saveStashes(stashes); err != nil {
			fmt.Fprintf(stderr, "stash: %v\n", err)
			return 1
		}
		// Reset the working graph to HEAD.
		g, err := headStateGraph(headSnap)
		if err != nil {
			fmt.Fprintf(stderr, "stash: %v\n", err)
			return 1
		}
		if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
			fmt.Fprintf(stderr, "stash: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "stashed working changes (%d stash(es))\n", len(stashes))
		return 0

	case "pop":
		if len(stashes) == 0 {
			fmt.Fprintln(stderr, "stash: no stashes")
			return 1
		}
		entry := stashes[len(stashes)-1]
		stashes = stashes[:len(stashes)-1]
		g, err := entry.Snapshot.Build(schema())
		if err != nil {
			fmt.Fprintf(stderr, "stash: %v\n", err)
			return 1
		}
		if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
			fmt.Fprintf(stderr, "stash: %v\n", err)
			return 1
		}
		if err := saveStashes(stashes); err != nil {
			fmt.Fprintf(stderr, "stash: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "restored stash (%d remaining)\n", len(stashes))
		return 0

	case "list":
		for i := len(stashes) - 1; i >= 0; i-- {
			fmt.Fprintf(stdout, "stash@{%d}: on %s\n", len(stashes)-1-i, stashes[i].Branch)
		}
		return 0

	default:
		fmt.Fprintln(stderr, "stash: expected push|pop|list")
		return 2
	}
}

// headStateGraph builds the graph for a branch's head snapshot, or an empty
// graph when the branch has no commit.
func headStateGraph(headSnap graph.Snapshot) (graph.Graph, error) {
	if len(headSnap.Vertices) == 0 && len(headSnap.Edges) == 0 && len(headSnap.Hyperedges) == 0 {
		return graph.NewGraph(schema()), nil
	}
	return headSnap.Build(schema())
}

func cmdRebase(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "rebase: expected <onto>")
		return 2
	}
	onto := args[0]
	cur := currentBranch()
	if onto == cur {
		fmt.Fprintln(stderr, "rebase: cannot rebase a branch onto itself")
		return 1
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "rebase: %v\n", err)
		return 1
	}
	ctx := context.Background()
	curTipID, ok := state.Namespace().ResolveRef(ctx, commitRefName(cur))
	if !ok {
		fmt.Fprintf(stderr, "rebase: current branch %q has no commits\n", cur)
		return 1
	}
	ontoTipID, ok := state.Namespace().ResolveRef(ctx, commitRefName(onto))
	if !ok {
		fmt.Fprintf(stderr, "rebase: branch %q has no commits\n", onto)
		return 1
	}
	curTip, ontoTip := commitFromVID(curTipID), commitFromVID(ontoTipID)
	if curTip == ontoTip {
		fmt.Fprintln(stdout, "already up to date")
		return 0
	}
	base, hasBase := log.MergeBase(curTip, ontoTip)
	if !hasBase {
		fmt.Fprintln(stderr, "rebase: no common ancestor; cannot rebase")
		return 1
	}
	if base == ontoTip {
		fmt.Fprintln(stdout, "already up to date")
		return 0
	}
	if base == curTip {
		// Current is an ancestor of onto: fast-forward.
		ontoCommit, _ := log.Get(ontoTip)
		g, err := ontoCommit.Snapshot.Build(schema())
		if err != nil {
			fmt.Fprintf(stderr, "rebase: %v\n", err)
			return 1
		}
		if err := state.Namespace().BindRef(ctx, commitRefName(cur), ontoTipID); err != nil {
			fmt.Fprintf(stderr, "rebase: %v\n", err)
			return 1
		}
		if err := saveState(repo.NewState(g, state.Namespace())); err != nil {
			fmt.Fprintf(stderr, "rebase: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "fast-forward: %s onto %s\n", cur, shortID(ontoTipID[:]))
		return 0
	}

	// Collect current's commits above the merge base, oldest-first.
	anc, err := log.Ancestors(curTip)
	if err != nil {
		fmt.Fprintf(stderr, "rebase: %v\n", err)
		return 1
	}
	var replay []history.Commit
	for _, c := range anc {
		if c.ID == base {
			break
		}
		replay = append(replay, c)
	}
	for i, j := 0, len(replay)-1; i < j; i, j = i+1, j-1 {
		replay[i], replay[j] = replay[j], replay[i]
	}

	svc := newService()
	running := ontoTip
	runningCommit, _ := log.Get(ontoTip)
	runningSnap := runningCommit.Snapshot
	var finalGraph graph.Graph
	for _, c := range replay {
		var parentSnap graph.Snapshot
		if len(c.Parents) > 0 {
			if p, ok := log.Get(c.Parents[0]); ok {
				parentSnap = p.Snapshot
			}
		}
		forward := graph.Diff(parentSnap, c.Snapshot)
		newSnap := applySnapDelta(runningSnap, forward)
		g, err := newSnap.Build(schema())
		if err != nil {
			fmt.Fprintf(stderr, "rebase: %v\n", err)
			return 1
		}
		newC, err := svc.Commit(ctx, repo.NewState(g, state.Namespace()), log, c.Message, c.Actor, []history.CommitID{running})
		if err != nil {
			fmt.Fprintf(stderr, "rebase: %v\n", err)
			return 1
		}
		running = newC.ID
		runningSnap = newC.Snapshot
		finalGraph = g
	}
	if err := saveHistory(log); err != nil {
		fmt.Fprintf(stderr, "rebase: %v\n", err)
		return 1
	}
	if err := state.Namespace().BindRef(ctx, commitRefName(cur), vidFromCommit(running)); err != nil {
		fmt.Fprintf(stderr, "rebase: %v\n", err)
		return 1
	}
	if err := saveState(repo.NewState(finalGraph, state.Namespace())); err != nil {
		fmt.Fprintf(stderr, "rebase: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "rebased %d commit(s) of %s onto %s\n", len(replay), cur, onto)
	return 0
}

func cmdCherryPick(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cherry-pick", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var actor string
	fs.StringVar(&actor, "actor", "", "author")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "cherry-pick: expected <commit-ish>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "cherry-pick: %v\n", err)
		return 1
	}
	ctx := context.Background()
	cid, ok := resolveCommit(state, log, fs.Arg(0))
	if !ok {
		fmt.Fprintf(stderr, "cherry-pick: unknown commit-ish %q\n", fs.Arg(0))
		return 1
	}
	c, _ := log.Get(cid)

	var parentSnap graph.Snapshot
	if len(c.Parents) > 0 {
		if p, ok := log.Get(c.Parents[0]); ok {
			parentSnap = p.Snapshot
		}
	}
	// Forward delta (parent -> c) applied to the current working state.
	forward := graph.Diff(parentSnap, c.Snapshot)
	appliedSnap := applySnapDelta(graph.EncodeSnapshot(state.Graph()), forward)
	applied, err := appliedSnap.Build(schema())
	if err != nil {
		fmt.Fprintf(stderr, "cherry-pick: %v\n", err)
		return 1
	}

	branch := currentBranch()
	var parents []history.CommitID
	if id, ok := state.Namespace().ResolveRef(ctx, commitRefName(branch)); ok {
		parents = []history.CommitID{commitFromVID(id)}
	}
	newState := repo.NewState(applied, state.Namespace())
	if actor == "" {
		actor = c.Actor
	}
	newC, err := newService().Commit(ctx, newState, log, "cherry-pick "+shortID(cid[:])+": "+c.Message, actor, parents)
	if err != nil {
		fmt.Fprintf(stderr, "cherry-pick: %v\n", err)
		return 1
	}
	if err := saveHistory(log); err != nil {
		fmt.Fprintf(stderr, "cherry-pick: %v\n", err)
		return 1
	}
	if err := state.Namespace().BindRef(ctx, commitRefName(branch), vidFromCommit(newC.ID)); err != nil {
		fmt.Fprintf(stderr, "cherry-pick: %v\n", err)
		return 1
	}
	if err := saveState(newState); err != nil {
		fmt.Fprintf(stderr, "cherry-pick: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "cherry-picked %s as %s\n", shortID(cid[:]), shortID(newC.ID[:]))
	return 0
}

func cmdAmend(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("amend", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var message string
	fs.StringVar(&message, "m", "", "new commit message (default: keep existing)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "amend: %v\n", err)
		return 1
	}
	ctx := context.Background()
	branch := currentBranch()
	headID, ok := state.Namespace().ResolveRef(ctx, commitRefName(branch))
	if !ok {
		fmt.Fprintln(stderr, "amend: no commit to amend")
		return 1
	}
	head, _ := log.Get(commitFromVID(headID))
	msg := head.Message
	if message != "" {
		msg = message
	}
	// The amended commit keeps the original's parents but takes the current
	// working state as its snapshot. The old commit becomes unreferenced.
	newC, err := newService().Commit(ctx, state, log, msg, head.Actor, head.Parents)
	if err != nil {
		fmt.Fprintf(stderr, "amend: %v\n", err)
		return 1
	}
	if err := saveHistory(log); err != nil {
		fmt.Fprintf(stderr, "amend: %v\n", err)
		return 1
	}
	if err := state.Namespace().BindRef(ctx, commitRefName(branch), vidFromCommit(newC.ID)); err != nil {
		fmt.Fprintf(stderr, "amend: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "amended: %s\n", shortID(newC.ID[:]))
	return 0
}

func cmdMergeBase(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "merge-base: expected <branchA> <branchB>")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "merge-base: %v\n", err)
		return 1
	}
	ctx := context.Background()
	a, okA := state.Namespace().ResolveRef(ctx, commitRefName(args[0]))
	b, okB := state.Namespace().ResolveRef(ctx, commitRefName(args[1]))
	if !okA || !okB {
		fmt.Fprintln(stderr, "merge-base: both branches must have commits")
		return 1
	}
	base, ok := log.MergeBase(commitFromVID(a), commitFromVID(b))
	if !ok {
		fmt.Fprintln(stdout, "no common ancestor")
		return 0
	}
	fmt.Fprintln(stdout, shortID(base[:]))
	return 0
}

func cmdMerge3(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		fmt.Fprintln(stderr, "merge3: expected <ancestor> <left> <right> (comma-separated vertex names)")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ancestor, err := frontierFromList(state.Graph(), args[0])
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	left, err := frontierFromList(state.Graph(), args[1])
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	right, err := frontierFromList(state.Graph(), args[2])
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	_, mr, err := newService().MergeThreeWay(context.Background(), state, ancestor, left, right, nil)
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	return printMergeResult(stdout, nameIndex(state.Graph()), mr)
}

func cmdMaterialize(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "materialize: takes no arguments")
		return 2
	}
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ids := state.Graph().VertexIDs()
	bundle, err := newService().Materialize(context.Background(), state, projection.InduceSpec{IDs: ids}, realization.ManifestTarget)
	if err != nil {
		fmt.Fprintf(stderr, "materialize: %v\n", err)
		return 1
	}
	paths := bundle.Paths()
	sort.Strings(paths)
	fmt.Fprintf(stdout, "bundle target=%s, %d path(s)\n", bundle.Target(), len(paths))
	for _, p := range paths {
		fmt.Fprintf(stdout, "  %s\n", p)
	}
	return 0
}

// frontierFromList parses a comma-separated list of vertex names into a
// frontier, erroring on any unknown name. The frontier is an EditedFrontier
// carrying IDs only; three-way content therefore comes from the host graph
// (presence-only reconciliation).
func frontierFromList(g graph.Graph, csv string) (projection.Frontier, error) {
	names := splitCSV(csv)
	if len(names) == 0 {
		return nil, fmt.Errorf("empty vertex list")
	}
	ids := make([]identity.VertexID, 0, len(names))
	for _, name := range names {
		if _, ok := vertexNamed(g, name); !ok {
			return nil, fmt.Errorf("unknown vertex %q", name)
		}
		ids = append(ids, vid(name))
	}
	return projection.NewEditedFrontier(ids), nil
}

// printMergeResult renders a MergeResult: the merged vertex names + witness on
// success, or the typed conflicts on failure. Returns the process exit code.
func printMergeResult(stdout io.Writer, idx map[identity.VertexID]string, mr composition.MergeResult) int {
	if mr.Frontier != nil {
		names := make([]string, 0)
		for _, id := range mr.Frontier.VertexIDs() {
			if n, ok := idx[id]; ok {
				names = append(names, n)
			} else {
				names = append(names, shortID(id[:]))
			}
		}
		sort.Strings(names)
		fmt.Fprintf(stdout, "merged %d vertex(es): %s\n", len(names), strings.Join(names, ", "))
		fmt.Fprintf(stdout, "witness: %s\n", shortID(mr.Witness.ID[:]))
		return 0
	}
	fmt.Fprintf(stdout, "%d conflict(s):\n", len(mr.Conflicts))
	for _, c := range mr.Conflicts {
		if d, ok := c.(interface{ Detail() string }); ok {
			fmt.Fprintf(stdout, "  %s: %s\n", c.Kind(), d.Detail())
		} else {
			fmt.Fprintf(stdout, "  %s\n", c.Kind())
		}
	}
	return 0
}

func cmdTrace(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "trace: expected <from> <to>")
		return 2
	}
	from, to := args[0], args[1]
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	g := state.Graph()
	if _, ok := vertexNamed(g, from); !ok {
		fmt.Fprintf(stderr, "trace: unknown vertex %q\n", from)
		return 1
	}
	if _, ok := vertexNamed(g, to); !ok {
		fmt.Fprintf(stderr, "trace: unknown vertex %q\n", to)
		return 1
	}
	eng := provenance.NewEngine(ontology.CausalEdges)
	ctx := context.Background()
	connected, err := eng.Causes(ctx, g, vid(from), vid(to))
	if err != nil {
		fmt.Fprintf(stderr, "trace: %v\n", err)
		return 1
	}
	if !connected {
		fmt.Fprintf(stdout, "%s and %s are not causally connected\n", from, to)
		return 0
	}
	traces, err := eng.TraceSet(ctx, g, vid(from), vid(to))
	if err != nil {
		fmt.Fprintf(stderr, "trace: %v\n", err)
		return 1
	}
	idx := nameIndex(g)
	fmt.Fprintf(stdout, "%s -> %s: %d path(s)\n", from, to, len(traces))
	for _, tr := range traces {
		names := make([]string, 0, len(tr.Vertices))
		for _, id := range tr.Vertices {
			names = append(names, idx[id])
		}
		fmt.Fprintf(stdout, "  %s\n", joinArrow(names))
	}
	return 0
}

func cmdCone(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "cone: expected <name>")
		return 2
	}
	name := args[0]
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	g := state.Graph()
	if _, ok := vertexNamed(g, name); !ok {
		fmt.Fprintf(stderr, "cone: unknown vertex %q\n", name)
		return 1
	}
	eng := provenance.NewEngine(ontology.CausalEdges)
	cone, err := eng.Cone(context.Background(), g, vid(name))
	if err != nil {
		fmt.Fprintf(stderr, "cone: %v\n", err)
		return 1
	}
	idx := nameIndex(g)
	names := make([]string, 0, len(cone))
	for _, id := range cone {
		names = append(names, idx[id])
	}
	sort.Strings(names)
	fmt.Fprintf(stdout, "cone(%s): %d vertex(es)\n", name, len(names))
	for _, n := range names {
		fmt.Fprintf(stdout, "  %s\n", n)
	}
	return 0
}
