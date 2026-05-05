# UC-S19: Check whether a change capsule is replayable

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/revision` |
| Primary actor | `revision.Engine` |
| Stakeholders & interests | Replay caller: yes/no on whether the capsule's recorded vertices line up with the current graph. |
| Preconditions | A `ChangeCapsule` and a host graph are supplied. |
| Trigger | Replay flow (UC-U14) needs to gate the actual re-execution. |
| Success postcondition | `nil` is returned: every `Consumed` and `Produced` vertex in the capsule is present in the host graph. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System checks each `id` in `capsule.Consumed`: is `id ∈ g.VertexIDs()`?
2. System checks each `id` in `capsule.Produced` similarly.
3. System returns nil.

## Extensions

### Successful variations

- **1a. Capsule has empty `Consumed` and `Produced`:**
  - 1a1. System returns nil immediately.

### Failure paths

- **1b. A consumed vertex is missing:**
  - 1b1. System returns `revision.ErrNoMatch` wrapping the missing ID.
- **2a. A produced vertex is missing:**
  - 2a1. System returns `revision.ErrNoMatch` wrapping the missing ID.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Strictness:** the default checks both Consumed and Produced presence. A weaker variant could check only Consumed (used for "would replay apply?" queries).

## Related use cases

- Included by: UC-U14 (Replay capsule).
