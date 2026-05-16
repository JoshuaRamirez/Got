# Changelog

Notable changes to the project. Grouped by PR-merge order on `main`.
For day-by-day session detail see `docs/devlog/`.

## Unreleased

### Added — hardening

- **`graph.Builder`** — O(n) bulk graph construction. ~100x faster than
  repeated `WithVertex` at n=1000. (#13)
- **Composability helpers** — `realization.JSONManifestTarget`,
  `capability.AllPoliciesNamed`, `verification.WeightedAverageEvaluator`. (#13)
- **Concurrency stress tests** — `graph.TestGraphConcurrentReads` and
  `TestGraphConcurrentInduce` under `go test -race`. (#13)
- **Fuzz tests** — `identity.FuzzVertexIDDeterminism`, `FuzzIDDistinctnessAcrossKinds`,
  `graph.FuzzWithVertexValidate`, `FuzzEmptyPreservesSchema`. (#12)
- **Benchmarks** — `graph.BenchmarkWithVertex_1000`, `BenchmarkValidate_1000`,
  `BenchmarkInduce_500of1000`, `provenance.BenchmarkClose_1000`,
  `BenchmarkCauses_endToEnd_1000`. (#12)
- **Failure-path test gaps filled** — `revision.TestApplyInsertViolatesSchema`
  (UC-S02 4a), `verification.TestCertifyUnknownBlocks` (UC-S06 2b). (#13)

### Added — roadmap completion (37/37 Verified)

- **Phase 4 — `repo`** — `DefaultService` composes every engine.
  `DefaultState` bundles graph + namespace. `VertexPayload` and
  `EdgePayload` provided for `Ingest`. Verifies UC-U01..UC-U06. (#11)
- **Phase 3 — `composition`, `replay`, `capability`, `release`** —
  composition merges via union under governance gate; replay validates
  capsule replayability + environment match; capability evaluates
  Predicate sequences; release manages alias lifecycle with in-memory
  ledger. Verifies UC-S03, UC-S04, UC-U14, UC-U16, UC-U07, UC-U08. (#10)
- **Phase 2 — `verification`** — composes governance and a domain
  Evaluator; `Prove` reads Proves/Refutes edges; `Certify` builds a
  Certificate when GateRelease passes. Verifies UC-S05, UC-S06, UC-U15. (#9)
- **Phase 1B — `governance`, `realization`** — governance aggregates
  per-policy decisions via the three-valued rule; realization is a
  Target → Materializer registry preloaded with `ManifestTarget`.
  Verifies UC-S12, UC-S13, UC-S14. (#8)
- **Phase 1A — `projection`, `temporal`, `multiagent`, `revision`** —
  four parallel-safe leaves. Projection wraps Selector/Spec; temporal
  reads TimeTriple; multiagent walks authorship edges; revision is a
  DPO rewrite engine. Verifies UC-S02, UC-S10, UC-S11, UC-S19, UC-S20,
  UC-U12, UC-U13. Also adds `graph.Graph.Empty()` so revision can
  rebuild without leaking schema. (#7)

### Added — process and structure

- **UC roadmap** with explicit phase ordering and the `/use-case
  roadmap` / `/use-case next` subcommands. (#6)
- **Per-folder CLAUDE.md** files for `internal/`, `docs/`,
  `.github/`, `docs/requirements/`, plus the project-level `librarian`,
  `requirements`, `devlog` skills under `.claude/skills/`. (#5)
- **UC ledger** tracking implementation/verification status of every UC.
  Root `CLAUDE.md` declares UCs as the project's primary requirements.
  `/use-case` skill landed. (#4)
- **Cockburn-style use case catalogue** — 17 user UCs, 20 system UCs,
  Cockburn template, index, convention. (#3)

### Added — design baseline

- **All P2 design questions resolved across 17 packages** — `context.Context`
  on Engine/Service methods, sentinel errors per package, single-getter
  data holders converted to structs, `repo.Service.Ingest(any)` replaced
  with typed `Payload`, every package has at least minimum tests, gofmt
  format check added to CI. (#0c2ea91-era squashed into PRs #2 and prior)
- **Devlog system** with one-file-per-UTC-day convention. (#earlier)
- **Modular per-folder CLAUDE.md** files for `internal/`, `docs/`,
  `.github/`. (#earlier)

### Spec / impl divergence notes

- `composition.Merge` is set-union under a governance gate, not a
  categorical guarded pushout. UC failure paths referring to
  pushout-complement failures are unreachable.
- `revision.Apply` deletes/adds via Rule-declared IDs rather than
  constructing a pushout complement from scratch.
- `provenance.Close` treats causal edges as undirected.
- `replay.Replay` confirms structural feasibility + environment match;
  does not re-execute the rewrite.

These simplifications are intentional; a future categorical
implementation would re-enable the now-unreachable UC failure paths
without changing the interfaces.
