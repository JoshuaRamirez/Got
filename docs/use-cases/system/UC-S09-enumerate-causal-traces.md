# UC-S09: Enumerate causal traces between two vertices

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/provenance` |
| Primary actor | `provenance.Engine` |
| Stakeholders & interests | Auditor: every distinct simple path between two vertices through causal edges. |
| Preconditions | Both `from` and `to` exist in the graph. |
| Trigger | A higher-level flow needs the trace set. |
| Success postcondition | A slice of `Trace{Vertices: [...]}` values is returned, each starting at `from` and ending at `to`. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System builds the undirected adjacency over causal edges.
2. System DFS from `from`, maintaining a visited set and current path.
3. For each path that reaches `to`, system copies the path into a `Trace` value.
4. System returns the accumulated traces.

## Extensions

### Successful variations

- **3a. No path exists:**
  - 3a1. System returns `nil` traces and nil error.
- **3b. Many paths exist:**
  - 3b1. System enumerates all simple paths (no cycles).

### Failure paths

- **1a. Either endpoint not in graph:**
  - 1a1. System returns `provenance.ErrUnknownVertex`.
- **\*. `ctx` cancelled mid-traversal:**
  - System returns `ctx.Err()`. No partial trace set is exposed.

## Sub-variations

- **Path semantics:** simple paths only; cycle detection elides revisits.

## Related use cases

- Included by: UC-U11 (Trace provenance).
