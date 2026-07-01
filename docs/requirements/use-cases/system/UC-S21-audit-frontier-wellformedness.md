# UC-S21: Audit a frontier for structural and temporal well-formedness

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `composition.Engine` (via `composition.DefaultEngine.Audit`, the `Auditor` capability) |
| Primary actor | `composition.Engine` |
| Stakeholders & interests | `repo.Service.ReleaseStrict`: refuse to release a frontier that is not structurally/temporally well-formed. Integrator: catch a malformed frontier before it becomes a release, not only at merge time. |
| Preconditions | A host graph and a frontier over it are supplied. |
| Trigger | A caller (e.g. `repo.Service.ReleaseStrict`) asks for the in-graph audit of a frontier, independently of a merge. |
| Success postcondition | A (possibly empty) list of typed `Conflict`s is returned. An empty list means the frontier is structurally and temporally well-formed in the graph. |
| Failure postcondition | `ctx.Err()` is returned if the context is cancelled. |

## Main success scenario

1. Caller invokes `composition.DefaultEngine.Audit(ctx, g, f)`.
2. System runs the in-graph structural audit: distinct edges sharing the same `(from, to)` endpoint pair with incompatible types, where both endpoints lie in the frontier, are reported as `Structural` conflicts (`composition.structuralAudit`).
3. System runs the in-graph temporal audit: vertices in the frontier whose `TimeTriple` is malformed (`ValidTo != 0 && ValidTo < ValidFrom`) are reported as `Temporal` conflicts (`composition.temporalAudit`).
4. System returns the accumulated conflicts (empty if the frontier is well-formed).

## Extensions

### Successful variations

- **1a. Empty / well-formed frontier:**
  - 1a1. The audit finds nothing and returns an empty slice.
- **2a. Strictness independence:**
  - 2a1. `Audit` runs the checks regardless of the engine's `Lenient`/`Strict` setting — it is an explicit, always-on call, unlike `Merge` which only audits in `Strict` mode.

### Failure paths

- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Per-side data not required:** unlike the per-side merge audit (Textual/Trust/Schema, which needs two `projection.Edited` frontiers), the in-graph audit needs only the host graph and a single frontier.

## Related use cases

- Consumed by: `repo.Service.ReleaseStrict`, which runs this audit before the governance gate (UC-S13) and refuses release with `repo.ErrReleaseAudit` on a non-empty result. Plain `repo.Service.Release` deliberately omits this audit.
- Shares the audit implementation with UC-S03 (guarded-pushout merge), whose `Strict` mode runs the same in-graph checks before gating.
