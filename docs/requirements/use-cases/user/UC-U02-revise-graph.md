# UC-U02: Revise the graph via a rewrite rule

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo.Service` |
| Primary actor | Author / refactor tool |
| Stakeholders & interests | Author: a structural change lands atomically. Auditor: every rewrite produces a `ChangeCapsule` that records what was consumed and produced. Operator: rewrite cannot break well-formedness. |
| Preconditions | A `revision.Rule` and a `revision.Match` are available. The match's vertex mapping points only into the current graph. |
| Trigger | Author wants to refactor part of the graph. |
| Success postcondition | A new `State` is returned whose graph extends the input graph by deleting the consumed pattern, preserving the context, and adding the produced pattern. A `ChangeCapsule` records the rewrite. |
| Failure postcondition | The input `State` is unchanged. An error is reported. |

## Main success scenario

1. Actor invokes `repo.Service.Revise(ctx, state, rule, match)`.
2. System checks the match embeds into the host graph (UC-S02 step 1).
3. System verifies all `rule.SideConditions()` predicates hold against the match.
4. System applies the DPO rewrite (UC-S02): removes consumed vertices/edges, retains context, inserts the right-hand side.
5. System validates the resulting graph (UC-S01).
6. System emits a `ChangeCapsule` recording consumed and produced vertex IDs, the actor, environment, and policies in scope.
7. System returns a new `State` plus the capsule embedded in the graph as audit metadata.

## Extensions

### Successful variations

- **3a. Side conditions partially evaluable:**
  - 3a1. System logs the unknowns and proceeds when all decidable predicates pass.
- **4a. Rewrite is identity (left side equals right side):**
  - 4a1. System returns the input state, but still emits a capsule with empty Consumed/Produced for audit.

### Failure paths

- **2b. Match does not embed (vertex IDs in the match map are not all present in the graph):**
  - 2b1. System returns `revision.ErrNoMatch`.
- **3b. A side condition fails:**
  - 3b1. System returns `revision.ErrSideConditionFailed`, wrapped with the offending predicate's identity.
- **4b. Pushout complement does not exist (DPO inapplicable):**
  - 4b1. System returns an error wrapping `revision.ErrNoMatch`.
- **5b. Resulting graph is not well-formed:**
  - 5b1. System returns `graph.ErrNotWellFormed` and discards the partial result.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()` and leaves the input state untouched.

## Sub-variations

- **Rule source:** library-defined, user-supplied, derived from another rewrite.
- **Match cardinality:** unique match supplied; ambiguous match (multiple embeddings exist) — caller chose one explicitly.
- **Audit detail:** capsule metadata can include actor, environment, policies, or be minimal.

## Related use cases

- Includes: UC-S02 (Apply DPO rewrite), UC-S01 (Validate graph), UC-S19 (Check replay feasibility).
- Extended by: UC-U14 (Replay change capsule).
