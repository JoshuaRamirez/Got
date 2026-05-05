# UC-S05: Evaluate a frontier in a given environment

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/verification` |
| Primary actor | `verification.Engine` |
| Stakeholders & interests | `repo.Service.Evaluate`: receive an `Evaluation`. Auditor: target and environment are recorded. |
| Preconditions | The frontier vertices exist in the graph. The `EnvironmentBinding` identifies a registered environment. |
| Trigger | `repo.Service.Evaluate` (UC-U05) calls down. |
| Success postcondition | An `Evaluation` value whose `Target() == frontier` and `Environment() == env` is returned. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System resolves the environment binding to its concrete handler.
2. System runs the evaluation against the frontier in that environment.
3. System constructs the `Evaluation` with the target frontier, environment binding, and result value.
4. System returns the evaluation.

## Extensions

### Successful variations

- **2a. Evaluation supports caching:**
  - 2a1. System returns a cached prior evaluation when `(frontier, env)` match exactly.

### Failure paths

- **1a. Environment unknown:**
  - 1a1. System returns `verification.ErrEnvironmentMismatch`.
- **1b. Frontier vertex missing:**
  - 1b1. System returns `graph.ErrVertexNotFound`.
- **2b. Evaluation handler errors:**
  - 2b1. System returns the error wrapped with the frontier identity.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Result kind:** boolean, scalar, structured. All implement `ResultValue.Compare`.

## Related use cases

- Included by: UC-U05 (Evaluate frontier), UC-S06 (Issue certificate).
