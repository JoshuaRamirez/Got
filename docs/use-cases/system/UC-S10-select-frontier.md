# UC-S10: Select a frontier from the graph

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/projection` |
| Primary actor | `projection.Engine` |
| Stakeholders & interests | Caller: a `Frontier` whose vertex IDs are a subset of the graph. |
| Preconditions | A `Selector` is supplied. |
| Trigger | A higher-level flow needs to choose a frontier. |
| Success postcondition | A `Frontier` is returned. `frontier.VertexIDs() ⊆ g.VertexIDs()`. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System invokes `selector.Frontier(g)`.
2. System wraps the returned vertex IDs in a `Frontier` value.
3. System returns the frontier.

## Extensions

### Successful variations

- **1a. Selector returns the empty set:**
  - 1a1. System returns an empty `Frontier` (no error).

### Failure paths

- **1b. Selector cannot be evaluated against this graph:**
  - 1b1. System returns `projection.ErrInvalidSelector`.
- **1c. Selector returns IDs not in the graph:**
  - 1c1. System returns an error wrapping `graph.ErrVertexNotFound`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Selector kinds:** branch selector, label selector, query-derived selector.

## Related use cases

- Included by: UC-S11 (Apply projection spec), UC-U10 (Query graph).
