package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"

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
	case "trace":
		return cmdTrace(rest, stdout, stderr)
	case "cone":
		return cmdCone(rest, stdout, stderr)
	case "revise":
		return cmdRevise(rest, stdout, stderr)
	case "merge":
		return cmdMerge(rest, stdout, stderr)
	case "materialize":
		return cmdMaterialize(rest, stdout, stderr)
	case "release":
		return cmdRelease(rest, stdout, stderr)
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
  got trace <from> <to>
  got cone <name>
  got revise <artifact> <new-revision>
  got merge --left <v,...> --right <v,...> [--ancestor <v,...>]
  got materialize <v,...> [--target manifest|manifest.json]
  got release <v,...>

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

// cmdRevise applies a DPO rewrite that derives a new Revision vertex from an
// existing Artifact, adding a derived_from edge from the revision to the
// artifact. It exercises repo.Service.Revise (UC-U02) end-to-end and persists
// the produced vertex and edge.
func cmdRevise(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "revise: expected <artifact> <new-revision>")
		return 2
	}
	anchor, newName := args[0], args[1]

	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	anchorRec, ok := snap.vertexByName(anchor)
	if !ok {
		fmt.Fprintf(stderr, "revise: unknown artifact %q\n", anchor)
		return 1
	}
	if anchorRec.Type != string(ontology.Artifact) {
		fmt.Fprintf(stderr, "revise: %q is %s, not an Artifact\n", anchor, anchorRec.Type)
		return 1
	}
	if _, exists := snap.vertexByName(newName); exists {
		fmt.Fprintf(stderr, "revise: vertex %q already exists\n", newName)
		return 1
	}

	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "revise: %v\n", err)
		return 1
	}

	edgeName := fmt.Sprintf("%s-derived_from-%s", newName, anchor)
	anchorV := graph.Vertex{ID: vid(anchor), Type: ontology.VertexType(anchorRec.Type), Attrs: attrMap(anchorRec.Attrs)}
	revV := graph.Vertex{ID: vid(newName), Type: ontology.Revision}
	newEdge := graph.Edge{ID: eid(edgeName), Type: ontology.DerivedFrom, From: revV.ID, To: anchorV.ID}

	r := rule{
		left:  subgraph{ids: []identity.VertexID{anchorV.ID}, verts: []graph.Vertex{anchorV}},
		ctx:   subgraph{ids: []identity.VertexID{anchorV.ID}, verts: []graph.Vertex{anchorV}},
		right: subgraph{ids: []identity.VertexID{anchorV.ID, revV.ID}, verts: []graph.Vertex{anchorV, revV}, edges: []graph.Edge{newEdge}},
	}
	m := match{m: map[identity.VertexID]identity.VertexID{anchorV.ID: anchorV.ID}}

	svc := newService()
	if _, err := svc.Revise(context.Background(), state, r, m); err != nil {
		fmt.Fprintf(stderr, "revise: %v\n", err)
		return 1
	}

	snap.Vertices = append(snap.Vertices, vertexRec{Name: newName, Type: string(ontology.Revision)})
	snap.Edges = append(snap.Edges, edgeRec{Name: edgeName, Type: string(ontology.DerivedFrom), From: newName, To: anchor})
	if err := saveSnapshot(snap); err != nil {
		fmt.Fprintf(stderr, "revise: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "revised %q: added revision %q derived from %q\n", anchor, newName, anchor)
	return 0
}

// cmdMerge reconciles two frontiers of named vertices. With --ancestor it runs
// the three-way merge (repo.Service.MergeThreeWay, UC-U18); otherwise the
// two-way union merge (repo.Service.Merge, UC-U04). It reports the merged
// vertex set or the typed conflicts, and does not mutate persisted state.
func cmdMerge(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("merge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var leftCSV, rightCSV, ancestorCSV string
	fs.StringVar(&leftCSV, "left", "", "comma-separated left vertex names")
	fs.StringVar(&rightCSV, "right", "", "comma-separated right vertex names")
	fs.StringVar(&ancestorCSV, "ancestor", "", "comma-separated ancestor vertex names (enables three-way)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if leftCSV == "" || rightCSV == "" {
		fmt.Fprintln(stderr, "merge: --left and --right are required")
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

	leftIDs, err := resolveNames(snap, splitCSV(leftCSV))
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}
	rightIDs, err := resolveNames(snap, splitCSV(rightCSV))
	if err != nil {
		fmt.Fprintf(stderr, "merge: %v\n", err)
		return 1
	}

	ctx := context.Background()
	pe := projection.NewEngine()
	left, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: leftIDs})
	right, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: rightIDs})

	svc := newService()
	var mr composition.MergeResult
	if ancestorCSV != "" {
		ancestorIDs, err := resolveNames(snap, splitCSV(ancestorCSV))
		if err != nil {
			fmt.Fprintf(stderr, "merge: %v\n", err)
			return 1
		}
		ancestor, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: ancestorIDs})
		_, mr, err = svc.MergeThreeWay(ctx, state, ancestor, left, right, nil)
		if err != nil {
			fmt.Fprintf(stderr, "merge: %v\n", err)
			return 1
		}
	} else {
		_, mr, err = svc.Merge(ctx, state, left, right, nil)
		if err != nil {
			fmt.Fprintf(stderr, "merge: %v\n", err)
			return 1
		}
	}

	if len(mr.Conflicts) > 0 {
		idx := snap.nameIndex()
		fmt.Fprintf(stdout, "merge: %d conflict(s)\n", len(mr.Conflicts))
		for _, c := range mr.Conflicts {
			names := make([]string, 0, len(c.Boundary()))
			for _, id := range c.Boundary() {
				names = append(names, idx[id])
			}
			sort.Strings(names)
			fmt.Fprintf(stdout, "  %s: %s\n", c.Kind(), joinComma(names))
		}
		return 1
	}

	idx := snap.nameIndex()
	names := make([]string, 0, len(mr.Frontier.VertexIDs()))
	for _, id := range mr.Frontier.VertexIDs() {
		names = append(names, idx[id])
	}
	sort.Strings(names)
	fmt.Fprintf(stdout, "merged %d vertex(es): %s\n", len(names), joinComma(names))
	return 0
}

