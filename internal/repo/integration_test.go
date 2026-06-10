package repo_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/realization"
	"github.com/joshuaramirez/got/internal/repo"
	"github.com/joshuaramirez/got/internal/revision"
	"github.com/joshuaramirez/got/internal/verification"
)

// Integration tests for repo.DefaultService drive multi-step scenarios
// end-to-end through the facade rather than testing individual methods
// in isolation. Each test is a narrative: a sequence of operations that
// a real user would perform, with assertions on the observable state at
// each step.

// helper: build a fully-wired service with a Lenient composition engine.
func integrationService(t *testing.T) (*repo.DefaultService, verification.Evaluator) {
	t.Helper()
	gov := governance.NewEngine()
	eval := verification.EvaluatorFunc(func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
		return verification.ScalarResult(1.0), nil
	})
	ver := verification.NewEngine(gov, eval)
	svc := repo.NewService(
		composition.NewEngine(gov, ver),
		gov,
		projection.NewEngine(),
		realization.NewEngine(),
		revision.NewEngine(),
		ver,
	)
	return svc, eval
}

// helper: build a fully-wired service with a Strict composition engine.
func integrationServiceStrict(t *testing.T) *repo.DefaultService {
	t.Helper()
	gov := governance.NewEngine()
	eval := verification.EvaluatorFunc(func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
		return verification.ScalarResult(1.0), nil
	})
	ver := verification.NewEngine(gov, eval)
	return repo.NewService(
		composition.NewEngineStrict(gov, ver),
		gov,
		projection.NewEngine(),
		realization.NewEngine(),
		revision.NewEngine(),
		ver,
	)
}

func freshState() repo.State {
	g := graph.NewGraph(ontology.NewDefaultSchema())
	return repo.NewState(g, namespace.NewStore())
}

func id(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func edgeID(s string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(s)))
}

// --- Scenario 1: full content lifecycle through the facade ---

// Given a fresh repository, when an author ingests content, branches
// it, materializes a manifest, and releases under no policies, then
// every step preserves the prior step's effects and the final manifest
// covers the released artifacts.
func TestIntegrationContentLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, _ := integrationService(t)
	state := freshState()

	artifact := graph.Vertex{ID: id("lifecycle-artifact"), Type: ontology.Artifact}
	revisionV := graph.Vertex{ID: id("lifecycle-revision"), Type: ontology.Revision}

	t.Run("Given_fresh_state_When_ingest_content_Then_graph_grows", func(t *testing.T) {
		var err error
		state, err = svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{artifact, revisionV}})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := state.Graph().Vertex(artifact.ID); !ok {
			t.Fatal("artifact not present after ingest")
		}
		if _, ok := state.Graph().Vertex(revisionV.ID); !ok {
			t.Fatal("revision not present after ingest")
		}
	})

	t.Run("Given_ingested_artifact_When_branch_Then_ref_resolves", func(t *testing.T) {
		var err error
		state, err = svc.Branch(ctx, state, "main", artifact.ID)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := state.Namespace().ResolveRef(ctx, "main")
		if !ok || got != artifact.ID {
			t.Fatalf("ResolveRef(main) = (%v, %v), want (%v, true)", got, ok, artifact.ID)
		}
	})

	t.Run("Given_branched_state_When_materialize_Then_bundle_covers_view", func(t *testing.T) {
		bundle, err := svc.Materialize(ctx, state,
			projection.InduceSpec{IDs: []identity.VertexID{artifact.ID, revisionV.ID}},
			realization.ManifestTarget)
		if err != nil {
			t.Fatal(err)
		}
		if len(bundle.Paths()) != 2 {
			t.Fatalf("expected 2 paths in manifest, got %d", len(bundle.Paths()))
		}
		// Provenance for each path must lie inside the materialized view.
		viewSet := map[identity.VertexID]bool{artifact.ID: true, revisionV.ID: true}
		for _, p := range bundle.Paths() {
			for _, prov := range bundle.Provenance(p) {
				if !viewSet[prov] {
					t.Fatalf("provenance %v for path %q escapes view", prov, p)
				}
			}
		}
	})

	t.Run("Given_materialized_state_When_release_empty_policies_Then_succeeds", func(t *testing.T) {
		pe := projection.NewEngine()
		f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{artifact.ID}})
		_, err := svc.Release(ctx, state, f, nil)
		if err != nil {
			t.Fatalf("release with no policies should succeed, got %v", err)
		}
	})
}

// --- Scenario 2: disjoint merge through the facade ---

