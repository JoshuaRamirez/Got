package main

import (
	"bytes"
	"strings"
	"testing"
)

// runCLI invokes run with a fresh state directory rooted at t.TempDir via the
// GOT_DIR env var, returning exit code and captured stdout/stderr.
func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// initRepo sets GOT_DIR to a temp dir and runs `init`.
func initRepo(t *testing.T) {
	t.Helper()
	t.Setenv("GOT_DIR", t.TempDir())
	if code, _, errs := runCLI(t, "init"); code != 0 {
		t.Fatalf("init failed: code=%d err=%s", code, errs)
	}
}

func TestNoArgsShowsUsage(t *testing.T) {
	code, _, errs := runCLI(t, "")
	// empty string is a single arg; run treats args[0]=="" as unknown command.
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0 (err=%s)", errs)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errs := runCLI(t, "frobnicate")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(errs, "unknown command") {
		t.Fatalf("expected unknown-command diagnostic, got %q", errs)
	}
}

func TestCommandBeforeInit(t *testing.T) {
	t.Setenv("GOT_DIR", t.TempDir())
	code, _, errs := runCLI(t, "list", "vertices")
	if code == 0 {
		t.Fatal("expected non-zero exit before init")
	}
	if !strings.Contains(errs, "run 'got init'") {
		t.Fatalf("expected init hint, got %q", errs)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	t.Setenv("GOT_DIR", t.TempDir())
	if code, _, _ := runCLI(t, "init"); code != 0 {
		t.Fatal("first init should succeed")
	}
	code, out, _ := runCLI(t, "init")
	if code != 0 {
		t.Fatal("second init should still exit 0")
	}
	if !strings.Contains(out, "already exists") {
		t.Fatalf("expected already-exists message, got %q", out)
	}
}

func TestAddVertexAndList(t *testing.T) {
	initRepo(t)
	if code, _, errs := runCLI(t, "add-vertex", "art", "--type", "Artifact"); code != 0 {
		t.Fatalf("add-vertex failed: %s", errs)
	}
	code, out, _ := runCLI(t, "list", "vertices")
	if code != 0 {
		t.Fatal("list failed")
	}
	if !strings.Contains(out, "art\tArtifact") {
		t.Fatalf("expected vertex in list, got %q", out)
	}
}

func TestAddVertexUnknownType(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "add-vertex", "x", "--type", "Nonsense")
	if code != 1 {
		t.Fatalf("expected exit 1 for unknown type, got %d", code)
	}
	if !strings.Contains(errs, "unknown vertex type") {
		t.Fatalf("expected unknown-type diagnostic, got %q", errs)
	}
}

func TestAddVertexDuplicate(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "x", "--type", "Artifact")
	code, _, errs := runCLI(t, "add-vertex", "x", "--type", "Artifact")
	if code != 1 || !strings.Contains(errs, "already exists") {
		t.Fatalf("expected duplicate rejection, code=%d err=%q", code, errs)
	}
}

func TestAddEdgeAdmissible(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "exec", "--type", "Execution")
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	if code, _, errs := runCLI(t, "add-edge", "e1", "--type", "materializes", "--from", "exec", "--to", "art"); code != 0 {
		t.Fatalf("admissible edge should succeed: %s", errs)
	}
	_, out, _ := runCLI(t, "list", "edges")
	if !strings.Contains(out, "exec -materializes-> art") {
		t.Fatalf("expected edge in list, got %q", out)
	}
}

func TestAddEdgeInadmissibleRejected(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "exec", "--type", "Execution")
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	// Execution -authored_by-> Artifact is not in the admissibility table.
	code, _, errs := runCLI(t, "add-edge", "bad", "--type", "authored_by", "--from", "exec", "--to", "art")
	if code != 1 {
		t.Fatalf("expected exit 1 for inadmissible edge, got %d", code)
	}
	if !strings.Contains(errs, "not admissible") {
		t.Fatalf("expected admissibility error, got %q", errs)
	}
	// State must be unchanged: the bad edge is not persisted.
	_, out, _ := runCLI(t, "list", "edges")
	if strings.Contains(out, "bad") {
		t.Fatal("rejected edge must not be persisted")
	}
}

func TestAddEdgeMissingEndpoint(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "exec", "--type", "Execution")
	code, _, errs := runCLI(t, "add-edge", "e1", "--type", "materializes", "--from", "exec", "--to", "ghost")
	if code != 1 || !strings.Contains(errs, "unknown --to vertex") {
		t.Fatalf("expected missing-endpoint rejection, code=%d err=%q", code, errs)
	}
}

func TestBindAndResolve(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	if code, _, errs := runCLI(t, "bind", "main", "art"); code != 0 {
		t.Fatalf("bind failed: %s", errs)
	}
	code, out, _ := runCLI(t, "resolve", "main")
	if code != 0 {
		t.Fatal("resolve failed")
	}
	if !strings.Contains(out, "main -> art") {
		t.Fatalf("expected resolved ref, got %q", out)
	}
}

func TestBindUnknownVertex(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "bind", "main", "ghost")
	if code != 1 {
		t.Fatalf("expected exit 1 binding unknown vertex, got %d", code)
	}
	if !strings.Contains(errs, "not found") {
		t.Fatalf("expected vertex-not-found, got %q", errs)
	}
}

func TestResolveUnbound(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "resolve", "nope")
	if code != 1 || !strings.Contains(errs, "unbound") {
		t.Fatalf("expected unbound diagnostic, code=%d err=%q", code, errs)
	}
}

