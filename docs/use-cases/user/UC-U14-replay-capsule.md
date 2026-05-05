# UC-U14: Replay a change capsule

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `replay.Engine` |
| Primary actor | CI / Auditor |
| Stakeholders & interests | Auditor: confirm a recorded rewrite is deterministically reproducible. CI: gate releases on replay determinism. Operator: replay does not mutate the live graph. |
| Preconditions | A `revision.ChangeCapsule` and an `EnvironmentBinding` are supplied. The host graph contains all `Consumed` and `Produced` vertices referenced by the capsule. |
| Trigger | Auditor asks the system to verify a capsule's deterministic reproducibility. |
| Success postcondition | An `Outcome` value is returned with `Deterministic = true`. |
| Failure postcondition | `Outcome.Deterministic = false`, or `replay.ErrNonDeterministic`, or another error. |

## Main success scenario

1. Actor calls `Replay(ctx, g, capsule, env)`.
2. System checks the capsule is replayable against `g` (UC-S19).
3. System re-executes the rewrite recorded by the capsule in the supplied environment.
4. System compares the produced result with the capsule's recorded `Produced` set.
5. System returns `Outcome{Deterministic: true}` if the sets match.

## Extensions

### Successful variations

- **3a. Environment is the same as the capsule's recorded environment:**
  - 3a1. System uses the recorded environment for the comparison; result is determined entirely by the rewrite engine.

### Failure paths

- **2a. Capsule is not replayable (consumed/produced vertices missing in g):**
  - 2a1. System returns the wrapped `revision.ErrNoMatch` from `Replayable`.
- **3b. Environment unrecognized:**
  - 3b1. System returns `verification.ErrEnvironmentMismatch`.
- **4a. Produced set differs from the capsule:**
  - 4a1. System returns `Outcome{Deterministic: false}` and `replay.ErrNonDeterministic`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Comparison granularity:** vertex-set equality (default) vs. structural equality of the produced subgraph.

## Related use cases

- Includes: UC-S19 (Check replay feasibility), UC-S02 (Apply DPO rewrite).
- Related: UC-U02 (Revise the graph via a rewrite rule).
