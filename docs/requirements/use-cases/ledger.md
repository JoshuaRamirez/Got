# UC ledger

Status of every use case in the catalogue. Update this file in the same
change that moves a UC between statuses.

## Status values

| Status | Meaning |
|---|---|
| Specified | UC is documented in the catalogue; no implementation yet. |
| Partial | Some paths implemented; not all extensions covered. |
| Implemented | All paths covered by code; no behavioral tests on the UC's level. |
| Verified | Implemented and exercised by behavioral tests covering the main success path and at least one failure path per extension group. |
| Retired | UC removed from active scope; ID retained for stability. |

`Verified` is the only status that satisfies the test-gating rule in
`docs/design-rules.md`. Anything below `Verified` blocks the UC's user from
relying on the system.

## Update protocol

When a commit changes the implementation or test coverage of any UC:

1. Update the row in this ledger in the same commit.
2. Bump `Last reviewed` to the commit's UTC date.
3. Cite the implementing package or file in `Implementation`.
4. Cite the test file(s) in `Tests`.
5. Note any partial-coverage caveats in `Notes`.

When a UC is retired:

- Change status to `Retired`. Do not delete the row.
- Add a `Notes` entry citing the commit and reason.

## User use cases

| ID | Title | Status | Implementation | Tests | Last reviewed | Notes |
|---|---|---|---|---|---|---|
| [UC-U01](user/UC-U01-ingest-content.md) | Ingest content into repository | Verified | `internal/repo/service.go` (`Ingest`) | `internal/repo/repo_test.go` | 2026-05-05 | VertexPayload and EdgePayload handled; nil-payload, unknown-kind, missing-endpoint failure paths covered. |
| [UC-U02](user/UC-U02-revise-graph.md) | Revise the graph via a rewrite rule | Verified | `internal/repo/service.go` (`Revise`) | `internal/repo/repo_test.go` | 2026-05-05 | Delegates to revision.Engine.Apply; add-vertex rule exercised end-to-end through the facade. |
| [UC-U03](user/UC-U03-create-or-update-branch.md) | Create or update a branch | Verified | `internal/repo/service.go` (`Branch`) | `internal/repo/repo_test.go` | 2026-05-05 | Ingest + Branch + ResolveRef cycle and missing-target failure path covered. |
| [UC-U04](user/UC-U04-merge-frontiers.md) | Merge two frontiers | Verified | `internal/repo/service.go` (`Merge`) | `internal/repo/repo_test.go`, `internal/composition/composition_props_test.go` | 2026-06-10 | Routes to composition.Engine.Merge; happy path with two-vertex union exercised. Property tests assert set-union law, frontier commutativity, idempotence, witness determinism, and the XOR invariant across 300 random frontier pairs. |
| [UC-U05](user/UC-U05-evaluate-frontier.md) | Evaluate a frontier in an environment | Verified | `internal/repo/service.go` (`Evaluate`) | `internal/repo/repo_test.go` | 2026-05-05 | Routes to verification.Engine.Evaluate; ScalarResult round-trip exercised. |
| [UC-U06](user/UC-U06-materialize-bundle.md) | Materialize a bundle from a projection | Verified | `internal/repo/service.go` (`Materialize`) | `internal/repo/repo_test.go` | 2026-05-05 | Project + Materialize chain via ManifestTarget; bundle path count verified. |
| [UC-U07](user/UC-U07-promote-release.md) | Promote a frontier to a release alias | Verified | `internal/release/service.go` (`Promote`) | `internal/release/release_test.go` | 2026-05-05 | Happy path + empty-frontier, nil-certificate, target-mismatch failure paths covered. Trusts the supplied certificate's gate decision. |
| [UC-U08](user/UC-U08-rollback-release.md) | Rollback a release alias | Verified | `internal/release/service.go` (`Rollback`) | `internal/release/release_test.go` | 2026-05-05 | Two-version Promote/Rollback cycle and ErrUnknownVersion failure path covered. In-memory ledger keyed by (alias, version). |
| [UC-U09](user/UC-U09-resolve-name.md) | Resolve a name to a vertex | Verified | `internal/namespace/mem.go` | `internal/namespace/namespace_test.go` | 2026-05-05 | Main path + unbound-name failure path covered. |
| [UC-U10](user/UC-U10-query-graph.md) | Query the graph | Verified | `internal/graph/mem.go`, `internal/graph/query.go` | `internal/graph/graph_test.go`, `internal/graph/query_test.go` | 2026-06-16 | `Vertex`/`Edge`/`Hyperedge`/`VertexIDs`/`Induce` covered. `Query` now evaluates a composable query language (ByType, ByAttr, And, Or) returning the induced subgraph (UC-S24); unknown query types still return `ErrQueryUnsupported`. |
| [UC-U11](user/UC-U11-trace-provenance.md) | Trace causal provenance | Verified | `internal/provenance/engine.go` | `internal/provenance/provenance_test.go`, `internal/provenance/provenance_props_test.go` | 2026-06-10 | All four read methods covered including reflexivity, monotonicity, idempotence axioms. Property tests assert extensive/idempotent/monotone closure, Cone=singleton-Close, Close=union-of-cones, Causes symmetry and closure-agreement, and TraceSet simple-path well-formedness across 300 random causal graphs. |
| [UC-U12](user/UC-U12-trace-authorship.md) | Trace authorship and responsibility | Verified | `internal/multiagent/engine.go` | `internal/multiagent/multiagent_test.go` | 2026-05-05 | Authorship and ResponsibilityTrace covered; ErrNoAuthorship and graph.ErrVertexNotFound failure paths exercised. |
| [UC-U13](user/UC-U13-check-freshness.md) | Check temporal freshness of a vertex | Verified | `internal/temporal/engine.go` | `internal/temporal/temporal_test.go` | 2026-05-05 | Validity, Fresh half-open semantics, indefinite-`ValidTo`, malformed triple, and unknown-vertex paths covered. |
| [UC-U14](user/UC-U14-replay-capsule.md) | Replay a change capsule | Verified | `internal/replay/engine.go` (`Replay`) | `internal/replay/replay_test.go` | 2026-05-05 | Delegates to revision.Replayable. Happy path, empty-environment "any" treatment, env-mismatch and consumed-missing failure paths covered. |
| [UC-U15](user/UC-U15-prove-claim.md) | Prove a claim with a proof | Verified | `internal/verification/engine.go` (`Prove`) | `internal/verification/verification_test.go` | 2026-05-05 | Proves edge → true, no-edge → false, vertex-not-found failure path covered. |
| [UC-U16](user/UC-U16-detect-emergent-capability.md) | Detect an emergent capability | Verified | `internal/capability/engine.go` (`Emerges`) | `internal/capability/capability_test.go` | 2026-05-05 | Predicate-list dispatch; first-match wins; built-in `CertifiedNonEmpty` predicate; ErrNoEmergence failure path covered. |
| [UC-U17](user/UC-U17-resolve-merge-conflicts.md) | Resolve merge conflicts | Verified | `internal/composition/engine.go` (`Resolve`), reachable via `repo.Service.Merge` then re-call | `internal/composition/composition_test.go` | 2026-05-05 | Composition.Resolve verified; UC-U17 main path "actor invokes composition.Engine.Resolve" is fully covered via the composition behavioral test. |
| [UC-U18](user/UC-U18-three-way-merge.md) | Three-way merge against a common ancestor | Verified | `internal/composition/threeway.go` (`DefaultEngine.MergeThreeWay`, `ThreeWayMerger`); facade `internal/repo/service.go` (`MergeThreeWay`) | `internal/composition/threeway_test.go`, `internal/repo/repo_test.go` | 2026-06-16 | Additive concrete method (no composition.Engine interface change). Ancestor-relative reconciliation: only-left/only-right/agreed change, add, honored deletion, both-delete success paths; modify/modify, add/add, modify/delete, schema, and Unsat-policy conflict paths; plain-frontier presence-only degradation; ctx cancellation. Content via projection.Edited. Reachable through repo.Service.MergeThreeWay (delegates to ThreeWayMerger; repo.ErrThreeWayUnsupported otherwise); facade one-sided-change and conflict paths covered. Edge-level reconciliation (via EdgeEdits) mirrors the vertex rules — only-left/add/honored-deletion success paths; modify/modify, add/add, modify/delete Structural conflict paths covered. |
| [UC-U19](user/UC-U19-operate-from-cli.md) | Operate the repository from the command line | Verified | `cmd/got` (`run.go`, `store.go`, `helpers.go`) | `cmd/got/run_test.go` | 2026-06-16 | CLI shell over the library, persisted as a repository directory under $GOT_DIR via repo.SaveState/LoadState (UC-U20): graph.json + namespace.json. Human names carried in the reserved `got.name` attribute so they survive the codec. init/add-vertex/add-edge/bind/resolve/list/trace/cone/merge/merge3/materialize drive repo.Ingest, repo.Branch, repo.Merge, repo.MergeThreeWay, repo.Materialize, namespace.ResolveRef, and provenance.Engine. Tests cover happy paths plus unknown-command, before-init, unknown-type, duplicate, inadmissible-edge (state unchanged), missing-endpoint, bind-unknown, resolve-unbound, unconnected-trace, merge-unknown-vertex, merge3 deletion-honoring, materialize, and cross-invocation persistence. New delivery channel for UC-U01/U03/U04/U06/U09/U10/U11/U18/U20/S08 — no new engine behavior. |

