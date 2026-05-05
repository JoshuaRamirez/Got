# UC-S03: Compute the guarded pushout of two frontiers

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/composition` |
| Primary actor | `composition.Engine` |
| Stakeholders & interests | `repo.Service.Merge`: get a merged frontier or a typed conflict set. Governance: the result must be admissible under the policy subcategory. |
| Preconditions | The host graph contains both frontiers' vertices. A policy set is supplied. |
| Trigger | `repo.Service.Merge` (UC-U04) calls down. |
| Success postcondition | A `MergeResult` with populated `Frontier`, `Witness`, `Certificate` and empty `Conflicts` is returned. |
| Failure postcondition | A `MergeResult` with non-empty `Conflicts` is returned, or an error. |

## Main success scenario

1. System constructs the span `left ←  pullback → right` over the shared subgraph.
2. System constructs the candidate pushout in `Repo_Sigma`.
3. System checks the candidate against the policy subcategory `Repo_Pi` (UC-S12).
4. System extends the graph with a synthetic merge witness vertex.
5. System invokes UC-S06 to issue a certificate for the merged frontier.
6. System returns the `MergeResult` with `Conflicts` empty.

## Extensions

### Successful variations

- **1a. Frontiers identical:**
  - 1a1. System adopts either as the merged frontier and proceeds to step 4.
- **1b. One frontier empty:**
  - 1b1. System adopts the other as the merged frontier.

### Failure paths

- **1c. Pullback yields incompatible structure:**
  - 1c1. System constructs a typed `Conflict` (kind: `Structural` or `Schema`) and returns it in `MergeResult.Conflicts`.
- **2a. Pushout would violate schema admissibility:**
  - 2a1. System returns a `Schema`-kind conflict.
- **3a. Policy aggregate is `Unsat`:**
  - 3a1. System returns a `Policy`-kind conflict naming the violated obligations.
- **3b. No admissible pushout under `Repo_Pi`:**
  - 3b1. System returns `composition.ErrNoPushout`.
- **5a. Certification fails:**
  - 5a1. System returns `verification.ErrCertificationFailed`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Conflict typing:** any combination of `Textual`, `Structural`, `Schema`, `Policy`, `Trust`, `Capability`, `Evaluation`, `Temporal`.
- **Witness shape:** synthetic vertex; configurable type (default `MergeWitness`).

## Related use cases

- Includes: UC-S12 (Check policy aggregate), UC-S06 (Issue certificate), UC-S01 (Validate graph).
- Included by: UC-U04 (Merge), UC-U17 (Resolve conflicts).
