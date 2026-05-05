# UC-U13: Check temporal freshness of a vertex

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `temporal.Engine` |
| Primary actor | Reader / CI |
| Stakeholders & interests | Reader: determine whether a vertex's validity covers the current moment. Compliance: temporal staleness can disqualify a frontier from release. |
| Preconditions | The target vertex exists in the graph and carries a `TimeTriple` with `ValidFrom` / `ValidTo`. |
| Trigger | Reader needs to know the vertex's current freshness or its full validity interval. |
| Success postcondition | `Validity` returns the `Interval`. `Fresh` returns `true` if `now ∈ [ValidFrom, ValidTo)`, else `false`. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. Actor calls `Validity(ctx, g, id)` or `Fresh(ctx, g, id, now)`.
2. System looks up the vertex, reads its `TimeTriple`.
3. System constructs `Interval{From: ValidFrom, To: ValidTo}` (UC-S20) or computes the membership predicate.
4. System returns the result.

## Extensions

### Successful variations

- **3a. Half-open semantics:**
  - 3a1. `Fresh` returns `true` iff `ValidFrom <= now < ValidTo`. `now == ValidTo` is `false`.
- **3b. `ValidTo` is sentinel (e.g. zero or `MaxInt64` for "indefinite"):**
  - 3b1. System treats per its configured sentinel convention; default behavior is documented in the engine.

### Failure paths

- **2a. Vertex not in graph:**
  - 2a1. System returns `temporal.ErrUnknownVertex`.
- **2b. Vertex carries malformed time triple (`ValidTo < ValidFrom`):**
  - 2b1. System returns an error wrapping the offending triple.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Time semantics:** wall-clock (`EventTime`), causal (`CausalTime`), valid (`ValidFrom`/`ValidTo`). Freshness uses the valid interval.

## Related use cases

- Includes: UC-S20 (Check temporal validity).