// Given two disjoint frontiers, when merged through the facade, then
// the union is the merged frontier and a certificate is issued.
func TestIntegrationDisjointMerge(t *testing.T) {
	ctx := context.Background()
	svc, _ := integrationService(t)
	state := freshState()

	a := graph.Vertex{ID: id("merge-a"), Type: ontology.Artifact}
	b := graph.Vertex{ID: id("merge-b"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a, b}})
	if err != nil {
		t.Fatal(err)
	}

	pe := projection.NewEngine()
	left, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{a.ID}})
	right, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{b.ID}})

	_, mr, err := svc.Merge(ctx, state, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.Conflicts) != 0 {
		t.Fatalf("disjoint merge should have no conflicts, got %v", mr.Conflicts)
	}
	if mr.Frontier == nil {
		t.Fatal("disjoint merge should produce a frontier")
	}
	if len(mr.Frontier.VertexIDs()) != 2 {
		t.Fatalf("merged frontier should have 2 IDs, got %d", len(mr.Frontier.VertexIDs()))
	}
	if mr.Certificate == nil {
		t.Fatal("disjoint merge should issue a certificate")
	}
}

// --- Scenario 3: revise-then-replay-feasibility round trip ---

// Given an ingested vertex, when a rewrite rule adds an edge to a new
// vertex, then the rewritten graph contains both endpoints and the new
// edge.
func TestIntegrationReviseAddsEdge(t *testing.T) {
	ctx := context.Background()
	svc, _ := integrationService(t)
	state := freshState()

	a := graph.Vertex{ID: id("revise-a"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a}})
	if err != nil {
		t.Fatal(err)
	}

	// Rule: L = {a}, K = {a}, R = {a, b, e:b->a (Revision -DerivedFrom-> Artifact)}.
	b := graph.Vertex{ID: id("revise-b"), Type: ontology.Revision}
	newEdge := graph.Edge{ID: edgeID("revise-e"), Type: ontology.DerivedFrom, From: b.ID, To: a.ID}

	leftSub := &inlineSub{ids: []identity.VertexID{a.ID}, verts: []graph.Vertex{a}}
	ctxSub := &inlineSub{ids: []identity.VertexID{a.ID}, verts: []graph.Vertex{a}}
	rightSub := &inlineSub{
		ids:   []identity.VertexID{a.ID, b.ID},
		verts: []graph.Vertex{a, b},
		edges: []graph.Edge{newEdge},
	}
	rule := integrationRule{left: leftSub, ctx: ctxSub, right: rightSub}
	match := integrationMatch{m: map[identity.VertexID]identity.VertexID{a.ID: a.ID}}

	out, err := svc.Revise(ctx, state, rule, match)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Graph().Vertex(b.ID); !ok {
		t.Fatal("revision did not add b")
	}
	if _, ok := out.Graph().Edge(newEdge.ID); !ok {
		t.Fatal("revision did not add the derived_from edge")
	}
}

// --- Scenario 4: strict-mode conflict detection through the facade ---

// Given two EditedFrontiers disagreeing on a vertex's Attrs, when
// merged through a Strict facade, then a Textual conflict is returned
// and the merged frontier is empty.
func TestIntegrationStrictDetectsTextualConflict(t *testing.T) {
	ctx := context.Background()
	svc := integrationServiceStrict(t)
	state := freshState()

	a := graph.Vertex{ID: id("strict-attrs"), Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a}})
	if err != nil {
		t.Fatal(err)
	}

	left := projection.NewEditedFrontier([]identity.VertexID{a.ID})
	left.Vertices[a.ID] = graph.Vertex{ID: a.ID, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}}
	right := projection.NewEditedFrontier([]identity.VertexID{a.ID})
	right.Vertices[a.ID] = graph.Vertex{ID: a.ID, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "review"}}

	_, mr, err := svc.Merge(ctx, state, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Frontier != nil {
		t.Fatal("strict merge with conflict should leave Frontier zero")
	}
	if !hasConflictKind(mr.Conflicts, composition.Textual) {
		t.Fatalf("expected Textual conflict, got %v", mr.Conflicts)
	}
}

// --- Scenario 5: temporal-malformed vertex blocks release ---