| [UC-U20](user/UC-U20-persist-reload-repository.md) | Persist and reload a repository | Verified | `internal/repo/persist.go` (`SaveState`, `LoadState`) | `internal/repo/repo_test.go` | 2026-06-16 | Directory persistence: graph.json (UC-S23 codec, explicit SaveState, atomic write-then-rename) + namespace.json (UC-S22 FileStore, continuous). End-to-end save→reload round-trip through the facade (graph + edge + ref survive a simulated restart); empty-dir load; repeated-save overwrite; corrupt-graph rejected on load. |

| [UC-U21](user/UC-U21-first-class-branches.md) | Manage first-class branches | Verified | `internal/repo/branch.go` (`CreateBranch`, `Branches`, `BranchLineage`, `Branch`), `internal/ontology/schema.go` (`{BranchSelector,ForksFrom,BranchSelector}`) | `internal/repo/repo_test.go`, `cmd/got/run_test.go` | 2026-06-16 | A branch is a first-class BranchSelector vertex (identity + metadata + forks_from lineage), not a bare pointer; the mutable tip is a namespace binding. Create (with parent + tip + metadata), list, and fork-lineage traversal; ErrBranchExists, ErrUnknownBranch, unknown-tip failure paths. CLI: branch/branches/branch-log. Fork ancestry (branch-log) is a capability git structurally lacks. |

