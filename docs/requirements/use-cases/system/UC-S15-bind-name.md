# UC-S15: Bind a name to a vertex

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/namespace` |
| Primary actor | `namespace.Store` |
| Stakeholders & interests | Caller: subsequent `Resolve*` returns the bound vertex. Operator: the graph is unchanged; only the namespace state mutates. |
| Preconditions | A name (`RefName` / `Alias` / `ProjectionHandle`) and a target `VertexID` are supplied. |
| Trigger | A higher-level write flow needs to attach a mutable name. |
| Success postcondition | `Resolve*(name) == (target, true)` after this call. |
| Failure postcondition | The store is unchanged; an error is returned. |

## Main success scenario

1. System invokes the corresponding `Bind*` method on the store with the supplied name and target.
2. System persists the binding (in-memory map for default backend; remote write for persistent backends).
3. System returns nil.

## Extensions

### Successful variations

- **1a. Name already bound to the same target:**
  - 1a1. System completes as a no-op success.
- **1b. Name already bound to a different target:**
  - 1b1. System overwrites; previous binding is not retained.

### Failure paths

- **2a. Backing store rejects the write (transient):**
  - 2a1. System returns the store's error unchanged.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Name kind:** `RefName`, `Alias`, `ProjectionHandle`.
- **Backend:** in-memory or persistent.

## Related use cases

- Included by: UC-U03 (Branch), UC-U07 (Promote release), UC-U08 (Rollback release).
