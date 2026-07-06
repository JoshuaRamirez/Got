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

// --- merge / merge3 / materialize ---

// merge3 honors a one-sided deletion: art deleted on left, unchanged on right.
func TestMerge3HonorsDeletion(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "exec", "--type", "Execution")
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	code, out, errs := runCLI(t, "merge3", "exec,art", "exec", "exec,art")
	if code != 0 {
		t.Fatalf("merge3 failed: %s", errs)
	}
	if !strings.Contains(out, "merged 1 vertex(es): exec") {
		t.Fatalf("expected deletion of art honored (merged={exec}), got %q", out)
	}
}

func TestMaterializeCommand(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	code, out, errs := runCLI(t, "materialize")
	if code != 0 {
		t.Fatalf("materialize failed: %s", errs)
	}
	if !strings.Contains(out, "target=manifest") || !strings.Contains(out, "2 path(s)") {
		t.Fatalf("expected a 2-path manifest bundle, got %q", out)
	}
}

// --- first-class branches (UC-U21) ---

func TestBranchCreateAndList(t *testing.T) {
	initRepo(t)
	if code, _, errs := runCLI(t, "branch", "main"); code != 0 {
		t.Fatalf("branch main: %s", errs)
	}
	if code, out, errs := runCLI(t, "branch", "feature", "--from", "main", "--desc", "new work"); code != 0 {
		t.Fatalf("branch feature: %s (%s)", errs, out)
	}
	code, out, _ := runCLI(t, "branches")
	if code != 0 {
		t.Fatal("branches failed")
	}
	if !strings.Contains(out, "main") || !strings.Contains(out, "feature") {
		t.Fatalf("expected both branches, got %q", out)
	}
	if !strings.Contains(out, "(from main)") || !strings.Contains(out, "new work") {
		t.Fatalf("expected parent + desc metadata, got %q", out)
	}
}

// branch-log shows fork ancestry — the thing git can't represent.
func TestBranchLog(t *testing.T) {
	initRepo(t)
	runCLI(t, "branch", "main")
	runCLI(t, "branch", "release-2", "--from", "main")
	runCLI(t, "branch", "feature", "--from", "release-2")

	code, out, errs := runCLI(t, "branch-log", "feature")
	if code != 0 {
		t.Fatalf("branch-log: %s", errs)
	}
	if !strings.Contains(out, "feature <- release-2 <- main") {
		t.Fatalf("expected fork ancestry, got %q", out)
	}
}

func TestBranchUnknownParent(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "branch", "x", "--from", "ghost")
	if code != 1 || !strings.Contains(errs, "unknown branch") {
		t.Fatalf("expected unknown-parent rejection, code=%d err=%q", code, errs)
	}
}

func TestBranchDuplicate(t *testing.T) {
	initRepo(t)
	runCLI(t, "branch", "main")
	code, _, errs := runCLI(t, "branch", "main")
	if code != 1 || !strings.Contains(errs, "already exists") {
		t.Fatalf("expected duplicate rejection, code=%d err=%q", code, errs)
	}
}

// A branch tip binds to a vertex and resolves like any ref.
func TestBranchWithTip(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "art", "--type", "Artifact")
	if code, _, errs := runCLI(t, "branch", "main", "--tip", "art"); code != 0 {
		t.Fatalf("branch with tip: %s", errs)
	}
	code, out, _ := runCLI(t, "resolve", "main")
	if code != 0 || !strings.Contains(out, "main -> art") {
		t.Fatalf("expected tip to resolve to art, got %q", out)
	}
	// The branch persists across invocations as a first-class vertex.
	_, out, _ = runCLI(t, "branches")
	if !strings.Contains(out, "main") {
		t.Fatalf("branch should persist, got %q", out)
	}
}

// --- commit / log (UC-U22) ---