| [UC-U22](user/UC-U22-record-browse-history.md) | Record and browse repository history | Verified | `internal/repo/commit.go` (`Commit`, `LoadHistory`, `SaveHistory`), `cmd/got` (`commit`, `log`) | `internal/repo/repo_test.go`, `cmd/got/run_test.go` | 2026-06-16 | repo.Commit snapshots the current graph into a new commit with a computed vertex-delta and parent = branch's current commit; persisted to history.json; branch commit pointer (commit:<branch> ref) advances. CLI commit -m / log (newest-first ancestry). Tests: commit + ancestry + delta, save/load, empty-dir, CLI commit/log order+authors, no-commits, message-required, persistence. Non-lossy commit history — the git-loses-information fix, end to end. |

| [UC-U23](user/UC-U23-current-branch-checkout-status.md) | Switch branches and see working status | Verified | `cmd/got` (`checkout`/`switch`, `status`, HEAD; `run.go`, `store.go`) | `cmd/got/run_test.go` | 2026-06-16 | Git-style HEAD file (current branch); init sets HEAD=main. checkout/switch [-b] [--force] updates the working graph to the target branch's committed snapshot with dirty-tree safety and branch-existence check. status shows current branch + uncommitted content changes (branch vertices excluded via contentOnly). commit/log/diff default to HEAD. Tests: status flow, checkout -b + working-tree-follows-HEAD, nonexistent, dirty-refused + --force, first-class-branch-not-dirty. |

| [UC-U24](user/UC-U24-merge-branches.md) | Merge a branch semantically | Verified | `internal/repo/merge.go` (`MergeStates`), `internal/history/history.go` (`MergeBase`), `cmd/got` (`merge`, `merge-base`) | `internal/repo/repo_test.go`, `internal/history/history_test.go`, `cmd/got/run_test.go` | 2026-06-16 | merge <branch> into HEAD: MergeBase finds the common commit; fast-forward when current is an ancestor; else semantic three-way merge (UC-U18) of the two tip states -> merge commit with two parents, or typed conflicts (abort). Tests: history merge-base (LCA + unrelated), repo MergeStates clean/conflict, CLI divergent-merge/fast-forward/merge-base/self-refused. Semantic (not textual) branch merge — the headline git-beating op. |

