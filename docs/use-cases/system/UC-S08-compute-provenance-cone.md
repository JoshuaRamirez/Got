# UC-S08: Compute the provenance cone of a vertex

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/provenance` |
| Primary actor | `provenance.Engine` |
| Stakeholders & interests | Auditor: full causal cone of a single vertex. Caller: result equals `Close({v})`. |
| Preconditions | The seed vertex exists in the graph. |
| Trigger | A higher-level flow needs the cone of a single vertex. |
| Success postcondition | A vertex slice equal to `Close(g, {seed})` is returned. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System delegates to `Close(ctx, g, []VertexID{seed})` (UC-S07).
2. System returns the resulting slice.

## Extensions

### Successful variations

- **1a. Seed vertex isolated (no causal neighbors):**
  - 1a1. System returns `[seed]`.

### Failure paths

- **1b. Seed not in graph:**
  - 1b1. System returns `provenance.ErrUnknownVertex`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- None beyond UC-S07.

## Related use cases

- Equivalent (by axiom): UC-S07 with singleton seed.
- Included by: UC-U11 (Trace provenance).