func TestCommitAndLog(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	if code, out, errs := runCLI(t, "commit", "-m", "add a", "--actor", "alice"); code != 0 {
		t.Fatalf("commit: %s %s", errs, out)
	}
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	if code, _, errs := runCLI(t, "commit", "-m", "add b", "--actor", "bob"); code != 0 {
		t.Fatalf("commit 2: %s", errs)
	}
	code, out, _ := runCLI(t, "log")
	if code != 0 {
		t.Fatal("log failed")
	}
	// Newest first, with messages and authors.
	if !strings.Contains(out, "add b") || !strings.Contains(out, "add a") {
		t.Fatalf("expected both commits in log, got %q", out)
	}
	if !strings.Contains(out, "bob") || !strings.Contains(out, "alice") {
		t.Fatalf("expected authors in log, got %q", out)
	}
	if strings.Index(out, "add b") > strings.Index(out, "add a") {
		t.Fatalf("expected newest-first order, got %q", out)
	}
}

func TestLogNoCommits(t *testing.T) {
	initRepo(t)
	code, out, _ := runCLI(t, "log")
	if code != 0 || !strings.Contains(out, "no commits") {
		t.Fatalf("expected 'no commits', code=%d out=%q", code, out)
	}
}

func TestCommitRequiresMessage(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	code, _, errs := runCLI(t, "commit", "--actor", "x")
	if code != 2 || !strings.Contains(errs, "required") {
		t.Fatalf("expected message-required error, code=%d err=%q", code, errs)
	}
}

// Commits persist across invocations (separate process-equivalent calls).
func TestCommitPersistence(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "first")
	// Fresh run call must see the commit.
	_, out, _ := runCLI(t, "log")
	if !strings.Contains(out, "first") {
		t.Fatalf("commit should persist across invocations, got %q", out)
	}
}

// --- diff (UC-S27 via CLI) ---

func TestDiffLastCommit(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "v1")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "add-edge", "e", "--type", "derived_from", "--from", "b", "--to", "a")
	runCLI(t, "commit", "-m", "v2")

	code, out, errs := runCLI(t, "diff", "main")
	if code != 0 {
		t.Fatalf("diff: %s", errs)
	}
	if !strings.Contains(out, "+ vertex b") || !strings.Contains(out, "+ edge e") {
		t.Fatalf("expected the v2 additions, got %q", out)
	}
	if strings.Contains(out, "vertex a") {
		t.Fatalf("a was in the parent; it should not appear as a change: %q", out)
	}
}

func TestDiffNoCommits(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "diff", "main")
	if code != 1 || !strings.Contains(errs, "no commits") {
		t.Fatalf("expected no-commits error, code=%d err=%q", code, errs)
	}
}

func TestDiffBadArgs(t *testing.T) {
	initRepo(t)
	code, _, _ := runCLI(t, "diff", "a", "b", "c")
	if code != 2 {
		t.Fatalf("expected usage error for 3 args, got %d", code)
	}
}

// --- HEAD / checkout / status (UC-U23) ---

func TestStatusFlow(t *testing.T) {
	initRepo(t)
	code, out, _ := runCLI(t, "status")
	if code != 0 || !strings.Contains(out, "On branch main") || !strings.Contains(out, "clean") {
		t.Fatalf("fresh status: %q", out)
	}
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	_, out, _ = runCLI(t, "status")
	if !strings.Contains(out, "Uncommitted changes") || !strings.Contains(out, "+ vertex a") {
		t.Fatalf("dirty status: %q", out)
	}
	runCLI(t, "commit", "-m", "add a")
	_, out, _ = runCLI(t, "status")
	if !strings.Contains(out, "clean") {
		t.Fatalf("post-commit status should be clean: %q", out)
	}
}

func TestCheckoutCreateSwitchAndWorkingTree(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")

	if code, out, errs := runCLI(t, "checkout", "-b", "dev"); code != 0 || !strings.Contains(out, "dev") {
		t.Fatalf("checkout -b dev: %s %s", errs, out)
	}
	// commit defaults to HEAD (dev now).
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add b")
	_, out, _ := runCLI(t, "log")
	if !strings.Contains(out, "add b") {
		t.Fatalf("log should default to dev: %q", out)
	}

	// Switch back to main: working tree follows HEAD (no b).
	runCLI(t, "checkout", "main")
	_, out, _ = runCLI(t, "list", "vertices")
	if !strings.Contains(out, "a") || strings.Contains(out, "b\t") {
		t.Fatalf("main working tree should have a, not b: %q", out)
	}
}