| [UC-U25](user/UC-U25-show-tag-revert.md) | Inspect, tag, and revert commits | Verified | `cmd/got` (`show`, `tag`/`tags`, `revert`; commit-ish resolution + tags.json in store.go) | `cmd/got/run_test.go` | 2026-06-16 | show <commit-ish> prints commit metadata + structural diff vs parent; commit-ish resolves a branch tip, tag, or commit-id prefix. tag/tags name commits (tags.json). revert applies the structural reverse-delta of a commit onto the working graph and records a Revert commit. Tests: tag+show, duplicate-tag, revert (removes added vertex, Revert commit in log), unknown-commit-ish. |

| [UC-U26](user/UC-U26-reset-restore.md) | Reset a branch and restore the working graph | Verified | `cmd/got` (`reset`, `restore`) | `cmd/got/run_test.go` | 2026-06-16 | reset [--hard] <commit-ish> repoints the current branch tip (--hard also rewrites the working graph); restore [<commit-ish>] rewrites the working graph to a commit (default HEAD), discarding uncommitted changes. Tests: reset --hard (drops later commit + clean tree), soft reset (keeps working, status dirty), restore (discards uncommitted). |

| [UC-U27](user/UC-U27-branch-delete-rename.md) | Delete and rename branches | Verified | `cmd/got` (`branch -d`, `branch -m`), `internal/namespace` (`DeleteRef` on Store/memStore/FileStore/HTTPStore + handler) | `cmd/got/run_test.go`, `internal/namespace/*_test.go` | 2026-06-16 | branch -d removes the commit pointer (DeleteRef) + first-class vertex (refused for current branch); branch -m moves the pointer + HEAD, drops the old vertex. DeleteRef added across all Store impls (idempotent, durable in FileStore, routed in HTTP handler). Tests: delete + gone, delete-current-refused, rename + old-gone; namespace mem/file/http DeleteRef. |

| [UC-U28](user/UC-U28-blame-node-history.md) | Blame a node and query its history | Verified | `cmd/got` (`blame`, `log --touching`) | `cmd/got/run_test.go` | 2026-06-16 | blame <name> walks the branch's commit ancestry chronologically to report the introducing and last-changing commits (author+message) for a node; log --touching <name> filters to commits whose graph.Diff vs parent added/removed/changed the node. Per-node provenance — better than git's per-line heuristic. Tests: blame introduced-by, log --touching filters, blame-unknown. |

| [UC-U29](user/UC-U29-cherry-pick-amend.md) | Cherry-pick and amend commits | Verified | `cmd/got` (`cherry-pick`, `amend`) | `cmd/got/run_test.go` | 2026-06-16 | cherry-pick <commit-ish> applies a commit's forward structural delta onto the current working graph and records a new commit; amend [-m] replaces the branch tip with the working state keeping the original parents (old commit orphaned). Tests: cherry-pick brings a node from another branch + new commit; amend folds in a working change + new message + clean status. |

| [UC-U30](user/UC-U30-stash.md) | Stash uncommitted working changes | Verified | `cmd/got` (`stash push`/`pop`/`list`) | `cmd/got/run_test.go` | 2026-06-16 | stash push saves the working snapshot onto a LIFO stack (stash.json) and resets the working graph to HEAD; pop restores the top; list shows the stack. Tests: stash+clean+list+pop restores; nothing-to-stash; pop-empty. |

| [UC-U31](user/UC-U31-rebase.md) | Rebase a branch onto another | Verified | `cmd/got` (`rebase`) | `cmd/got/run_test.go` | 2026-06-16 | rebase <onto>: replay the current branch's commits above the merge base onto <onto>'s tip as new commits (linear history rewrite); fast-forward when current is an ancestor, up-to-date when onto is; refuse unrelated histories / self. Tests: rebase (linear m1<-f1 + working tree), fast-forward, up-to-date. |

## System use cases

