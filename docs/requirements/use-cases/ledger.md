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
| [UC-U10](user/UC-U10-query-graph.md) | Query the graph | Verified | `internal/graph/mem.go` | `internal/graph/graph_test.go` | 2026-05-05 | `Vertex`/`Edge`/`Hyperedge`/`VertexIDs`/`Induce` covered; `Query` returns `ErrQueryUnsupported` (covered). |
| [UC-U11](user/UC-U11-trace-provenance.md) | Trace causal provenance | Verified | `internal/provenance/engine.go` | `internal/provenance/provenance_test.go`, `internal/provenance/provenance_props_test.go` | 2026-06-10 | All four read methods covered including reflexivity, monotonicity, idempotence axioms. Property tests assert extensive/idempotent/monotone closure, Cone=singleton-Close, Close=union-of-cones, Causes symmetry and closure-agreement, and TraceSet simple-path well-formedness across 300 random causal graphs. |
| [UC-U12](user/UC-U12-trace-authorship.md) | Trace authorship and responsibility | Verified | `internal/multiagent/engine.go` | `internal/multiagent/multiagent_test.go` | 2026-05-05 | Authorship and ResponsibilityTrace covered; ErrNoAuthorship and graph.ErrVertexNotFound failure paths exercised. |
| [UC-U13](user/UC-U13-check-freshness.md) | Check temporal freshness of a vertex | Verified | `internal/temporal/engine.go` | `internal/temporal/temporal_test.go` | 2026-05-05 | Validity, Fresh half-open semantics, indefinite-`ValidTo`, malformed triple, and unknown-vertex paths covered. |
| [UC-U14](user/UC-U14-replay-capsule.md) | Replay a change capsule | Verified | `internal/replay/engine.go` (`Replay`) | `internal/replay/replay_test.go` | 2026-05-05 | Delegates to revision.Replayable. Happy path, empty-environment "any" treatment, env-mismatch and consumed-missing failure paths covered. |
| [UC-U15](user/UC-U15-prove-claim.md) | Prove a claim with a proof | Verified | `internal/verification/engine.go` (`Prove`) | `internal/verification/verification_test.go` | 2026-05-05 | Proves edge → true, no-edge → false, vertex-not-found failure path covered. |
| [UC-U16](user/UC-U16-detect-emergent-capability.md) | Detect an emergent capability | Verified | `internal/capability/engine.go` (`Emerges`) | `internal/capability/capability_test.go` | 2026-05-05 | Predicate-list dispatch; first-match wins; built-in `CertifiedNonEmpty` predicate; ErrNoEmergence failure path covered. |
| [UC-U17](user/UC-U17-resolve-merge-conflicts.md) | Resolve merge conflicts | Verified | `internal/composition/engine.go` (`Resolve`), reachable via `repo.Service.Merge` then re-call | `internal/composition/composition_test.go` | 2026-05-05 | Composition.Resolve verified; UC-U17 main path "actor invokes composition.Engine.Resolve" is fully covered via the composition behavioral test. |
| [UC-U18](user/UC-U18-three-way-merge.md) | Three-way merge against a common ancestor | Verified | `internal/composition/threeway.go` (`DefaultEngine.MergeThreeWay`, `ThreeWayMerger`); facade `internal/repo/service.go` (`MergeThreeWay`) | `internal/composition/threeway_test.go`, `internal/repo/repo_test.go` | 2026-06-16 | Additive concrete method (no composition.Engine interface change). Ancestor-relative reconciliation: only-left/only-right/agreed change, add, honored deletion, both-delete success paths; modify/modify, add/add, modify/delete, schema, and Unsat-policy conflict paths; plain-frontier presence-only degradation; ctx cancellation. Content via projection.Edited. Reachable through repo.Service.MergeThreeWay (delegates to ThreeWayMerger; repo.ErrThreeWayUnsupported otherwise); facade one-sided-change and conflict paths covered. |
| [UC-U19](user/UC-U19-operate-from-cli.md) | Operate the repository from the command line | Verified | `cmd/got` (`run.go`, `store.go`, `helpers.go`) | `cmd/got/run_test.go` | 2026-06-16 | CLI shell over the library with JSON persistence under $GOT_DIR. init/add-vertex/add-edge/bind/resolve/list/trace/cone/merge/merge3/materialize commands drive repo.Ingest, repo.Branch, repo.Merge, repo.MergeThreeWay, repo.Materialize, namespace.ResolveRef, and provenance.Engine. Tests cover happy paths plus unknown-command, before-init, unknown-type, duplicate, inadmissible-edge (state unchanged), missing-endpoint, bind-unknown, resolve-unbound, unconnected-trace, merge-unknown-vertex, merge3 deletion-honoring, materialize, and cross-invocation persistence. New delivery channel for UC-U01/U03/U04/U06/U09/U10/U11/U18/S08 — no new engine behavior. |