func TestCheckoutNonexistent(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "checkout", "ghost")
	if code != 1 || !strings.Contains(errs, "no such branch") {
		t.Fatalf("expected no-such-branch, code=%d err=%q", code, errs)
	}
}

func TestCheckoutDirtyRefused(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")
	runCLI(t, "checkout", "-b", "dev")
	runCLI(t, "add-vertex", "b", "--type", "Artifact") // uncommitted
	code, _, errs := runCLI(t, "checkout", "main")
	if code != 1 || !strings.Contains(errs, "uncommitted") {
		t.Fatalf("expected dirty-refusal, code=%d err=%q", code, errs)
	}
	// --force discards and switches.
	if code, _, errs := runCLI(t, "checkout", "--force", "main"); code != 0 {
		t.Fatalf("checkout --force should succeed: %s", errs)
	}
}

// A first-class branch vertex does not count as an uncommitted content change.
func TestFirstClassBranchNotDirty(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")
	runCLI(t, "branch", "feature")
	_, out, _ := runCLI(t, "status")
	if !strings.Contains(out, "clean") {
		t.Fatalf("creating a first-class branch should not dirty status: %q", out)
	}
}

// --- semantic merge (UC-U24) ---

func TestMergeSemanticDivergent(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "base", "--type", "Artifact")
	runCLI(t, "commit", "-m", "base")
	runCLI(t, "checkout", "-b", "feature")
	runCLI(t, "add-vertex", "feat", "--type", "Artifact")
	runCLI(t, "commit", "-m", "feature work")
	runCLI(t, "checkout", "main")
	runCLI(t, "add-vertex", "mainw", "--type", "Artifact")
	runCLI(t, "commit", "-m", "main work")

	code, out, errs := runCLI(t, "merge", "feature")
	if code != 0 {
		t.Fatalf("merge failed: %s %s", errs, out)
	}
	if !strings.Contains(out, "merged") {
		t.Fatalf("expected a merge commit, got %q", out)
	}
	// Working tree has both sides.
	_, out, _ = runCLI(t, "list", "vertices")
	for _, n := range []string{"base", "feat", "mainw"} {
		if !strings.Contains(out, n) {
			t.Fatalf("merged tree should contain %q: %q", n, out)
		}
	}
	// Log shows the merge commit atop both lines.
	_, out, _ = runCLI(t, "log")
	if !strings.Contains(out, "merge feature into main") {
		t.Fatalf("expected merge commit in log: %q", out)
	}
}

func TestMergeFastForward(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "a")
	runCLI(t, "checkout", "-b", "topic")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "commit", "-m", "b")
	runCLI(t, "checkout", "main")
	code, out, _ := runCLI(t, "merge", "topic")
	if code != 0 || !strings.Contains(out, "fast-forward") {
		t.Fatalf("expected fast-forward, code=%d out=%q", code, out)
	}
}

func TestMergeBaseCmd(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "a")
	runCLI(t, "checkout", "-b", "x")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "commit", "-m", "b")
	runCLI(t, "checkout", "main")
	runCLI(t, "add-vertex", "c", "--type", "Artifact")
	runCLI(t, "commit", "-m", "c")
	code, out, _ := runCLI(t, "merge-base", "main", "x")
	if code != 0 || len(strings.TrimSpace(out)) < 12 {
		t.Fatalf("expected a merge-base id, got %q", out)
	}
}

func TestMergeSelfRefused(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "a")
	code, _, errs := runCLI(t, "merge", "main")
	if code != 1 || !strings.Contains(errs, "into itself") {
		t.Fatalf("expected self-merge refusal, code=%d err=%q", code, errs)
	}
}

// --- show / tag / revert (UC-U25) ---