// cmdMaterialize projects the subgraph induced by the named vertices and
// materializes it for a target (repo.Service.Materialize, UC-U06). It prints
// the bundle's target and emitted paths.
func cmdMaterialize(args []string, stdout, stderr io.Writer) int {
	names, rest, ok := splitName(args)
	if !ok {
		fmt.Fprintln(stderr, "materialize: expected '<v,...> [--target manifest|manifest.json]'")
		return 2
	}
	fs := flag.NewFlagSet("materialize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var target string
	fs.StringVar(&target, "target", string(realization.ManifestTarget), "materialization target")
	if err := fs.Parse(rest); err != nil {
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
	ids, err := resolveNames(snap, splitCSV(names))
	if err != nil {
		fmt.Fprintf(stderr, "materialize: %v\n", err)
		return 1
	}

	svc := newService()
	bundle, err := svc.Materialize(context.Background(), state, projection.InduceSpec{IDs: ids}, realization.Target(target))
	if err != nil {
		fmt.Fprintf(stderr, "materialize: %v\n", err)
		return 1
	}

	paths := append([]string(nil), bundle.Paths()...)
	sort.Strings(paths)
	fmt.Fprintf(stdout, "materialized %s: %d path(s)\n", bundle.Target(), len(paths))
	for _, p := range paths {
		fmt.Fprintf(stdout, "  %s\n", p)
	}
	return 0
}

// cmdRelease gates a frontier of named vertices for release
// (repo.Service.Release, UC-U07). With no policy set the governance gate is
// vacuously satisfied; the command reports the released vertex count.
func cmdRelease(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "release: expected <v,...>")
		return 2
	}
	snap, err := loadSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state, err := snap.buildState()
	if err != nil {
		fmt.Fprintf(stderr, "release: %v\n", err)
		return 1
	}
	ids, err := resolveNames(snap, splitCSV(args[0]))
	if err != nil {
		fmt.Fprintf(stderr, "release: %v\n", err)
		return 1
	}

	ctx := context.Background()
	pe := projection.NewEngine()
	f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: ids})

	svc := newService()
	if _, err := svc.Release(ctx, state, f, nil); err != nil {
		fmt.Fprintf(stderr, "release: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "released %d vertex(es)\n", len(ids))
	return 0
}
