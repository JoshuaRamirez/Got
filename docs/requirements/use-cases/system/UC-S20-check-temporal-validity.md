# UC-S20: Check the temporal validity of a vertex

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/temporal` |
| Primary actor | `temporal.Engine` |
| Stakeholders & interests | Caller: an `Interval` describing the vertex's validity window. Compliance: temporal staleness can be tested against this interval. |
| Preconditions | The target vertex exists in the graph and carries a `TimeTriple`. |
| Trigger | A higher-level flow needs the validity interval. |
| Success postcondition | An `Interval{From, To}` is returned. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System looks up the vertex.
2. System reads `vertex.Time.ValidFrom` and `vertex.Time.ValidTo`.
3. System returns `Interval{From: ValidFrom, To: ValidTo}`.

## Extensions

### Successful variations

- **2a. `ValidTo` is the configured "indefinite" sentinel:**
  - 2a1. System returns the interval with the sentinel preserved; callers interpret per the engine's contract.

### Failure paths

- **1a. Vertex not in graph:**
  - 1a1. System returns `temporal.ErrUnknownVertex`.
- **2b. Time triple malformed (`ValidTo < ValidFrom`):**
  - 2b1. System returns an error wrapping the offending triple.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Time sources:** `EventTime`, `CausalTime`, `ValidFrom`/`ValidTo`. This UC concerns only the validity interval.

## Related use cases

- Included by: UC-U13 (Check freshness).
