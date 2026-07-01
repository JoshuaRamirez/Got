package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
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
  got list vertices|edges
  got merge <listA> <listB>              (comma-separated vertex names)
  got merge3 <ancestor> <left> <right>   (three-way, comma-separated lists)
  got materialize                        (manifest bundle of the whole graph)
  got trace <from> <to>
  got cone <name>

state is persisted as JSON under $GOT_DIR (default .got).
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
	if repoExists() {
		fmt.Fprintf(stdout, "repository already exists at %s\n", statePath())
		return 0
	}
	if err := saveSnapshot(&snapshot{Refs: map[string]string{}}); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "initialized empty repository at %s\n", statePath())
	return 0
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
	if !ontology.NewDefaultSchema().KnownVertexType(ontology.VertexType(typ)) {
		fmt.Fprintf(stderr, "add-vertex: unknown vertex type %q\n", typ)
		return 1
	}
	parsed, err := attrs.parse()
	if err != nil {
		fmt.Fprintf(stderr, "add-vertex: %v\n", err)
		return 2
	}

	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, ok := snap.vertexByName(name); ok {
		fmt.Fprintf(stderr, "add-vertex: vertex %q already exists\n", name)
		return 1
	}

	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "add-vertex: %v\n", err)
		return 1
	}
	svc := newService()
	_, err = svc.Ingest(context.Background(), state, repo.VertexPayload{
		Vertices: []graph.Vertex{{ID: vid(name), Type: ontology.VertexType(typ), Attrs: attrMap(parsed)}},
	})
	if err != nil {
		fmt.Fprintf(stderr, "add-vertex: %v\n", err)
		return 1
	}

	snap.Vertices = append(snap.Vertices, vertexRec{Name: name, Type: typ, Attrs: parsed})
	if err := saveSnapshot(snap); err != nil {
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

	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, ok := snap.vertexByName(from); !ok {
		fmt.Fprintf(stderr, "add-edge: unknown --from vertex %q\n", from)
		return 1
	}
	if _, ok := snap.vertexByName(to); !ok {
		fmt.Fprintf(stderr, "add-edge: unknown --to vertex %q\n", to)
		return 1
	}

	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "add-edge: %v\n", err)
		return 1
	}
	svc := newService()
	_, err = svc.Ingest(context.Background(), state, repo.EdgePayload{
		Edges: []graph.Edge{{ID: eid(name), Type: ontology.EdgeType(typ), From: vid(from), To: vid(to)}},
	})
	if err != nil {
		fmt.Fprintf(stderr, "add-edge: %v\n", err)
		return 1
	}

	snap.Edges = append(snap.Edges, edgeRec{Name: name, Type: typ, From: from, To: to})
	if err := saveSnapshot(snap); err != nil {
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

	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "bind: %v\n", err)
		return 1
	}
	svc := newService()
	if _, err := svc.Branch(context.Background(), state, refName(ref), vid(target)); err != nil {
		fmt.Fprintf(stderr, "bind: %v\n", err)
		return 1
	}

	snap.Refs[ref] = target
	if err := saveSnapshot(snap); err != nil {
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
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "resolve: %v\n", err)
		return 1
	}
	id, ok := state.Namespace().ResolveRef(context.Background(), refName(ref))
	if !ok {
		fmt.Fprintf(stderr, "resolve: ref %q is unbound\n", ref)
		return 1
	}
	name := snap.nameIndex()[id]
	fmt.Fprintf(stdout, "%s -> %s (%s)\n", ref, name, shortID(id[:]))
	return 0
}

