# Design rules

This document records decisions that govern API shape across the `internal/`
packages. Apply them to new code and to refactors of existing code.

## 1. `context.Context`

Add `ctx context.Context` as the first parameter of every interface method on
`Engine` and `Service` types — these are orchestration surfaces that may run
arbitrarily long, perform I/O, or compose cancellable work.

Do **not** add `ctx` to:
- Pure value-type accessors. `Graph`, `Subgraph`, `Frontier`, `View`, `Trace`,
  `Bundle`, `Certificate`, `Evaluation`, `Match`, `Rule`, `Predicate`,
  `Schema`, `Registry`, `Selector`, `Spec`, and the various
  `*Witness`/`*Result` getters expose data, not work.
- `identity.Hasher.Sum` / `identity.Factory.*ID` — these are pure deterministic
  computations with no I/O surface to cancel.
- `governance.Policy.Check` — `Policy` is a value (the rule), not an Engine.
  The Engine method that *runs* policies takes `ctx` and is responsible for
  honoring it.

`namespace.Store` is an exception: it mutates state and the interface may be
backed by a remote/persistent store. All `namespace.Store` methods take `ctx`.

Implementations that don't actually consult `ctx` should still accept and
propagate it; do not add `_ context.Context` discards in interface signatures.

## 2. Error types

Each package that produces typed failure modes declares sentinel errors at
package scope. Wrap them with `fmt.Errorf("%w: ...", ErrX, detail)` to add
context while preserving `errors.Is` matchability.

Required sentinels (where applicable to the package's domain):
- `graph.ErrVertexNotFound`, `graph.ErrMissingEndpoint`,
  `graph.ErrNotWellFormed`, `graph.ErrQueryUnsupported` (already present)
- `governance.ErrPolicyViolation`, `governance.ErrUnknownDecision`
- `composition.ErrConflictUnresolvable`, `composition.ErrNoPushout`
- `verification.ErrCertificationFailed`, `verification.ErrEnvironmentMismatch`
- `revision.ErrSideConditionFailed`, `revision.ErrNoMatch`
- `realization.ErrTargetUnsupported`
- `release.ErrPolicyGate`, `release.ErrUnknownVersion`
- `replay.ErrNonDeterministic`
- `repo.ErrIngestRejected`
- `namespace.ErrUnknownName`
- `temporal.ErrUnknownVertex`
- `multiagent.ErrNoAuthorship`
- `capability.ErrNoEmergence`
- `projection.ErrInvalidSelector`

Add new sentinels as new failure modes appear. Avoid bare `errors.New(...)` at
the call site for failures the caller might want to react to.

## 3. Struct vs interface

Use a **struct** when the type is a pure data holder — its identity is its
fields, and there is no reasonable alternative implementation.

Use an **interface** when:
- the type has more than one method (i.e. real behavior, not a single getter), or
- there are real or planned alternative implementations (in-mem vs. persistent,
  eager vs. lazy, etc.), or
- the type wraps an opaque computation result whose internals callers should
  not depend on (e.g. `graph.Graph`, `graph.Subgraph`, `composition.Resolution`).

Single-getter "data holder" types are structs. The following are reclassified
from interfaces to structs under this rule:
- `provenance.Trace` (was `Vertices() []identity.VertexID` getter)
- `replay.Outcome` (was `Deterministic() bool` getter)
- `realization.FidelityContract` (was `Name() string` getter)
- `capability.Witness` (was `Name() string` getter)
- `composition.MergeWitness` (was `ID() identity.VertexID` getter)

Multi-method interfaces (`composition.Conflict` with `Kind`+`Boundary`,
`verification.Certificate` with three getters, etc.) stay as interfaces.

## 4. `repo.Service.Ingest` typing

Replace `Ingest(State, any) (State, error)` with a typed `Payload` interface:

```go
type Payload interface {
    PayloadKind() string
}
```

This mirrors the discriminator pattern already used by `graph.Query`. Concrete
payload types (`VertexPayload`, `EdgePayload`, etc.) implement `Payload` and
carry their own typed fields. The `any` escape hatch is removed.

## 5. Test gating

Tests follow implementations. CI's `go test -v -race` must stay green, so:
- Every package with a concrete implementation must have a test file that
  exercises the happy path and one error path per public API.
- Interface-only packages (those whose `*.go` declares only types and no
  implementations) are not required to ship a test file. When the first
  implementation lands, tests land with it in the same change.

After the current refactor, packages with concrete impls and tests are:
`graph`, `identity`, `namespace`, `ontology`, `provenance`. The other twelve
remain interface-only.
