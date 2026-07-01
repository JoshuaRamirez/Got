# UC-U18: Three-way merge against a common ancestor

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `composition.Engine` (via `composition.DefaultEngine.MergeThreeWay`), reachable through `repo.Service.MergeThreeWay` |
| Primary actor | Integrator |
| Stakeholders & interests | Integrator: combine two divergent frontiers without losing either side's intentional changes, and be told precisely where the sides genuinely conflict. Auditor: a deletion on one branch must not be silently undone by the other branch. |
| Preconditions | Three frontiers are supplied — a common `ancestor` and two descendants `left` and `right` — over a host graph `g`. To detect content-level (modify/modify) divergence the frontiers must carry per-side content (`projection.Edited`); plain ID frontiers yield presence-only three-way semantics. |
| Trigger | Integrator asks the system to merge `left` and `right` given their last common ancestor. |
| Success postcondition | A `MergeResult` with a populated merged frontier, a witness, and a certificate is returned. The merged frontier honors each side's additions, modifications, and deletions relative to the ancestor; changes made on only one side are taken automatically. |
| Failure postcondition | A `MergeResult` carrying one or more typed `Conflict`s (and no merged frontier) is returned, or an error. The host graph is unchanged. |

## Main success scenario

1. Integrator supplies `ancestor`, `left`, `right` frontiers, the host graph `g`, and a policy set `Ps`.
2. System computes, for the union of vertex IDs across the three frontiers, the per-vertex three-way decision relative to the ancestor (`composition.DefaultEngine.MergeThreeWay`):
   - changed on neither side → keep the ancestor value;
   - changed on exactly one side → take that side's value;
   - changed on both sides to the **same** value → take it (no conflict);
   - added on one side only → include the addition;
   - deleted on one side, unchanged on the other → honor the deletion.
3. System gathers the surviving vertex IDs into a merged frontier carrying the chosen per-side content.
4. System gates the merged frontier through governance (`governance.Engine.Check`); on `Sat` it asks verification to issue a certificate (`verification.Engine.Certify`).
5. System returns a `MergeResult` with the merged frontier, a deterministic witness, and the certificate.

## Extensions

### Successful variations

- **2a. No per-side content (plain frontiers):**
  - 2a1. With non-`Edited` frontiers all three sides read the same content from `g`, so no modify/modify divergence is visible. The merge degrades to presence-only three-way: additions are unioned and deletions are honored, but no content conflicts are raised.
- **2b. Both sides delete the same vertex:**
  - 2b1. System omits the vertex from the merged frontier (deletion agreed).
- **2c. Both sides make the identical change:**
  - 2c1. System takes the agreed value with no conflict.

### Failure paths

- **2d. Modify/modify conflict:**
  - 2d1. A vertex present in all three frontiers is changed on both sides to different values. System emits a typed `Conflict` classified by the first differing dimension — `Schema` (type), `Trust`, `Temporal` (time), or `Textual` (attrs) — and returns it in the `MergeResult` with no merged frontier.
- **2e. Add/add conflict:**
  - 2e1. A vertex absent from the ancestor is added on both sides with different content. System emits a typed `Conflict` (classified as in 2d) and returns it with no merged frontier.
- **2f. Modify/delete conflict:**
  - 2f1. A vertex is deleted on one side and modified (relative to the ancestor) on the other. System emits a `Structural` `Conflict` whose detail records the modify/delete disagreement, and returns it with no merged frontier.
- **4a. Policy gate returns `Unsat`:**
  - 4a1. System returns a single `Policy` `Conflict` carrying the outstanding obligations, with no merged frontier.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Resolution:** the conflicts returned by a failed three-way merge are the same `composition.Conflict` type the two-way `Merge` produces, so they can be fed to `composition.DefaultEngine.ResolveTyped` with the stock typed resolvers (UC-S04, UC-U17).

## Related use cases

- Facade: `repo.Service.MergeThreeWay` delegates to the composition engine when it satisfies `composition.ThreeWayMerger` (the default engine does); otherwise it returns `repo.ErrThreeWayUnsupported`.
- Extends: UC-U04 (Merge two frontiers) — three-way merge adds ancestor-relative reconciliation on top of the two-way union.
- Includes: UC-S03 (guarded pushout gate), UC-S06 (issue certificate), UC-S12 (policy aggregate).
- Resolution path: UC-U17 (Resolve merge conflicts), UC-S04 (apply conflict resolutions).