// Given a vertex with a malformed TimeTriple (ValidTo < ValidFrom),
// when a Strict-mode merge is attempted that includes it, then a
// Temporal conflict is surfaced; when a release is attempted with
// Sat policies, then the release still succeeds (release does NOT
// run the Strict audit — only governance gating).
//
// This documents a real seam: Strict mode only fires through Merge.
// Release uses governance.GateRelease directly and is blind to
// temporal audits. If a UC wanted release to also catch temporal
// malformation, the audit would need to be added to Release too.
func TestIntegrationTemporalConflictSurfaceArea(t *testing.T) {
	ctx := context.Background()
	svc := integrationServiceStrict(t)
	state := freshState()

	bad := graph.Vertex{
		ID:   id("temporal-bad"),
		Type: ontology.Artifact,
		Time: graph.TimeTriple{ValidFrom: 500, ValidTo: 100},
	}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{bad}})
	if err != nil {
		t.Fatal(err)
	}

	pe := projection.NewEngine()
	f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{bad.ID}})

	// Strict merge surfaces the Temporal conflict.
	_, mr, err := svc.Merge(ctx, state, f, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasConflictKind(mr.Conflicts, composition.Temporal) {
		t.Fatalf("expected Temporal conflict on malformed TimeTriple, got %v", mr.Conflicts)
	}

	// Release does NOT run the Strict audit; with Sat policies it succeeds.
	_, err = svc.Release(ctx, state, f, nil)
	if err != nil {
		t.Errorf("release does not audit Strict; expected success, got %v", err)
	}
}

// --- Scenario 6: evaluate-certify-merge composition ---

// Given two evaluations attached to a frontier, when merged under a
// Sat policy, then the merge issues a certificate. This exercises the
// composition → verification → governance call chain through the
// facade.
func TestIntegrationEvaluateThenMerge(t *testing.T) {
	ctx := context.Background()
	svc, _ := integrationService(t)
	state := freshState()

	a := graph.Vertex{ID: id("eval-a"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a}})
	if err != nil {
		t.Fatal(err)
	}

	pe := projection.NewEngine()
	f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{a.ID}})

	// Evaluate produces a ScalarResult(1.0).
	_, eval, err := svc.Evaluate(ctx, state, f, verification.EnvironmentBinding{ID: id("env"), Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := eval.Result().(verification.ScalarResult); s != 1.0 {
		t.Fatalf("Evaluate = %v, want 1.0", eval.Result())
	}

	// Merge with a passing policy: certificate issued.
	_, mr, err := svc.Merge(ctx, state, f, f, []governance.Policy{satPolicy{}})
	if err != nil {
		t.Fatal(err)
	}
	if mr.Certificate == nil {
		t.Fatal("expected certificate when merge succeeds under Sat policy")
	}
}

// --- Scenario 7: release blocked by Unsat policy ---

// Given a frontier and an Unsat policy, when release is attempted,
// then it is blocked with an obligation count.
func TestIntegrationReleaseBlocked(t *testing.T) {
	ctx := context.Background()
	svc, _ := integrationService(t)
	state := freshState()

	a := graph.Vertex{ID: id("release-blocked"), Type: ontology.Artifact}
	state, _ = svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a}})

	pe := projection.NewEngine()
	f, _ := pe.Select(ctx, state.Graph(), projection.IDsSelector{IDs: []identity.VertexID{a.ID}})

	_, err := svc.Release(ctx, state, f, []governance.Policy{
		unsatPolicy{obligations: []governance.Obligation{{Code: "REQ-1"}}},
	})
	if err == nil {
		t.Fatal("expected release to be blocked by Unsat policy")
	}
}

// --- Scenario 8: ingest error rolls back graph state ---

// Given a partially-valid VertexPayload (one vertex OK, one with a
// duplicate-but-different vertex would still succeed under append
// semantics; instead: an EdgePayload with a missing endpoint), when
// ingested, then ErrIngestRejected is returned and the prior state is
// unchanged.
func TestIntegrationIngestErrorPreservesState(t *testing.T) {
	ctx := context.Background()
	svc, _ := integrationService(t)
	state := freshState()

	a := graph.Vertex{ID: id("rollback-a"), Type: ontology.Artifact}
	state, err := svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{a}})
	if err != nil {
		t.Fatal(err)
	}
	originalVCount := len(state.Graph().Vertices())

	// Bad edge: endpoint not in graph.
	bad := graph.Edge{ID: edgeID("rollback-bad"), Type: ontology.DerivedFrom, From: id("nonexistent"), To: a.ID}
	_, err = svc.Ingest(ctx, state, repo.EdgePayload{Edges: []graph.Edge{bad}})
	if !errors.Is(err, repo.ErrIngestRejected) {
		t.Fatalf("expected ErrIngestRejected, got %v", err)
	}
	if len(state.Graph().Vertices()) != originalVCount {
		t.Fatalf("input state should be unchanged after failed ingest, vertex count drifted: %d → %d",
			originalVCount, len(state.Graph().Vertices()))
	}
}

// --- Scenario 9: strict-mode resolver round trip ---

