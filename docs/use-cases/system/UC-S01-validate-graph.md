# UC-S01: Validate graph well-formedness

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/graph` |
| Primary actor | Calling Engine or Service |
| Stakeholders & interests | All write paths: an inadmissible graph must never be returned. Caller: receive a precise sentinel describing the violation. |
| Preconditions | A `graph.Graph` instance is in hand. |
| Trigger | A write path needs to confirm well-formedness before returning a new graph value. |
| Success postcondition | `nil` is returned; the graph is well-formed under the configured `ontology.Schema`. |
| Failure postcondition | `graph.ErrNotWellFormed` (wrapped with detail) is returned. |

## Main success scenario

1. Caller invokes `graph.Graph.Validate()`.
2. System iterates every edge: looks up source and destination vertices and calls `Schema.EdgeAllowed(srcType, edgeType, dstType)` (UC-S18).
3. System iterates every hyperedge: collects input and output vertex types and calls `Schema.HyperedgeAllowed(inputs, edgeType, outputs)` (UC-S18).
4. System returns nil.

## Extensions

### Successful variations

- **1a. Empty graph:**
  - 1a1. System returns nil immediately.
- **2a. Edges only, no hyperedges:**
  - 2a1. System completes after the edge loop.

### Failure paths

- **2b. Edge references missing source or destination:**
  - 2b1. System returns `graph.ErrNotWellFormed` with the missing endpoint ID.
- **2c. Edge type not admissible for the endpoint vertex types:**
  - 2c1. System returns `graph.ErrNotWellFormed` naming the offending triple `(srcType, edgeType, dstType)`.
- **3b. Hyperedge references missing input or output vertex:**
  - 3b1. System returns `graph.ErrNotWellFormed`.
- **3c. Hyperedge type not admissible for the input/output type signature:**
  - 3c1. System returns `graph.ErrNotWellFormed`.

## Sub-variations

- **Schema source:** `ontology.NewDefaultSchema()` or a caller-supplied schema.

## Related use cases

- Includes: UC-S18 (Check ontology admissibility).
- Included by: UC-U01 (Ingest), UC-U02 (Revise), UC-U05 (Evaluate), UC-U06 (Materialize), and any other write path.
