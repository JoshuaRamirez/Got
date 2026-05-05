# UC-U05: Evaluate a frontier in an environment

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo.Service` |
| Primary actor | Reviewer / CI |
| Stakeholders & interests | Reviewer: get a deterministic verdict on a frontier. Auditor: evaluation result is recorded with environment binding. Operator: evaluation does not mutate the graph except to record the evaluation vertex. |
| Preconditions | The supplied `Frontier` is a subset of `state.Graph()` vertex IDs. The `EnvironmentBinding` identifies a known environment. |
| Trigger | Reviewer or CI pipeline asks the system to score a frontier. |
| Success postcondition | A new `State` is returned together with an `Evaluation` whose `Target` matches the frontier and whose `Environment` matches the binding. The graph contains an `Evaluation` vertex and an `evaluated_by` edge linking it to the frontier. |
| Failure postcondition | The input `State` is unchanged. An error is reported. |

## Main success scenario

1. Actor invokes `repo.Service.Evaluate(ctx, state, frontier, env)`.
2. System invokes `verification.Engine.Evaluate(ctx, graph, frontier, env)` (UC-S05).
3. System derives an Evaluation vertex with content-addressed identity (UC-S17).
4. System extends the graph with the Evaluation vertex and `evaluated_by` edge.
5. System validates the resulting graph (UC-S01).
6. System returns the new `State` and the `Evaluation` value.

## Extensions

### Successful variations

- **2a. Environment supports cached evaluation:**
  - 2a1. System reuses a prior evaluation with identical `(frontier, env)` and skips to step 6.
- **3a. Evaluation produces multiple result rows:**
  - 3a1. System aggregates them into the single returned `Evaluation` per the engine's contract.

### Failure paths

- **2b. Environment unrecognized:**
  - 2b1. System returns `verification.ErrEnvironmentMismatch`.
- **2c. Frontier vertex missing:**
  - 2c1. System returns `graph.ErrVertexNotFound`.
- **4a. Evaluation vertex would violate ontology admissibility:**
  - 4a1. System returns `graph.ErrNotWellFormed`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()` and emits no Evaluation vertex.

## Sub-variations

- **Result kind:** boolean pass/fail, scalar score, structured object — all carry a `ResultValue` that supports `Compare`.
- **Cache hit / miss:** affects latency, not result.

## Related use cases

- Includes: UC-S05 (Evaluate frontier in environment), UC-S01 (Validate graph), UC-S17 (Compute identifier).
- Related: UC-U15 (Prove a claim with a proof).
