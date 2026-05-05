# UC-U09: Resolve a name to a vertex

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `namespace.Store` |
| Primary actor | Reader / tool |
| Stakeholders & interests | Reader: deterministic resolution at a moment in time. Operator: resolution is a pure read against the store. |
| Preconditions | A `RefName`, `Alias`, or `ProjectionHandle` is supplied. |
| Trigger | Reader needs to dereference a name. |
| Success postcondition | The bound `identity.VertexID` is returned together with `true`. |
| Failure postcondition | The zero `VertexID` is returned together with `false` (idiomatic Go absent-value pattern; not an error). |

## Main success scenario

1. Actor invokes the appropriate `Resolve*` method on `namespace.Store` (UC-S16).
2. System looks up the binding in its underlying map or backing store.
3. System returns `(VertexID, true)`.

## Extensions

### Successful variations

- **1a. Caller resolves multiple kinds (`RefName` and `Alias` of the same string):**
  - 1a1. System treats the kinds as orthogonal namespaces — the caller queries each independently.

### Failure paths

- **2a. Name not bound:**
  - 2a1. System returns `(zero VertexID, false)`. This is the canonical absent path; no error is raised.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()` via the implementation's typical short-circuit. (For in-memory stores, cancellation is best-effort.)

## Sub-variations

- **Name kind:** `RefName` (mutable branch), `Alias` (release tag), `ProjectionHandle` (stored projection spec).
- **Backend:** in-memory `namespace.NewStore`, future persistent backends.

## Related use cases

- Includes: UC-S16 (Resolve a name binding).
- Related: UC-U03 (Create or update a branch), UC-U07 (Promote release).
