# UC-U04: Merge two frontiers

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo.Service` |
| Primary actor | Integrator |
| Stakeholders & interests | Integrator: combine two frontiers into one. Auditor: the merge produces either a `MergeWitness` and `Certificate` or a typed `Conflict` set, never both. Governance: the result is admissible under the supplied policy set. |
| Preconditions | Both `Frontier` values are subsets of `state.Graph()` vertex IDs. The policy set `[]governance.Policy` is supplied. |
| Trigger | Integrator wants to combine work from two branches or projections. |
| Success postcondition | A new `State` is returned together with a `composition.MergeResult` whose `Frontier`, `Witness`, and `Certificate` are populated and whose `Conflicts` is empty. |
| Failure postcondition | The input `State` is returned unchanged together with a `MergeResult` whose `Conflicts` slice is non-empty (and the success fields are zero), or an error. |

## Main success scenario

1. Actor invokes `repo.Service.Merge(ctx, state, left, right, policies)`.
2. System checks that both frontiers' vertex IDs exist in the graph.
3. System computes the guarded pushout under the policy subcategory (UC-S03).
4. System runs the policy aggregate over the merged frontier (UC-S12).
5. System issues a certificate for the merged frontier given evaluations and policies (UC-S06).
6. System extends the graph with the merged frontier's witness vertex.
7. System returns the new `State` and a `MergeResult` populated with the merged `Frontier`, `Witness`, and `Certificate`; `Conflicts` is empty.

## Extensions

### Successful variations

- **3a. One frontier is empty:**
  - 3a1. System adopts the other frontier as the merge result and proceeds to step 4.
- **3b. Frontiers are identical:**
  - 3b1. System returns the existing frontier as the merge result with a synthetic witness; `Conflicts` is empty.

### Failure paths

- **2a. A frontier vertex is missing from the graph:**
  - 2a1. System returns `graph.ErrVertexNotFound`.
- **3c. Pushout complement does not exist (typed conflicts found):**
  - 3c1. System returns the input state and a `MergeResult` with `Conflicts` populated and the success fields zero. Caller may invoke UC-U17 to resolve.
- **3d. No admissible pushout under the policy subcategory:**
  - 3d1. System returns `composition.ErrNoPushout`.
- **4a. Policy aggregate is `Unsat`:**
  - 4a1. System returns the input state and a `MergeResult` with a Policy-kind `Conflict` describing the violated obligations; `Witness` and `Certificate` are zero.
- **5a. Certification fails (obligations remain):**
  - 5a1. System returns `verification.ErrCertificationFailed` and discards any partial witness.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Conflict kinds detected:** any of `Textual`, `Structural`, `Schema`, `Policy`, `Trust`, `Capability`, `Evaluation`, `Temporal`. The `MergeResult` may carry multiple kinds.
- **Policy strictness:** strict (any `Unsat` aborts) or advisory (caller filters obligations).

## Related use cases

- Includes: UC-S03 (Compute guarded pushout), UC-S12 (Check policy aggregate), UC-S06 (Issue certificate).
- Extended by: UC-U17 (Resolve merge conflicts).
