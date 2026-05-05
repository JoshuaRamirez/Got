# UC-S14: Materialize a view for a specific target

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/realization` |
| Primary actor | `realization.Engine` |
| Stakeholders & interests | Consumer: a `Bundle` for the target. Auditor: every path's provenance lies in the source view. |
| Preconditions | A `View` and a `Target` are supplied. |
| Trigger | `repo.Service.Materialize` (UC-U06) calls down. |
| Success postcondition | A `Bundle` is returned with paths and provenance witnesses. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System looks up the registered materializer for `target`.
2. System invokes the materializer against `view.Subgraph()`.
3. System assembles the bundle with paths, provenance witnesses, and `FidelityContract`.
4. System returns the bundle.

## Extensions

### Successful variations

- **2a. Empty view:**
  - 2a1. System returns a bundle with zero paths.
- **3a. Streaming materializer:**
  - 3a1. Bundle exposes paths lazily; provenance is computed at access time.

### Failure paths

- **1a. No materializer registered for target:**
  - 1a1. System returns `realization.ErrTargetUnsupported`.
- **3b. Provenance for some path escapes the view:**
  - 3b1. System returns an error describing the offending path; the bundle is discarded.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Fidelity contract:** lossless, lossy-with-witness, etc.

## Related use cases

- Included by: UC-U06 (Materialize bundle).