// Build a small causal chain and verify trace + cone.
func TestTraceAndCone(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "exec", "--type", "Execution")
	runCLI(t, "add-vertex", "prompt", "--type", "Prompt")
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	runCLI(t, "add-edge", "e1", "--type", "derived_from", "--from", "exec", "--to", "prompt")
	runCLI(t, "add-edge", "e2", "--type", "materializes", "--from", "exec", "--to", "art")

	code, out, errs := runCLI(t, "trace", "prompt", "art")
	if code != 0 {
		t.Fatalf("trace failed: %s", errs)
	}
	if !strings.Contains(out, "path(s)") || !strings.Contains(out, "prompt -> exec -> art") {
		t.Fatalf("expected a causal path prompt->exec->art, got %q", out)
	}

	code, out, _ = runCLI(t, "cone", "exec")
	if code != 0 {
		t.Fatal("cone failed")
	}
	for _, n := range []string{"exec", "prompt", "art"} {
		if !strings.Contains(out, n) {
			t.Fatalf("cone(exec) should contain %q, got %q", n, out)
		}
	}
}

func TestTraceUnconnected(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	code, out, _ := runCLI(t, "trace", "a", "b")
	if code != 0 {
		t.Fatal("trace of unconnected vertices should still exit 0")
	}
	if !strings.Contains(out, "not causally connected") {
		t.Fatalf("expected unconnected message, got %q", out)
	}
}

func TestReviseDerivesRevision(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	code, out, errs := runCLI(t, "revise", "art", "rev1")
	if code != 0 {
		t.Fatalf("revise failed: code=%d err=%s", code, errs)
	}
	if !strings.Contains(out, "added revision \"rev1\"") {
		t.Fatalf("expected revise confirmation, got %q", out)
	}
	// The produced Revision vertex and derived_from edge are persisted.
	_, vout, _ := runCLI(t, "list", "vertices")
	if !strings.Contains(vout, "rev1\tRevision") {
		t.Fatalf("expected persisted revision vertex, got %q", vout)
	}
	_, eout, _ := runCLI(t, "list", "edges")
	if !strings.Contains(eout, "rev1 -derived_from-> art") {
		t.Fatalf("expected persisted derived_from edge, got %q", eout)
	}
}

func TestReviseUnknownArtifact(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "revise", "ghost", "rev1")
	if code != 1 || !strings.Contains(errs, "unknown artifact") {
		t.Fatalf("expected unknown-artifact rejection, code=%d err=%q", code, errs)
	}
}

func TestReviseNonArtifactAnchor(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "p", "--type", "Prompt")
	code, _, errs := runCLI(t, "revise", "p", "rev1")
	if code != 1 || !strings.Contains(errs, "not an Artifact") {
		t.Fatalf("expected non-Artifact rejection, code=%d err=%q", code, errs)
	}
}

func TestMergeTwoWayUnion(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	code, out, errs := runCLI(t, "merge", "--left", "a", "--right", "b")
	if code != 0 {
		t.Fatalf("merge failed: code=%d err=%s", code, errs)
	}
	if !strings.Contains(out, "merged 2 vertex(es)") || !strings.Contains(out, "a, b") {
		t.Fatalf("expected union of a,b, got %q", out)
	}
}

func TestMergeThreeWayHonorsAddition(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "base", "--type", "Artifact")
	runCLI(t, "add-vertex", "x", "--type", "Artifact")
	// ancestor={base}, left={base}, right={base,x}: x is a right-side addition.
	code, out, errs := runCLI(t, "merge", "--left", "base", "--right", "base,x", "--ancestor", "base")
	if code != 0 {
		t.Fatalf("three-way merge failed: code=%d err=%s", code, errs)
	}
	if !strings.Contains(out, "merged 2 vertex(es)") || !strings.Contains(out, "base, x") {
		t.Fatalf("expected base,x in three-way merge, got %q", out)
	}
}

func TestMergeMissingFlags(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "merge", "--left", "a")
	if code != 2 || !strings.Contains(errs, "required") {
		t.Fatalf("expected required-flags diagnostic, code=%d err=%q", code, errs)
	}
}

func TestMergeUnknownVertex(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	code, _, errs := runCLI(t, "merge", "--left", "a", "--right", "ghost")
	if code != 1 || !strings.Contains(errs, "unknown vertex") {
		t.Fatalf("expected unknown-vertex rejection, code=%d err=%q", code, errs)
	}
}

func TestMaterializeManifest(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	code, out, errs := runCLI(t, "materialize", "a,b")
	if code != 0 {
		t.Fatalf("materialize failed: code=%d err=%s", code, errs)
	}
	if !strings.Contains(out, "materialized manifest: 2 path(s)") {
		t.Fatalf("expected 2-path manifest, got %q", out)
	}
}

func TestMaterializeUnsupportedTarget(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	code, _, errs := runCLI(t, "materialize", "a", "--target", "nonsense")
	if code != 1 || !strings.Contains(errs, "unsupported") {
		t.Fatalf("expected unsupported-target rejection, code=%d err=%q", code, errs)
	}
}

func TestReleaseNoPolicies(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	code, out, errs := runCLI(t, "release", "a")
	if code != 0 {
		t.Fatalf("release failed: code=%d err=%s", code, errs)
	}
	if !strings.Contains(out, "released 1 vertex(es)") {
		t.Fatalf("expected release confirmation, got %q", out)
	}
}

func TestReleaseUnknownVertex(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "release", "ghost")
	if code != 1 || !strings.Contains(errs, "unknown vertex") {
		t.Fatalf("expected unknown-vertex rejection, code=%d err=%q", code, errs)
	}
}

// Persistence: state written by one invocation is visible to the next, even
// across a fresh process-equivalent run call.
func TestPersistenceAcrossInvocations(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	// A completely separate run call (new buffers) must see the vertex.
	_, out, _ := runCLI(t, "list", "vertices")
	if !strings.Contains(out, "art") {
		t.Fatalf("expected persisted vertex across invocations, got %q", out)
	}
}