| ID | Title | Status | Implementation | Tests | Last reviewed | Notes |
|---|---|---|---|---|---|---|
| [UC-S01](system/UC-S01-validate-graph.md) | Validate graph well-formedness | Verified | `internal/graph/mem.go` (`Validate`) | `internal/graph/graph_test.go` | 2026-05-05 | All four `ErrNotWellFormed` failure modes exercised. |
| [UC-S02](system/UC-S02-apply-dpo-rewrite.md) | Apply a DPO rewrite | Verified | `internal/revision/engine.go` (`Apply`) | `internal/revision/revision_test.go` | 2026-06-10 | Add-vertex, add-edge, delete-vertex paths; ErrNoMatch, ErrSideConditionFailed, ErrNotWellFormed failure paths exercised. Strict bridge: ErrDanglingEdge (delete-side pushout-complement, step 1), ErrIdentityCollision (produce-side content-addressing, step 2), and full hyperedge handling (step 3 — L\K hyperedges deleted, R\K inserted, both Strict audits cover hyperedges) refuse unfaithful rewrites; Lenient drops/overwrites silently. Idempotent re-statement, delete-then-add, and hyperedge insert/delete/collision/restatement variations covered. |
| [UC-S03](system/UC-S03-compute-pushout.md) | Compute the guarded pushout of two frontiers | Verified | `internal/composition/engine.go` (`Merge`) | `internal/composition/composition_test.go`, `internal/composition/composition_props_test.go` | 2026-06-10 | Union frontier with governance gate; Sat → Certificate via verification.Certify; Unsat → Policy-kind Conflict. Identical-frontiers and empty-policy paths covered. Property tests assert the set-union law, frontier commutativity/idempotence, witness determinism, and the merged-xor-conflicted invariant (both directions) across 300 random frontier pairs. Per-side audit now emits Capability/Evaluation conflicts (typed payloads) when same-typed Capability/Evaluation vertices disagree on attrs — the two previously-unreachable ConflictKinds; other types stay Textual. |
| [UC-S04](system/UC-S04-resolve-conflicts.md) | Apply conflict resolutions | Verified | `internal/composition/engine.go` (`Resolve`) | `internal/composition/composition_test.go` | 2026-05-05 | Sequential resolution application + re-merge; no-op resolution and ErrConflictUnresolvable failure path covered. |
| [UC-S05](system/UC-S05-evaluate-in-environment.md) | Evaluate a frontier in a given environment | Verified | `internal/verification/engine.go` (`Evaluate`) | `internal/verification/verification_test.go` | 2026-05-05 | Dispatches to registered Evaluator; main path, no-evaluator and evaluator-error failure paths, ctx cancel covered. |
| [UC-S06](system/UC-S06-issue-certificate.md) | Issue a certificate for a frontier | Verified | `internal/verification/engine.go` (`Certify`) | `internal/verification/verification_test.go` | 2026-05-05 | Delegates to governance.GateRelease; happy path, Unsat failure, outstanding-obligations failure, empty-policy trivial path covered. |
| [UC-S07](system/UC-S07-compute-provenance-closure.md) | Compute the provenance closure of a seed set | Verified | `internal/provenance/engine.go` (`Close`) | `internal/provenance/provenance_test.go`, `internal/provenance/provenance_props_test.go` | 2026-06-10 | Extensivity, monotonicity, idempotence axioms tested by fixtures and by property tests over 300 random causal graphs (incl. Close = union of cones). |
| [UC-S08](system/UC-S08-compute-provenance-cone.md) | Compute the provenance cone of a vertex | Verified | `internal/provenance/engine.go` (`Cone`) | `internal/provenance/provenance_test.go`, `internal/provenance/provenance_props_test.go` | 2026-06-10 | `Cone == Close({seed})` axiom tested by fixtures and property tests over 300 random causal graphs. |
| [UC-S09](system/UC-S09-enumerate-causal-traces.md) | Enumerate causal traces between two vertices | Verified | `internal/provenance/engine.go` (`TraceSet`) | `internal/provenance/provenance_test.go`, `internal/provenance/provenance_props_test.go` | 2026-06-10 | Simple-path enumeration verified by fixtures; property tests assert every trace is a simple path within Close({from}) and TraceSet is non-empty iff endpoints are causally connected. |
| [UC-S10](system/UC-S10-select-frontier.md) | Select a frontier from the graph | Verified | `internal/projection/engine.go` (`Select`, `IDsSelector`) | `internal/projection/projection_test.go` | 2026-05-05 | Main path, empty selector, ErrInvalidSelector failure paths, ctx cancellation covered. |
| [UC-S11](system/UC-S11-apply-projection-spec.md) | Apply a full projection spec | Verified | `internal/projection/engine.go` (`Project`, `InduceSpec`) | `internal/projection/projection_test.go` | 2026-05-05 | Main path + spec-error failure path covered. |
| [UC-S12](system/UC-S12-check-policy-aggregate.md) | Check the aggregate decision over a policy set | Verified | `internal/governance/engine.go` (`Check`) | `internal/governance/governance_test.go` | 2026-05-05 | Aggregate rule (Unsat dominates, then Unknown), empty policy set, obligation concatenation, policy-error wrap, ctx cancel covered. |
| [UC-S13](system/UC-S13-gate-release.md) | Gate a frontier for release | Verified | `internal/governance/engine.go` (`GateRelease`) | `internal/governance/governance_test.go` | 2026-05-05 | Sat + no obligations → true, outstanding obligations block, Unsat blocks, empty policies → trivially true. |
| [UC-S14](system/UC-S14-materialize-for-target.md) | Materialize a view for a specific target | Verified | `internal/realization/engine.go` (`DefaultEngine`, `ManifestTarget`) | `internal/realization/realization_test.go` | 2026-05-05 | Manifest materializer, custom registration, empty view, unsupported-target failure path, ctx cancel covered. |
| [UC-S15](system/UC-S15-bind-name.md) | Bind a name to a vertex | Verified | `internal/namespace/mem.go` (`BindRef`/`BindAlias`/`BindProjection`) | `internal/namespace/namespace_test.go` | 2026-05-05 | All three name kinds exercised; rebind-overwrite path covered. DeleteRef added (remove a ref; idempotent) across memStore/FileStore/HTTPStore for branch delete/rename. |
| [UC-S16](system/UC-S16-resolve-binding.md) | Resolve a name binding | Verified | `internal/namespace/mem.go` (`Resolve*`) | `internal/namespace/namespace_test.go` | 2026-05-05 | Bound + unbound paths covered. |
| [UC-S17](system/UC-S17-compute-content-id.md) | Compute a content-addressed identifier | Verified | `internal/identity/sha256.go` | `internal/identity/identity_test.go` | 2026-05-05 | SHA-256-backed factory; canonical-bytes contract covered. |
| [UC-S18](system/UC-S18-check-ontology-admissibility.md) | Check whether an edge or hyperedge is admissible | Verified | `internal/ontology/schema.go` | `internal/ontology/schema_test.go` | 2026-05-05 | Edge and hyperedge admissibility tables exercised. |
| [UC-S19](system/UC-S19-check-replay-feasibility.md) | Check whether a change capsule is replayable | Verified | `internal/revision/engine.go` (`Replayable`) | `internal/revision/revision_test.go` | 2026-05-05 | Happy path, empty capsule, consumed-missing, produced-missing failure paths covered. |
| [UC-S20](system/UC-S20-check-temporal-validity.md) | Check the temporal validity of a vertex | Verified | `internal/temporal/engine.go` (`Validity`) | `internal/temporal/temporal_test.go` | 2026-05-05 | Main path, malformed-triple and unknown-vertex failure paths covered. |
| [UC-S21](system/UC-S21-audit-frontier-wellformedness.md) | Audit a frontier for structural and temporal well-formedness | Verified | `internal/composition/audit.go` (`DefaultEngine.Audit`, `Auditor`); consumer `internal/repo/service.go` (`ReleaseStrict`) | `internal/composition/composition_test.go`, `internal/repo/repo_test.go`, `internal/repo/integration_test.go` | 2026-06-16 | In-graph structural/temporal audit exposed independently of Merge; strictness-independent. Auditor capability assertion, temporal-detect and clean paths covered. repo.Service.ReleaseStrict runs it before the gate (closes the seam in TestIntegrationTemporalConflictSurfaceArea): blocks a malformed TimeTriple with ErrReleaseAudit that plain Release accepts; clean frontier passes; ErrAuditUnsupported when the engine is not an Auditor. |
| [UC-S22](system/UC-S22-persist-namespace.md) | Persist namespace bindings to durable storage | Verified | `internal/namespace/file.go` (`FileStore`) | `internal/namespace/file_test.go` | 2026-06-16 | Durable, concurrency-safe Store backed by an atomic JSON file. All three name kinds bind/resolve; durability across reopen; rebind-overwrite; corrupt-file error; unbound; 16-goroutine concurrent-writer test under -race. Only the mutable namespace is persisted (graph is content-addressed/reconstructable). |
| [UC-S23](system/UC-S23-serialize-graph.md) | Serialize and deserialize a graph | Verified | `internal/graph/codec.go` (`EncodeSnapshot`, `Snapshot.Build`, `Marshal`, `Unmarshal`) | `internal/graph/codec_test.go` | 2026-06-16 | Lossless snapshot/JSON codec carrying all vertex/edge/hyperedge fields (hex IDs). Round-trip (incl. attrs/time/trust/edges/hyperedge) and JSON round-trip; empty; validate-on-load runs graph.Validate so malformed-ID, missing-endpoint, and inadmissible snapshots are rejected on decode. |
| [UC-S24](system/UC-S24-evaluate-graph-query.md) | Evaluate a graph query | Verified | `internal/graph/query.go` (`ByType`, `ByAttr`, `And`, `Or`, `matchVertices`), `internal/graph/mem.go` (`Query`) | `internal/graph/query_test.go` | 2026-06-16 | Composable query language: ByType, ByAttr (deep-equality, absent-key non-match), And (intersection), Or (union), nesting, empty composites, induced-edge inclusion, and ErrQueryUnsupported for unknown types (incl. propagation through composites). |
| [UC-S27](system/UC-S27-structural-diff.md) | Compute a structural diff between two graph states | Verified | `internal/graph/diff.go` (`Diff`, `Delta`, `VertexChange`, `EdgeChange`), CLI `cmd/got` (`diff`) | `internal/graph/diff_test.go`, `cmd/got/run_test.go` | 2026-06-16 | Structure-aware (not textual) diff of two snapshots, matched by ID: added/removed/changed vertices and edges. Changed is meaningful because IDs are sha256(name); identical → Empty; reversible. CLI: diff <branch> (vs parent) / diff <a> <b>. Tests: identical/added-removed/changed/edges + CLI last-commit/no-commits/bad-args. |
| [UC-S26](system/UC-S26-commit-history.md) | Record operation-first commit history | Verified | `internal/history/history.go` (`Commit`, `Log`, `NewCommit`, `Ancestors`, `Marshal`/`Unmarshal`) | `internal/history/history_test.go` | 2026-06-16 | Operation-first commit DAG: each commit records its consumed/produced delta + resulting snapshot + parents + actor + message; content-addressed CommitID (delta excluded from identity). Merge commits (multi-parent), BFS ancestry walk, unknown-parent/unknown-commit failures, JSON round-trip. The non-lossy alternative to git's snapshot-only commits. |
| [UC-S25](system/UC-S25-remote-namespace.md) | Bind and resolve names over a network | Verified | `internal/namespace/http.go` (`HTTPStore`, `NewHTTPHandler`) | `internal/namespace/http_test.go` | 2026-06-16 | Network-transparent Store: HTTP client + server over JSON, ctx threaded onto each request. httptest round-trip for all three name kinds (bind lands in backing store), unbound → not-found, URL-escaped special-char names, and cancelled-ctx (bind errors, resolve → not-found). Realizes the remote backing the Store's ctx parameter anticipates. |

