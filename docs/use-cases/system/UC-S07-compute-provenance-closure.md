# UC-S07: Compute the provenance closure of a seed set

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/provenance` |
| Primary actor | `provenance.Engine` |
| Stakeholders & interests | Auditor: a closed seed set. Caller: the result satisfies extensivity, monotonicity, and idempotence. |
| Preconditions | The seed vertices exist in the graph. The engine is configured with a causal-edge set. |
| Trigger | A higher-level flow needs the closure of a seed set. |
| Success postcondition | A vertex slice is returned that is a superset of the seed and closed under causal reachability. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System builds the undirected adjacency list over causal edges (UC-S18 informs which edge types qualify).
2. System initializes the visited set with the seed.
3. System BFS-traverses adjacency until the queue is empty, checking `ctx.Err()` per iteration.
4. System returns the visited vertex IDs as a slice.

## Extensions

### Successful variations

- **2a. Empty seed:**
  - 2a1. System returns an empty slice and nil error.
- **3a. Seed vertices are mutually unreachable:**
  - 3a1. System returns the seed unchanged after one BFS that finds no neighbors.

### Failure paths

- **2b. Seed vertex not in graph:**
  - 2b1. System returns `provenance.ErrUnknownVertex`.
- **\*. `ctx` cancelled mid-traversal:**
  - System returns `ctx.Err()`. No partial closure is exposed.

## Sub-variations

- **Causal-edge set:** typically `ontology.CausalEdges`; configurable per engine.

## Related use cases

- Included by: UC-U11 (Trace provenance), UC-S08 (Compute provenance cone).
