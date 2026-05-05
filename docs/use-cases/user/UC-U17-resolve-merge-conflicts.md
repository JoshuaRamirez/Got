# UC-U17: Resolve merge conflicts

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo.Service`, `composition.Engine` |
| Primary actor | Integrator |
| Stakeholders & interests | Integrator: drive a previously-conflicted merge to success by supplying resolutions. Auditor: every applied resolution is recorded as a graph mutation. |
| Preconditions | A `MergeResult` from a previous `Merge` call is in hand whose `Conflicts` slice is non-empty. A list of `Resolution` values is prepared, one per conflict (or per conflict kind). |
| Trigger | Integrator wants to discharge conflicts and produce a merged frontier. |
| Success postcondition | A new `MergeResult` is returned with empty `Conflicts` and populated `Frontier` / `Witness` / `Certificate`. |
| Failure postcondition | A `MergeResult` is returned with non-empty `Conflicts` (some resolutions failed), or an error. |

## Main success scenario

1. Actor invokes `composition.Engine.Resolve(ctx, g, mr, resolutions)` (UC-S04).
2. System applies each `Resolution.Apply(g)` in sequence, accumulating the rewritten graph.
3. System re-runs `Merge` (or its core pushout step) against the resolved graph.
4. System validates the new frontier against the policy set (UC-S12).
5. System returns the populated `MergeResult` with empty `Conflicts`.

## Extensions

### Successful variations

- **2a. Some resolutions are no-ops:**
  - 2a1. System still records them in the audit trail and proceeds.
- **3a. After resolution, the original conflicts are reframed (e.g., a `Textual` conflict becomes `Schema`):**
  - 3a1. System surfaces the new conflict set in the returned `MergeResult` rather than failing silently.

### Failure paths

- **2b. A `Resolution.Apply` returns an error:**
  - 2b1. System returns `composition.ErrConflictUnresolvable` wrapping the offending resolution and the original error.
- **3b. After applying resolutions, no admissible pushout exists:**
  - 3b1. System returns `composition.ErrNoPushout`.
- **4a. Policy aggregate is `Unsat` after resolution:**
  - 4a1. System returns the merge result with a `Policy`-kind `Conflict` populated.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Resolution ordering:** applied in the order supplied. Some implementations may reorder for performance — the audit trail records the actual order.

## Related use cases

- Extends: UC-U04 (Merge two frontiers).
- Includes: UC-S04 (Apply conflict resolutions), UC-S03 (Compute pushout), UC-S12 (Check policy aggregate).
