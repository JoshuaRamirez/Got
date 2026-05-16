# Architecture overview

A top-down read of the system. For the canonical statement of what the
system does, see `docs/requirements/use-cases/`. For working rules on
how to extend the code, see `docs/design-rules.md`. This document
explains how the pieces fit together.

## One-paragraph mental model

The repository is a **typed, attributed, content-addressed hypergraph**
with a mutable namespace shell on top. Everything observable — content,
revisions, agents, executions, evaluations, policies, claims — is a
vertex. Relationships are edges or hyperedges. Identity is the SHA-256
hash of a canonical byte encoding, so structurally equal objects always
share an ID. The graph is **append-only**: every write returns a new
graph value; the namespace is the single mutable component. On top of
that core, a set of `Engine` and `Service` types compose into a
`repo.Service` facade that drives end-to-end operations: ingest, revise,
branch, merge, evaluate, materialize, release.

## Dependency layers

```
                     identity   ontology              ← Layer 0: pure values
                         │           │
                  ┌──────┤   ┌───────┘
                  │      │   │
              namespace  │ graph                       ← Layer 0: core abstractions
                  │      │   │
                  │      ├───┤
                  │      │   │
                  │  provenance
                  │
                  │     multiagent  temporal  projection  revision
                  │                              │          │
                  │     governance           realization    │
                  │         │                                │
                  │     verification                         │
                  │         │                                │
                  │  ┌──────┼──────┬──────┐                  │
                  │  │      │      │      │                  │
                  │ replay capability composition           │
                  │                          │              │
                  └──────┬───────────────────┴──────────────┴── release
                         │
                       repo (facade)                          ← Layer 4: top of stack
```

Critical path from leaves to facade: `identity` / `ontology` →
`graph` → `projection` → `governance` → `verification` →
`composition` → `repo`. Five sequential implementations.

Every package's `package X` doc-comment declares its allowed imports.
The dependency graph is a strict DAG.

## Package roles

### Layer 0 — value types and core abstractions

- **`identity`** — content-addressed identifiers (`VertexID`, `EdgeID`,
  `HyperedgeID`) derived from SHA-256 hashes of canonical byte
  encodings. The leaf of the dependency graph.
- **`ontology`** — type system for vertices and edges, plus the
  admissibility schema that decides which edge-type triples are
  well-formed.
- **`namespace`** — the single mutable component. Maps `RefName`,
  `Alias`, `ProjectionHandle` names to `VertexID`s. The only interface
  whose methods take `context.Context` for I/O reasons (it may be backed
  by a remote store).
- **`graph`** — typed attributed hypergraph. Vertices, edges,
  hyperedges. Immutable: every `With*` returns a new graph value.
  `Builder` provides O(n) bulk construction; the streaming `With*` API
  is O(n²) per insert and meant for single-element work.
- **`provenance`** — closure operator over causal edges. `Cone`,
  `Close`, `Causes`, `TraceSet`. Treats causal edges as undirected
  because the admissibility table mixes directions.

### Layer 1 — single-purpose engines

- **`multiagent`** — authorship tracing. Walks `AuthoredBy` (and
  configurable other) edges to answer "who authored this?" and "what is
  the responsibility chain?"
- **`temporal`** — half-open validity intervals over vertex
  `TimeTriple.ValidFrom..ValidTo`. `Fresh(now)` is the membership test.
- **`projection`** — selectors and specs. `Engine.Select` wraps a
  selector's IDs in a `Frontier`; `Engine.Project` wraps a spec's
  subgraph in a `View`. Ships `IDsSelector` and `InduceSpec`.
- **`revision`** — DPO rewrite engine. `Apply` deletes consumed
  vertices/edges, retains context, inserts produced ones. `Replayable`
  checks vertex presence.
- **`governance`** — policy aggregation. `Check` runs the three-valued
  rule (Unsat dominates → Unknown → Sat). `GateRelease` requires Sat +
  empty obligations.
- **`realization`** — Target → Materializer registry. `ManifestTarget`
  emits one path per vertex; `JSONManifestTarget` emits a single
  manifest path covering everything.

### Layer 2 — verification

- **`verification`** — composes governance with a domain-supplied
  `Evaluator`. `Evaluate` dispatches to the Evaluator. `Prove` reads
  `Proves` / `Refutes` edges from the graph. `Certify` delegates the
  gate to governance and builds a `Certificate` on Sat. Ships
  `ScalarResult` and `WeightedAverageEvaluator`.

### Layer 3 — composed engines

- **`composition`** — merge as union frontier under governance gate.
  Witness ID is a deterministic SHA-256 of the union ID sequence. The
  spec describes a true categorical guarded pushout; the implementation
  is the simpler set-union interpretation. See "Spec / impl divergence"
  below.
- **`capability`** — emergence predicates evaluated in registration
  order; first match wins. Built-in predicates: `CertifiedNonEmpty`,
  `AllPoliciesNamed`.