func TestTagAndShow(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add b")

	if code, out, errs := runCLI(t, "tag", "v1"); code != 0 || !strings.Contains(out, "v1") {
		t.Fatalf("tag: %s %s", errs, out)
	}
	_, out, _ := runCLI(t, "tags")
	if !strings.Contains(out, "v1") {
		t.Fatalf("tags list: %q", out)
	}
	// show by tag: metadata + diff.
	code, out, _ := runCLI(t, "show", "v1")
	if code != 0 || !strings.Contains(out, "add b") || !strings.Contains(out, "+ vertex b") {
		t.Fatalf("show v1: %q", out)
	}
}

func TestTagDuplicate(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "a")
	runCLI(t, "tag", "v1")
	code, _, errs := runCLI(t, "tag", "v1")
	if code != 1 || !strings.Contains(errs, "already exists") {
		t.Fatalf("expected duplicate-tag error, code=%d err=%q", code, errs)
	}
}

func TestRevert(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add b")

	if code, out, errs := runCLI(t, "revert", "main"); code != 0 || !strings.Contains(out, "reverted") {
		t.Fatalf("revert: %s %s", errs, out)
	}
	// b is gone from the working tree.
	_, out, _ := runCLI(t, "list", "vertices")
	if strings.Contains(out, "b\t") {
		t.Fatalf("revert should have removed b: %q", out)
	}
	if !strings.Contains(out, "a\t") {
		t.Fatalf("a should remain: %q", out)
	}
	// log shows the Revert commit on top.
	_, out, _ = runCLI(t, "log")
	if !strings.Contains(out, "Revert: add b") {
		t.Fatalf("expected a revert commit: %q", out)
	}
}

func TestShowUnknown(t *testing.T) {
	initRepo(t)
	code, _, errs := runCLI(t, "show", "nope")
	if code != 1 || !strings.Contains(errs, "unknown commit-ish") {
		t.Fatalf("expected unknown-commit-ish, code=%d err=%q", code, errs)
	}
}

// --- reset / restore (UC-U26) ---

func oldestCommitShort(t *testing.T) string {
	t.Helper()
	_, out, _ := runCLI(t, "log")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		t.Fatal("no commits")
	}
	return strings.Split(lines[len(lines)-1], "\t")[0]
}

func TestResetHard(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")
	first := oldestCommitShort(t)
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add b")

	if code, out, errs := runCLI(t, "reset", "--hard", first); code != 0 || !strings.Contains(out, "hard") {
		t.Fatalf("reset --hard: %s %s", errs, out)
	}
	_, out, _ := runCLI(t, "list", "vertices")
	if strings.Contains(out, "b\t") || !strings.Contains(out, "a\t") {
		t.Fatalf("reset --hard should drop b, keep a: %q", out)
	}
	_, out, _ = runCLI(t, "status")
	if !strings.Contains(out, "clean") {
		t.Fatalf("reset --hard should leave a clean tree: %q", out)
	}
}

func TestResetSoftKeepsWorking(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")
	first := oldestCommitShort(t)
	runCLI(t, "add-vertex", "b", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add b")

	if code, out, _ := runCLI(t, "reset", first); code != 0 || !strings.Contains(out, "kept") {
		t.Fatalf("soft reset: %q", out)
	}
	// Working tree still has b, so status is dirty relative to the new tip.
	_, out, _ := runCLI(t, "status")
	if !strings.Contains(out, "+ vertex b") {
		t.Fatalf("soft reset should keep b as uncommitted: %q", out)
	}
}

func TestRestore(t *testing.T) {
	initRepo(t)
	runCLI(t, "add-vertex", "a", "--type", "Artifact")
	runCLI(t, "commit", "-m", "add a")
	runCLI(t, "add-vertex", "junk", "--type", "Artifact") // uncommitted
	if code, out, errs := runCLI(t, "restore"); code != 0 || !strings.Contains(out, "restored") {
		t.Fatalf("restore: %s %s", errs, out)
	}
	_, out, _ := runCLI(t, "status")
	if !strings.Contains(out, "clean") {
		t.Fatalf("restore should clean the tree: %q", out)
	}
	_, out, _ = runCLI(t, "list", "vertices")
	if strings.Contains(out, "junk") {
		t.Fatalf("restore should drop junk: %q", out)
	}
}
