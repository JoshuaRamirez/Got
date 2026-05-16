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
| [UC-U01](user/UC-U01-ingest-content.md) | Ingest content into repository | Specified | `internal/repo` (interface) | — | 2026-05-05 | `repo.Service.Ingest` is interface-only. |
| [UC-U02](user/UC-U02-revise-graph.md) | Revise the graph via a rewrite rule | Specified | `internal/repo` (interface) | — | 2026-05-05 | Awaits `repo.Service` and `revision.Engine` impls. |
| [UC-U03](user/UC-U03-create-or-update-branch.md) | Create or update a branch | Partial | `internal/namespace/mem.go` (binding side) | `internal/namespace/namespace_test.go` | 2026-05-05 | Underlying `BindRef` is verified; the user-facing facade `repo.Service.Branch` is interface-only. |
| [UC-U04](user/UC-U04-merge-frontiers.md) | Merge two frontiers | Specified | `internal/repo` (interface) | — | 2026-05-05 | Awaits `repo.Service` and `composition.Engine` impls. |
| [UC-U05](user/UC-U05-evaluate-frontier.md) | Evaluate a frontier in an environment | Specified | `internal/repo` (interface) | — | 2026-05-05 | Awaits `repo.Service` and `verification.Engine` impls. |
| [UC-U06](user/UC-U06-materialize-bundle.md) | Materialize a bundle from a projection | Specified | `internal/repo` (interface) | — | 2026-05-05 | Awaits `repo`, `projection`, `realization` impls. |
| [UC-U07](user/UC-U07-promote-release.md) | Promote a frontier to a release alias | Specified | `internal/release` (interface) | — | 2026-05-05 | Awaits `release.Service` and `governance.Engine` impls. |
| [UC-U08](user/UC-U08-rollback-release.md) | Rollback a release alias | Specified | `internal/release` (interface) | — | 2026-05-05 | Awaits `release.Service` impl + release ledger storage. |
| [UC-U09](user/UC-U09-resolve-name.md) | Resolve a name to a vertex | Verified | `internal/namespace/mem.go` | `internal/namespace/namespace_test.go` | 2026-05-05 | Main path + unbound-name failure path covered. |
| [UC-U10](user/UC-U10-query-graph.md) | Query the graph | Verified | `internal/graph/mem.go` | `internal/graph/graph_test.go` | 2026-05-05 | `Vertex`/`Edge`/`Hyperedge`/`VertexIDs`/`Induce` covered; `Query` returns `ErrQueryUnsupported` (covered). |
| [UC-U11](user/UC-U11-trace-provenance.md) | Trace causal provenance | Verified | `internal/provenance/engine.go` | `internal/provenance/provenance_test.go` | 2026-05-05 | All four read methods covered including reflexivity, monotonicity, idempotence axioms. |
| [UC-U12](user/UC-U12-trace-authorship.md) | Trace authorship and responsibility | Verified | `internal/multiagent/engine.go` | `internal/multiagent/multiagent_test.go` | 2026-05-05 | Authorship and ResponsibilityTrace covered; ErrNoAuthorship and graph.ErrVertexNotFound failure paths exercised. |
| [UC-U13](user/UC-U13-check-freshness.md) | Check temporal freshness of a vertex | Verified | `internal/temporal/engine.go` | `internal/temporal/temporal_test.go` | 2026-05-05 | Validity, Fresh half-open semantics, indefinite-`ValidTo`, malformed triple, and unknown-vertex paths covered. |
| [UC-U14](user/UC-U14-replay-capsule.md) | Replay a change capsule | Specified | `internal/replay` (interface) | — | 2026-05-05 | Awaits `replay.Engine` and `revision.Engine` impls. |
| [UC-U15](user/UC-U15-prove-claim.md) | Prove a claim with a proof | Specified | `internal/verification` (interface) | — | 2026-05-05 | Awaits `verification.Engine` impl. |
| [UC-U16](user/UC-U16-detect-emergent-capability.md) | Detect an emergent capability | Specified | `internal/capability` (interface) | — | 2026-05-05 | Awaits `capability.Engine` impl. |
| [UC-U17](user/UC-U17-resolve-merge-conflicts.md) | Resolve merge conflicts | Specified | `internal/composition` (interface) | — | 2026-05-05 | Awaits `composition.Engine.Resolve` impl. |

## System use cases

