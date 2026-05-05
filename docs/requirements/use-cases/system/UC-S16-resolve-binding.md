# UC-S16: Resolve a name binding

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/namespace` |
| Primary actor | `namespace.Store` |
| Stakeholders & interests | Caller: deterministic resolution at this moment. |
| Preconditions | A name is supplied. |
| Trigger | A higher-level read flow needs to dereference a name. |
| Success postcondition | `(VertexID, true)` is returned. |
| Failure postcondition | `(zero VertexID, false)` is returned (idiomatic absent-value, not an error), or `ctx.Err()`. |

## Main success scenario

1. System invokes the corresponding `Resolve*` method on the store.
2. System looks up the binding.
3. System returns `(VertexID, true)`.

## Extensions

### Successful variations

- None beyond the basic lookup.

### Failure paths

- **2a. Name not bound:**
  - 2a1. System returns `(zero, false)`. No error.
- **\*. `ctx` cancelled:**
  - System returns the implementation's typical short-circuit; for in-memory stores cancellation is best-effort.

## Sub-variations

- **Name kind:** `RefName`, `Alias`, `ProjectionHandle`.

## Related use cases

- Included by: UC-U09 (Resolve a name to a vertex).