## Summary

As of 2026-06-16:

| Layer | Specified | Partial | Implemented | Verified | Retired | Total |
|---|---:|---:|---:|---:|---:|---:|
| User | 0 | 0 | 0 | 31 | 0 | 31 |
| System | 0 | 0 | 0 | 27 | 0 | 27 |
| **Total** | **0** | **0** | **0** | **58** | **0** | **58** |

**Verified coverage: 58 / 58 = 100%.** UC-U18 (three-way merge) and
UC-U19 (`cmd/got` shell) added 2026-06-10; UC-S21 (frontier audit /
Strict-on-Release), UC-S22 (durable `FileStore` namespace), UC-S23
(graph snapshot codec), and UC-U20 (repository persist/reload) added
2026-06-16. All roadmap phases complete (see `roadmap.md`). Every public method on every internal `Engine` and
`Service` is reachable from at least one user use case and exercised by
at least one system use case, with behavioral tests covering the main
success path and at least one failure path per extension group.

## Next-bite candidates

The roadmap is complete. Subsequent work is either:

- **Hardening** — additional failure-path tests, fuzz testing, race
  testing under load, benchmarks.
- **New UCs** — add new requirements via `/use-case new`; they enter the
  catalogue at `Specified` and follow the same lifecycle.
- **Composability** — concrete materializers for non-manifest targets,
  domain-specific evaluators, additional emergence predicates.

Any new package added under `internal/` must be slotted into a phase in
`roadmap.md` (or motivate adding a new phase) and given a UC.
