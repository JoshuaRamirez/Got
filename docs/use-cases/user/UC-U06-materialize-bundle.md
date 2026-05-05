# UC-U06: Materialize a bundle from a projection

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo.Service` |
| Primary actor | Build system / Consumer |
| Stakeholders & interests | Consumer: receive a bundle in the requested target format. Auditor: every path in the bundle has a provenance witness inside the projected subgraph. Operator: materialization does not mutate the graph. |
| Preconditions | The `projection.Spec` is well-formed against `state.Graph()`. A `realization.Target` is supplied and registered. |
| Trigger | Consumer asks for a deliverable view of the repository for a target format. |
| Success postcondition | A `realization.Bundle` is returned. Each `Bundle.Provenance(path)` lies inside the projected subgraph's vertex IDs. The `State` is unchanged. |
| Failure postcondition | No bundle is returned. An error is reported. |

## Main success scenario

1. Actor invokes `repo.Service.Materialize(ctx, state, spec, target)`.
2. System applies the projection spec to obtain a `View` (UC-S11).
3. System invokes `realization.Engine.Materialize(ctx, view, target)` (UC-S14).
4. System verifies for each `path in Bundle.Paths()` that `Bundle.Provenance(path)` is a subset of the view's vertex IDs.
5. System returns the bundle.

## Extensions

### Successful variations

- **2a. Spec is already cached as a `View` in the namespace store:**
  - 2a1. System reuses the cached view and proceeds to step 3.
- **3a. Target supports streaming:**
  - 3a1. Bundle exposes paths incrementally; each is verified at retrieval time per step 4.

### Failure paths

- **2b. Spec invalid against current graph:**
  - 2b1. System returns `projection.ErrInvalidSelector`.
- **3b. Target has no registered materializer:**
  - 3b1. System returns `realization.ErrTargetUnsupported`.
- **4a. A bundle path's provenance escapes the projected subgraph (fidelity violated):**
  - 4a1. System returns an error wrapping the offending path. The bundle is discarded.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Target format:** any `realization.Target` with a registered materializer.
- **Fidelity contract:** lossless, lossy-with-witness, etc. — carried on the returned bundle.

## Related use cases

- Includes: UC-S11 (Apply projection spec), UC-S14 (Materialize for target).