func cmdList(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "vertices" && args[0] != "edges") {
		fmt.Fprintln(stderr, "list: expected 'vertices' or 'edges'")
		return 2
	}
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if args[0] == "vertices" {
		verts := append([]vertexRec(nil), snap.Vertices...)
		sort.Slice(verts, func(i, j int) bool { return verts[i].Name < verts[j].Name })
		for _, v := range verts {
			fmt.Fprintf(stdout, "%s\t%s\n", v.Name, v.Type)
		}
		return 0
	}
	edges := append([]edgeRec(nil), snap.Edges...)
	sort.Slice(edges, func(i, j int) bool { return edges[i].Name < edges[j].Name })
	for _, e := range edges {
		fmt.Fprintf(stdout, "%s\t%s -%s-> %s\n", e.Name, e.From, e.Type, e.To)
	}
	return 0
}

func cmdMerge(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "merge: expected <listA> <listB> (comma-separated vertex names)")
		return 2
	}
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	left, err := frontierFromList(snap, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	right, err := frontierFromList(snap, args[1])
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	_, mr, err := newService().Merge(context.Background(), state, left, right, nil)
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	return printMergeResult(stdout, snap, mr)
}

func cmdMerge3(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		fmt.Fprintln(stderr, "merge3: expected <ancestor> <left> <right> (comma-separated vertex names)")
		return 2
	}
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	ancestor, err := frontierFromList(snap, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	left, err := frontierFromList(snap, args[1])
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	right, err := frontierFromList(snap, args[2])
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	_, mr, err := newService().MergeThreeWay(context.Background(), state, ancestor, left, right, nil)
	if err != nil {
		fmt.Fprintf(stderr, "merge3: %v\n", err)
		return 1
	}
	return printMergeResult(stdout, snap, mr)
}

func cmdMaterialize(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "materialize: takes no arguments")
		return 2
	}
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "materialize: %v\n", err)
		return 1
	}
	ids := make([]identity.VertexID, 0, len(snap.Vertices))
	for _, v := range snap.Vertices {
		ids = append(ids, vid(v.Name))
	}
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
func frontierFromList(snap *snapshot, csv string) (projection.Frontier, error) {
	names := splitCSV(csv)
	if len(names) == 0 {
		return nil, fmt.Errorf("empty vertex list")
	}
	ids := make([]identity.VertexID, 0, len(names))
	for _, name := range names {
		if _, ok := snap.vertexByName(name); !ok {
			return nil, fmt.Errorf("unknown vertex %q", name)
		}
		ids = append(ids, vid(name))
	}
	return projection.NewEditedFrontier(ids), nil
}

// printMergeResult renders a MergeResult: the merged vertex names + witness on
// success, or the typed conflicts on failure. Returns the process exit code.
func printMergeResult(stdout io.Writer, snap *snapshot, mr composition.MergeResult) int {
	idx := snap.nameIndex()
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
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, ok := snap.vertexByName(from); !ok {
		fmt.Fprintf(stderr, "trace: unknown vertex %q\n", from)
		return 1
	}
	if _, ok := snap.vertexByName(to); !ok {
		fmt.Fprintf(stderr, "trace: unknown vertex %q\n", to)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "trace: %v\n", err)
		return 1
	}
	eng := provenance.NewEngine(ontology.CausalEdges)
	ctx := context.Background()
	connected, err := eng.Causes(ctx, state.Graph(), vid(from), vid(to))
	if err != nil {
		fmt.Fprintf(stderr, "trace: %v\n", err)
		return 1
	}
	if !connected {
		fmt.Fprintf(stdout, "%s and %s are not causally connected\n", from, to)
		return 0
	}
	traces, err := eng.TraceSet(ctx, state.Graph(), vid(from), vid(to))
	if err != nil {
		fmt.Fprintf(stderr, "trace: %v\n", err)
		return 1
	}
	idx := snap.nameIndex()
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
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, ok := snap.vertexByName(name); !ok {
		fmt.Fprintf(stderr, "cone: unknown vertex %q\n", name)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "cone: %v\n", err)
		return 1
	}
	eng := provenance.NewEngine(ontology.CausalEdges)
	cone, err := eng.Cone(context.Background(), state.Graph(), vid(name))
	if err != nil {
		fmt.Fprintf(stderr, "cone: %v\n", err)
		return 1
	}
	idx := snap.nameIndex()
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