## System use cases

| ID | Title | Status | Implementation | Tests | Last reviewed | Notes |
|---|---|---|---|---|---|---|
| [UC-S01](system/UC-S01-validate-graph.md) | Validate graph well-formedness | Verified | `internal/graph/mem.go` (`Validate`) | `internal/graph/graph_test.go` | 2026-05-05 | All four `ErrNotWellFormed` failure modes exercised. |
| [UC-S02](system/UC-S02-apply-dpo-rewrite.md) | Apply a DPO rewrite | Verified | `internal/revision/engine.go` (`Apply`) | `internal/revision/revision_test.go` | 2026-06-10 | Add-vertex, add-edge, delete-vertex paths; ErrNoMatch, ErrSideConditionFailed, ErrNotWellFormed failure paths exercised. Strict bridge: ErrDanglingEdge (delete-side pushout-complement, step 1) and ErrIdentityCollision (produce-side content-addressing, step 2) refuse unfaithful rewrites; Lenient drops/overwrites silently. Idempotent re-statement and delete-then-add success variations covered. |
| [UC-S03](system/UC-S03-compute-pushout.md) | Compute the guarded pushout of two frontiers | Verified | `internal/composition/engine.go` (`Merge`) | `internal/composition/composition_test.go`, `internal/composition/composition_props_test.go` | 2026-06-10 | Union frontier with governance gate; Sat → Certificate via verification.Certify; Unsat → Policy-kind Conflict. Identical-frontiers and empty-policy paths covered. Property tests assert the set-union law, frontier commutativity/idempotence, witness determinism, and the merged-xor-conflicted invariant (both directions) across 300 random frontier pairs. |
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
| [UC-S15](system/UC-S15-bind-name.md) | Bind a name to a vertex | Verified | `internal/namespace/mem.go` (`BindRef`/`BindAlias`/`BindProjection`) | `internal/namespace/namespace_test.go` | 2026-05-05 | All three name kinds exercised; rebind-overwrite path covered. |
| [UC-S16](system/UC-S16-resolve-binding.md) | Resolve a name binding | Verified | `internal/namespace/mem.go` (`Resolve*`) | `internal/namespace/namespace_test.go` | 2026-05-05 | Bound + unbound paths covered. |
| [UC-S17](system/UC-S17-compute-content-id.md) | Compute a content-addressed identifier | Verified | `internal/identity/sha256.go` | `internal/identity/identity_test.go` | 2026-05-05 | SHA-256-backed factory; canonical-bytes contract covered. |
| [UC-S18](system/UC-S18-check-ontology-admissibility.md) | Check whether an edge or hyperedge is admissible | Verified | `internal/ontology/schema.go` | `internal/ontology/schema_test.go` | 2026-05-05 | Edge and hyperedge admissibility tables exercised. |
| [UC-S19](system/UC-S19-check-replay-feasibility.md) | Check whether a change capsule is replayable | Verified | `internal/revision/engine.go` (`Replayable`) | `internal/revision/revision_test.go` | 2026-05-05 | Happy path, empty capsule, consumed-missing, produced-missing failure paths covered. |
| [UC-S20](system/UC-S20-check-temporal-validity.md) | Check the temporal validity of a vertex | Verified | `internal/temporal/engine.go` (`Validity`) | `internal/temporal/temporal_test.go` | 2026-05-05 | Main path, malformed-triple and unknown-vertex failure paths covered. |

## Summary

As of 2026-06-10:

| Layer | Specified | Partial | Implemented | Verified | Retired | Total |
|---|---:|---:|---:|---:|---:|---:|
| User | 0 | 0 | 0 | 19 | 0 | 19 |
| System | 0 | 0 | 0 | 20 | 0 | 20 |
| **Total** | **0** | **0** | **0** | **39** | **0** | **39** |

**Verified coverage: 39 / 39 = 100%.** UC-U18 (three-way merge) and
UC-U19 (command-line operation, the `cmd/got` shell) added 2026-06-10.
All roadmap phases complete (see `roadmap.md`). Every public method on every internal `Engine` and
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
