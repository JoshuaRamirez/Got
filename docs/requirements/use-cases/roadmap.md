# UC roadmap

The roadmap shows the optimal dependency-ordered implementation chain for
moving every UC from `Specified` to `Verified`. Combined with `ledger.md`,
these two files encode the plan at any moment without further discussion:

- `ledger.md` says **what is done** and **where each UC stands**.
- `roadmap.md` says **what to do next** and **why that order**.

A new contributor (human or agent) reads the ledger to find the current
state, then reads this roadmap to find the next package to implement.

## Goal

Move all 37 UCs to `Verified`. `Verified` requires concrete
implementation plus behavioral tests covering the main success path and
at least one failure path per extension group (per `docs/design-rules.md`).

The order below is determined by:
- The dependency graph extracted from each package's actual imports.
- The number of UCs each impl unlocks.
- Independence — packages in the same phase have no dependency on each
  other and can be implemented in parallel by separate workstreams.

## Dependency graph

```
                                                        Phase 0
                identity   ontology                     (DONE)
                    │           │
              ┌─────┤   ┌───────┘
              │     │   │
          namespace │ graph
              │     │   │
              │     ├───┤
              │     │   │
              │  provenance
              │
              │    Phase 1A — parallel-safe leaves
              │    ┌────────────┬──────────┬──────────┐
              │    │            │          │          │
              │  multiagent  temporal  projection  revision
              │                            │          │
              │     Phase 1B               │          │
              │     ┌──────────────────────┤          │
              │     │                      │          │
              │  governance            realization    │
              │     │                                 │
              │     │   Phase 2                       │
              │     │   verification                  │
              │     │      │                          │
              │     │      │   Phase 3                │
              │     │      ├────────┬──────┬──────────┤
              │     │      │        │      │          │
              │     │   replay  capability composition │
              │     │                         │       │
              └─────┴─────────────────────────┴───────┴──── release
                                                  │       (also Phase 3)
                                                  │
                                            Phase 4
                                              repo
```

Critical path (longest sequential chain to fully verify): `projection →
governance → verification → composition → repo`. Five packages.

## Phases

### Phase 0 — Done

| Package | UCs Verified |
|---|---|
| `identity` | UC-S17 |
| `ontology` | UC-S18 |
| `namespace` | UC-S15, UC-S16, UC-U09 |
| `graph` | UC-S01, UC-U10 |
| `provenance` | UC-S07, UC-S08, UC-S09, UC-U11 |

**Cumulative**: 11 / 37 Verified.

### Phase 1A — Done

| Package | UCs Verified |
|---|---|
| `multiagent` | UC-U12 |
| `temporal` | UC-U13, UC-S20 |
| `projection` | UC-S10, UC-S11 |
| `revision` | UC-S02, UC-S19 |

**Cumulative**: 18 / 37 Verified (+7 from Phase 0).

### Phase 1B — Done

| Package | UCs Verified |
|---|---|
| `governance` | UC-S12, UC-S13 |
| `realization` | UC-S14 |

**Cumulative**: 21 / 37 Verified (+3 from Phase 1A).

### Phase 2 — Done

| Package | UCs Verified |
|---|---|
| `verification` | UC-S05, UC-S06, UC-U15 |

**Cumulative**: 24 / 37 Verified (+3 from Phase 1B).

UC-U05, UC-U04, UC-U06, UC-U17 are still `Specified` — they route
through `repo.Service` which lands in Phase 4.

### Phase 3 — Done

| Package | UCs Verified |
|---|---|
| `replay` | UC-U14 |
| `capability` | UC-U16 |
| `composition` | UC-S03, UC-S04 |
| `release` | UC-U07, UC-U08 |

**Cumulative**: 30 / 37 Verified (+6 from Phase 2). All system-level UCs
are Verified at this point.

UC-U04 and UC-U17 stay `Specified` until Phase 4 lands `repo`.

### Phase 4 — Top of stack

| Package | Deps | UCs unlocked |
|---|---|---|
| `repo` | composition, governance, graph, identity, namespace, projection, realization, revision, verification | UC-U01, UC-U02, UC-U03 (full), UC-U04, UC-U05, UC-U06, UC-U17 |

**Cumulative**: 37 / 37 Verified (+7). Roadmap complete.

## Verification cumulative chart

| After phase | Verified | New | Total |
|---|---:|---:|---:|
| Phase 0 | 11 | — | 37 |
| Phase 1A | 18 | +7 | 37 |
| Phase 1B | 21 | +3 | 37 |
| Phase 2 | 24 | +3 | 37 |
| Phase 3 | 30 | +6 | 37 |
| Phase 4 | 37 | +7 | 37 |

## Current focus

**Active phase: Phase 4.** Phases 0–3 are all complete. All system-level
UCs are Verified. The single package remaining is the `repo` facade,
which lands the seven user-level UCs routed through `repo.Service`:

1. **`repo`** — UC-U01 (Ingest), UC-U02 (Revise), UC-U03 full (Branch),
   UC-U04 (Merge), UC-U05 (Evaluate), UC-U06 (Materialize), UC-U17
   (Resolve conflicts).

All dependencies are Verified. Implementing `repo` completes the
roadmap.

## Update protocol

This file changes when **either** the dependency graph changes (new
package added, deps revised) **or** a phase boundary is crossed.

1. When a package's UCs all move to `Verified` in the ledger, update the
   "Current focus" section to reflect the next active phase.
2. When all packages in a phase are `Verified`, move that phase block
   above "Active phase" and update the "Cumulative" line in lower phases.
3. If a new package is added under `internal/`, locate its phase by
   import depth and insert it. Update the dependency graph diagram.
4. Do not re-order phases without a corresponding architecture change.
   The order is determined by imports, not preference.

The `/use-case roadmap` slash command renders this file. The
`/use-case next` subcommand cross-references the ledger to suggest the
single next-best package to pick up.
