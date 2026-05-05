# UC-S12: Check the aggregate decision over a policy set

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/governance` |
| Primary actor | `governance.Engine` |
| Stakeholders & interests | Caller: aggregate `Decision` and outstanding `Obligation`s. Compliance: every gating decision passes through this UC. |
| Preconditions | A frontier and a policy set are supplied. |
| Trigger | A higher-level flow needs to check policies. |
| Success postcondition | A `Decision` (one of `Sat`, `Unknown`, `Unsat`) and a possibly-empty `[]Obligation` slice are returned. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System iterates each `Policy.Check(g, frontier)` (UC-S18 ensures admissibility prerequisites).
2. System aggregates per-policy decisions: any `Unsat` makes the aggregate `Unsat`; otherwise any `Unknown` makes it `Unknown`; otherwise `Sat`.
3. System concatenates the obligations from each policy.
4. System returns `(decision, obligations, nil)`.

## Extensions

### Successful variations

- **1a. Empty policy set:**
  - 1a1. System returns `(Sat, nil, nil)` per the trivial-aggregate convention.
- **2a. All policies return `Sat`:**
  - 2a1. System returns `(Sat, nil, nil)`.

### Failure paths

- **1b. A policy's `Check` returns an error:**
  - 1b1. System returns the error wrapped with the policy's name.
- **\*. `ctx` cancelled mid-iteration:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Aggregate strictness:** default rules above; alternate aggregators (e.g. weighted) may be configured.

## Related use cases

- Included by: UC-U04 (Merge), UC-U07 (Promote release), UC-S03 (Pushout), UC-S06 (Issue certificate), UC-S13 (Gate release), UC-U16 (Detect emergent capability).
