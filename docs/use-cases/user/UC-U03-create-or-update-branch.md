# UC-U03: Create or update a branch

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo.Service`, `namespace.Store` |
| Primary actor | Author |
| Stakeholders & interests | Author: a stable name points at a chosen vertex. Reviewer: branch resolution is deterministic at any moment. Operator: the underlying graph is unchanged. |
| Preconditions | The target `identity.VertexID` exists in `state.Graph()`. |
| Trigger | Author wants to label a vertex with a moving name (branch). |
| Success postcondition | The namespace is updated so that `Store.ResolveRef(name)` returns the target vertex. The graph is unchanged. |
| Failure postcondition | The namespace is unchanged. An error is reported. |

## Main success scenario

1. Actor invokes `repo.Service.Branch(ctx, state, name, target)`.
2. System verifies `target` is a vertex in `state.Graph()`.
3. System binds the name to the target via `namespace.Store.BindRef` (UC-S15).
4. System returns a new `State` whose namespace component reflects the new binding. The graph component is the same instance.

## Extensions

### Successful variations

- **3a. Name already bound to a different vertex:**
  - 3a1. System overwrites the prior binding (refs are mutable). The previous mapping is not retained.
- **3b. Name already bound to the same vertex:**
  - 3b1. System completes as a no-op success.

### Failure paths

- **2c. Target vertex not in graph:**
  - 2c1. System returns `graph.ErrVertexNotFound` wrapped with the supplied target ID.
- **3c. Underlying store rejects the bind (e.g. persistent backend transient failure):**
  - 3c1. System returns the store's error unchanged.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Name kind:** `RefName` (this UC), `Alias` (UC-U07), `ProjectionHandle` (separate flow inside `repo.Service.Materialize`).
- **Backend:** in-memory `namespace.NewStore`, future persistent backends.

## Related use cases

- Includes: UC-S15 (Bind name).
- Related: UC-U09 (Resolve a name to a vertex).