- **`replay`** — wraps `revision`. Checks `Replayable` and
  `capsule.Environment ==/matches env.ID`. Does not re-execute the
  rewrite (capsule does not carry the Rule).
- **`release`** — alias lifecycle via `namespace.Store` plus an
  in-memory `(alias, version)` ledger for rollback.

### Layer 4 — facade

- **`repo`** — composes every engine and service. `DefaultState`
  bundles a graph and namespace. Methods are thin orchestration over
  the lower layers: Ingest dispatches by `Payload` kind; Revise
  delegates to revision; Branch checks then binds; Merge to
  composition; Evaluate to verification; Materialize chains
  projection→realization; Release to governance.

## Key design rules

The rules of record are in `docs/design-rules.md`. The three that show
up most often when reading code:

1. **`context.Context` first parameter** on every `Engine`/`Service`
   method. Skipped on pure value-type accessors (`Graph`, `Subgraph`,
   `Frontier`, `View`), on `identity.Hasher`/`Factory`, on
   `ontology.Schema`/`Registry`, and on `governance.Policy.Check`.
   `namespace.Store` is the named exception that gets `ctx` everywhere.

2. **Sentinel errors at package scope** wrapped via `fmt.Errorf("%w:
   ...", ErrX, detail)`. Callers use `errors.Is`.

3. **Single-getter data holders are structs.** Multi-method or
   opaque-computation types are interfaces. `provenance.Trace`,
   `replay.Outcome`, `realization.FidelityContract`,
   `capability.Witness`, `composition.MergeWitness`,
   `multiagent.Responsibility` are all structs by this rule.

## Requirements traceability

Every public method on every `Engine` and `Service` is reachable from at
least one user use case (sea level, `docs/requirements/use-cases/user/`)
and exercised by at least one system use case (fish level,
`docs/requirements/use-cases/system/`). The ledger at
`docs/requirements/use-cases/ledger.md` records which UCs are
`Verified` (implementation + behavioral tests). The roadmap at
`docs/requirements/use-cases/roadmap.md` orders the dependency layers.

As of `aa23eaf` and through the hardening passes, all 37 UCs read
`Verified`.

## Spec / impl divergence

The system carries categorical pretensions (Kleisli morphisms,
pushouts, sheaves, closure operators). The current implementations are
the simplest faithful interpretations that satisfy the UC contracts but
are not full categorical mechanisms:

- **`composition.Merge`** is set-union under a governance gate, not a
  pushout in the policy subcategory. UC failure paths referring to
  "pushout complement does not exist" are unreachable.
- **`revision.Apply`** is "delete L\K, keep K, add R\K" using IDs
  declared by the Rule, not the construction of a pushout complement
  from scratch.
- **`provenance.Close`** treats causal edges as undirected. The
  axioms (extensive, monotone, idempotent) still hold, but a future
  directional interpretation would yield different traces.
- **`replay.Replay`** confirms structural feasibility and environment
  match; it does not re-execute the rewrite because the capsule does
  not carry the Rule.

These simplifications are intentional and isolated to the package
implementations. The UC specs and interface contracts remain the
target for a future categorical implementation.

## Performance characteristics

Measured at n=1000 vertices on a single CPU:

| Operation | Cost | Notes |
|---|---|---|
| `graph.WithVertex` (repeated) | O(n²) total | Each call copies the vertices map. |
| `graph.Builder.AddVertex` + `Build` | O(n) total | ~100x faster at n=1000. |
| `graph.Graph.Validate` | O(E+H) | E edges, H hyperedges. |
| `graph.Graph.Induce(k)` | O(E+H) | Linear scan to filter. |
| `provenance.Close` | O(V+E) | BFS over causal adjacency. |
| `provenance.Causes` | O(V+E) worst | Short-circuits on reach. |

The streaming `With*` API is fine for one-shot operations; bulk
construction should use `Builder`. `repo.Service.Ingest` currently uses
the streaming API to preserve error-reporting order; it could migrate
to `Builder` later if bulk performance matters there.

## Concurrency

The immutable `graph.Graph` is safe for concurrent reads (tested under
`-race` with 32 readers). The `graph.Builder` is single-writer — do
not call `Add*` from multiple goroutines without external
synchronization. The `namespace.Store` interface takes `context.Context`
because remote backings are anticipated; the default `mem.go`
implementation is not safe for concurrent writes (an in-memory map
without a mutex). Wrap with a mutex if you need concurrent writers.

## Where to read next

- New to the code? Start with `internal/graph/graph.go` then
  `internal/repo/repo.go` for the surface, then any UC in
  `docs/requirements/use-cases/user/` for an end-to-end example.
- Extending the code? Read `internal/CLAUDE.md` and
  `docs/design-rules.md`.
- Planning the next package? Read `docs/requirements/use-cases/roadmap.md`.
- Recording a decision? Use `/devlog append` and the convention in
  `docs/devlog/CLAUDE.md`.