// Given a Textual conflict, when ResolveTyped is applied with
// PreferLeftAttr against cloned frontiers, then the re-merge succeeds
// and the original frontiers are unchanged (Clone hygiene).
func TestIntegrationStrictResolverWithClone(t *testing.T) {
	ctx := context.Background()
	gov := governance.NewEngine()
	ver := verification.NewEngine(gov, verification.EvaluatorFunc(
		func(graph.Graph, projection.Frontier, verification.EnvironmentBinding) (verification.ResultValue, error) {
			return verification.ScalarResult(1.0), nil
		}))
	e := composition.NewEngineStrict(gov, ver)

	g := graph.NewGraph(ontology.NewDefaultSchema())
	a := id("resolver-roundtrip")
	g, _ = g.WithVertex(graph.Vertex{ID: a, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}})

	leftOrig := projection.NewEditedFrontier([]identity.VertexID{a})
	leftOrig.Vertices[a] = graph.Vertex{ID: a, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "draft"}}
	rightOrig := projection.NewEditedFrontier([]identity.VertexID{a})
	rightOrig.Vertices[a] = graph.Vertex{ID: a, Type: ontology.Artifact, Attrs: graph.AttrMap{"status": "review"}}

	// Clone inputs so the resolver does not mutate the originals.
	left := leftOrig.Clone()
	right := rightOrig.Clone()

	mr, err := e.Merge(ctx, g, left, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasConflictKind(mr.Conflicts, composition.Textual) {
		t.Fatalf("expected Textual conflict, got %v", mr.Conflicts)
	}

	resolved, err := e.ResolveTyped(ctx, g, left, right, mr,
		[]composition.Resolver{composition.PreferLeftAttr("status")})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Conflicts) != 0 {
		t.Fatalf("expected zero conflicts after resolve, got %v", resolved.Conflicts)
	}

	// Originals unchanged.
	if rightOrig.Vertices[a].Attrs["status"] != "review" {
		t.Fatalf("right ORIGINAL was mutated despite Clone: %v", rightOrig.Vertices[a].Attrs["status"])
	}
}

// --- Scenario 10: namespace persistence across ingests ---

// Given a branch bound, when subsequent ingests extend the graph,
// then the branch still resolves to the original vertex.
func TestIntegrationBranchSurvivesIngests(t *testing.T) {
	ctx := context.Background()
	svc, _ := integrationService(t)
	state := freshState()

	root := graph.Vertex{ID: id("branch-root"), Type: ontology.Artifact}
	state, _ = svc.Ingest(ctx, state, repo.VertexPayload{Vertices: []graph.Vertex{root}})
	state, _ = svc.Branch(ctx, state, "main", root.ID)

	// Ingest more vertices.
	more := []graph.Vertex{
		{ID: id("branch-other-1"), Type: ontology.Artifact},
		{ID: id("branch-other-2"), Type: ontology.Artifact},
	}
	state, _ = svc.Ingest(ctx, state, repo.VertexPayload{Vertices: more})

	got, ok := state.Namespace().ResolveRef(ctx, "main")
	if !ok || got != root.ID {
		t.Fatalf("ResolveRef(main) = (%v, %v), want (%v, true) after additional ingests",
			got, ok, root.ID)
	}
}

// --- helpers ---

// hasConflictKind reports whether any conflict in cs has the given kind.
func hasConflictKind(cs []composition.Conflict, k composition.ConflictKind) bool {
	for _, c := range cs {
		if c.Kind() == k {
			return true
		}
	}
	return false
}

// inlineSub implements graph.Subgraph for ad-hoc rule construction.
type inlineSub struct {
	ids   []identity.VertexID
	verts []graph.Vertex
	edges []graph.Edge
}

func (s *inlineSub) VertexIDs() []identity.VertexID { return s.ids }
func (s *inlineSub) Vertices() []graph.Vertex       { return s.verts }
func (s *inlineSub) Edges() []graph.Edge            { return s.edges }
func (s *inlineSub) Hyperedges() []graph.Hyperedge  { return nil }

type integrationRule struct {
	left, ctx, right graph.Subgraph
}

func (r integrationRule) Left() graph.Subgraph                 { return r.left }
func (r integrationRule) Context() graph.Subgraph              { return r.ctx }
func (r integrationRule) Right() graph.Subgraph                { return r.right }
func (r integrationRule) SideConditions() []revision.Predicate { return nil }

type integrationMatch struct {
	m map[identity.VertexID]identity.VertexID
}

func (m integrationMatch) Mapping() map[identity.VertexID]identity.VertexID { return m.m }

type satPolicy struct{}

func (satPolicy) Name() string { return "sat" }
func (satPolicy) Check(graph.Graph, projection.Frontier) (governance.Decision, []governance.Obligation, error) {
	return governance.Sat, nil, nil
}

type unsatPolicy struct {
	obligations []governance.Obligation
}

func (unsatPolicy) Name() string { return "unsat" }
func (p unsatPolicy) Check(graph.Graph, projection.Frontier) (governance.Decision, []governance.Obligation, error) {
	return governance.Unsat, p.obligations, nil
}
