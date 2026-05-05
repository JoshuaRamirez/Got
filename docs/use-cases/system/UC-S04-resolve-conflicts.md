# UC-S04: Apply conflict resolutions

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/composition` |
| Primary actor | `composition.Engine` |
| Stakeholders & interests | Integrator: discharge conflicts from a prior `Merge`. Auditor: every applied resolution is a recorded graph mutation. |
| Preconditions | A `MergeResult` with non-empty `Conflicts` and a list of `Resolution` values are supplied. |
| Trigger | `repo.Service` or higher caller invokes resolution after a conflicted merge. |
| Success postcondition | A new `MergeResult` with empty `Conflicts` is returned. |
| Failure postcondition | A `MergeResult` whose `Conflicts` is still non-empty, or an error. |

## Main success scenario

1. System iterates the supplied resolutions, calling `Resolution.Apply(g)` for each.
2. System accumulates the rewritten graph.
3. System re-runs the pushout step on the rewritten graph (UC-S03 step 1-2).
4. System checks the resulting frontier against policies (UC-S12).
5. System emits a fresh certificate (UC-S06).
6. System returns the populated `MergeResult` with empty `Conflicts`.

## Extensions

### Successful variations

- **1a. Resolution list shorter than conflict list:**
  - 1a1. System applies what is supplied. Remaining conflicts surface in the returned `MergeResult`.
- **2a. Resolution is a no-op:**
  - 2a1. System records it for audit and proceeds.

### Failure paths

- **1b. A `Resolution.Apply` returns an error:**
  - 1b1. System returns `composition.ErrConflictUnresolvable` wrapping the offending resolution.
- **3a. Pushout still inadmissible after resolution:**
  - 3a1. System returns `composition.ErrNoPushout`.
- **4a. Policy aggregate is `Unsat`:**
  - 4a1. System returns the merge result with a `Policy`-kind conflict.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Resolution ordering:** sequential by default. The audit trail records the actual apply order.

## Related use cases

- Includes: UC-S03 (Compute pushout), UC-S12 (Check policy aggregate), UC-S06 (Issue certificate).
- Included by: UC-U17 (Resolve merge conflicts).
