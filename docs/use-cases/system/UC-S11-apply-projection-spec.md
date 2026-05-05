# UC-S11: Apply a full projection spec

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/projection` |
| Primary actor | `projection.Engine` |
| Stakeholders & interests | Caller: a closed `View` derived from the graph by the spec. Auditor: idempotent application — `Project ∘ Project = Project`. |
| Preconditions | A `Spec` is supplied. |
| Trigger | A higher-level flow needs the full projected view. |
| Success postcondition | A `View` is returned wrapping a `graph.Subgraph`. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System invokes `spec.Apply(g)` to obtain the subgraph.
2. System wraps the subgraph in a `View`.
3. System returns the view.

## Extensions

### Successful variations

- **1a. Spec already applied (cached view in namespace store):**
  - 1a1. System returns the cached view.
- **1b. Spec yields the empty subgraph:**
  - 1b1. System returns an empty `View` (no error).

### Failure paths

- **1c. Spec invalid:**
  - 1c1. System returns `projection.ErrInvalidSelector`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Spec evaluation strategy:** eager (full subgraph computed) or lazy (subgraph computed on access).

## Related use cases

- Includes: UC-S10 (Select frontier) when the spec is selector-driven.
- Included by: UC-U06 (Materialize bundle).