| ID | Title | Status | Implementation | Tests | Last reviewed | Notes |
|---|---|---|---|---|---|---|
| [UC-S01](system/UC-S01-validate-graph.md) | Validate graph well-formedness | Verified | `internal/graph/mem.go` (`Validate`) | `internal/graph/graph_test.go` | 2026-05-05 | All four `ErrNotWellFormed` failure modes exercised. |
| [UC-S02](system/UC-S02-apply-dpo-rewrite.md) | Apply a DPO rewrite | Verified | `internal/revision/engine.go` (`Apply`) | `internal/revision/revision_test.go` | 2026-05-05 | Add-vertex, add-edge, delete-vertex paths; ErrNoMatch and ErrSideConditionFailed failure paths exercised. |
| [UC-S03](system/UC-S03-compute-pushout.md) | Compute the guarded pushout of two frontiers | Specified | `internal/composition` (interface) | — | 2026-05-05 | Awaits `composition.Engine.Merge` impl. |
| [UC-S04](system/UC-S04-resolve-conflicts.md) | Apply conflict resolutions | Specified | `internal/composition` (interface) | — | 2026-05-05 | Awaits `composition.Engine.Resolve` impl. |
| [UC-S05](system/UC-S05-evaluate-in-environment.md) | Evaluate a frontier in a given environment | Specified | `internal/verification` (interface) | — | 2026-05-05 | Awaits `verification.Engine.Evaluate` impl. |
| [UC-S06](system/UC-S06-issue-certificate.md) | Issue a certificate for a frontier | Specified | `internal/verification` (interface) | — | 2026-05-05 | Awaits `verification.Engine.Certify` impl. |
| [UC-S07](system/UC-S07-compute-provenance-closure.md) | Compute the provenance closure of a seed set | Verified | `internal/provenance/engine.go` (`Close`) | `internal/provenance/provenance_test.go` | 2026-05-05 | Extensivity, monotonicity, idempotence axioms tested. |
| [UC-S08](system/UC-S08-compute-provenance-cone.md) | Compute the provenance cone of a vertex | Verified | `internal/provenance/engine.go` (`Cone`) | `internal/provenance/provenance_test.go` | 2026-05-05 | `Cone == Close({seed})` axiom tested. |
| [UC-S09](system/UC-S09-enumerate-causal-traces.md) | Enumerate causal traces between two vertices | Verified | `internal/provenance/engine.go` (`TraceSet`) | `internal/provenance/provenance_test.go` | 2026-05-05 | Simple-path enumeration verified. |
| [UC-S10](system/UC-S10-select-frontier.md) | Select a frontier from the graph | Verified | `internal/projection/engine.go` (`Select`, `IDsSelector`) | `internal/projection/projection_test.go` | 2026-05-05 | Main path, empty selector, ErrInvalidSelector failure paths, ctx cancellation covered. |
| [UC-S11](system/UC-S11-apply-projection-spec.md) | Apply a full projection spec | Verified | `internal/projection/engine.go` (`Project`, `InduceSpec`) | `internal/projection/projection_test.go` | 2026-05-05 | Main path + spec-error failure path covered. |
| [UC-S12](system/UC-S12-check-policy-aggregate.md) | Check the aggregate decision over a policy set | Specified | `internal/governance` (interface) | — | 2026-05-05 | Awaits `governance.Engine.Check` impl. |
| [UC-S13](system/UC-S13-gate-release.md) | Gate a frontier for release | Specified | `internal/governance` (interface) | — | 2026-05-05 | Awaits `governance.Engine.GateRelease` impl. |
| [UC-S14](system/UC-S14-materialize-for-target.md) | Materialize a view for a specific target | Specified | `internal/realization` (interface) | — | 2026-05-05 | Awaits `realization.Engine.Materialize` impl. |
| [UC-S15](system/UC-S15-bind-name.md) | Bind a name to a vertex | Verified | `internal/namespace/mem.go` (`BindRef`/`BindAlias`/`BindProjection`) | `internal/namespace/namespace_test.go` | 2026-05-05 | All three name kinds exercised; rebind-overwrite path covered. |
| [UC-S16](system/UC-S16-resolve-binding.md) | Resolve a name binding | Verified | `internal/namespace/mem.go` (`Resolve*`) | `internal/namespace/namespace_test.go` | 2026-05-05 | Bound + unbound paths covered. |
| [UC-S17](system/UC-S17-compute-content-id.md) | Compute a content-addressed identifier | Verified | `internal/identity/sha256.go` | `internal/identity/identity_test.go` | 2026-05-05 | SHA-256-backed factory; canonical-bytes contract covered. |
| [UC-S18](system/UC-S18-check-ontology-admissibility.md) | Check whether an edge or hyperedge is admissible | Verified | `internal/ontology/schema.go` | `internal/ontology/schema_test.go` | 2026-05-05 | Edge and hyperedge admissibility tables exercised. |
| [UC-S19](system/UC-S19-check-replay-feasibility.md) | Check whether a change capsule is replayable | Verified | `internal/revision/engine.go` (`Replayable`) | `internal/revision/revision_test.go` | 2026-05-05 | Happy path, empty capsule, consumed-missing, produced-missing failure paths covered. |
| [UC-S20](system/UC-S20-check-temporal-validity.md) | Check the temporal validity of a vertex | Verified | `internal/temporal/engine.go` (`Validity`) | `internal/temporal/temporal_test.go` | 2026-05-05 | Main path, malformed-triple and unknown-vertex failure paths covered. |

## Summary

As of 2026-05-05:

| Layer | Specified | Partial | Implemented | Verified | Retired | Total |
|---|---:|---:|---:|---:|---:|---:|
| User | 11 | 1 | 0 | 5 | 0 | 17 |
| System | 7 | 0 | 0 | 13 | 0 | 20 |
| **Total** | **18** | **1** | **0** | **18** | **0** | **37** |

Verified coverage: 18 / 37 ≈ 49%. Phase 1A complete (see `roadmap.md`):
`projection`, `revision`, `temporal`, `multiagent` are all implemented
and tested. Active phase advances to Phase 1B (`governance`,
`realization`).

## Next-bite candidates

Per `roadmap.md`, the active phase is now **Phase 1B**:

1. **`governance.Engine`** — verifies UC-S12 and UC-S13. On the critical
   path to `verification` → `composition` → `repo`.
2. **`realization.Engine`** — verifies UC-S14. Needed by `repo` but off
   the critical path.

Both depend on `projection` (Verified in Phase 1A) and can be implemented
in parallel.
