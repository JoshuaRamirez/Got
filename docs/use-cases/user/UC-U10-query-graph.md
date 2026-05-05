# UC-U10: Query the graph

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `graph.Graph` |
| Primary actor | Reader / tool |
| Stakeholders & interests | Reader: retrieve specific vertices, edges, hyperedges, or subgraphs. Operator: queries are read-only against the immutable graph. |
| Preconditions | A `graph.Graph` value is in scope. |
| Trigger | Reader needs structural information. |
| Success postcondition | The requested element(s) or subgraph is returned. |
| Failure postcondition | An error is returned, or `false` for the absent-element pattern on `Vertex` / `Edge` / `Hyperedge` lookups. |

## Main success scenario

1. Actor calls one of the read methods on `graph.Graph`:
   - `Vertex(id)`, `Edge(id)`, `Hyperedge(id)` — single-element lookups.
   - `VertexIDs()`, `Vertices()`, `Edges()`, `Hyperedges()` — full enumerations.
   - `Induce(ids)` — subgraph induced by a vertex set.
   - `Query(q)` — arbitrary `graph.Query` evaluation.
2. System returns the requested data without mutation.

## Extensions

### Successful variations

- **1a. `Induce` with the empty vertex set:**
  - 1a1. System returns an empty subgraph (zero vertices, zero edges).
- **1b. `Query` with a known query type:**
  - 1b1. System dispatches to the registered evaluator and returns the matching subgraph.

### Failure paths

- **1c. Single-element lookup with absent ID:**
  - 1c1. System returns the zero value plus `false`. Not an error.
- **1d. `Induce` with an ID not in the graph:**
  - 1d1. System returns `graph.ErrVertexNotFound` wrapped with the offending ID.
- **1e. `Query` with an unrecognized query kind:**
  - 1e1. System returns `graph.ErrQueryUnsupported`.

## Sub-variations

- **Result mutability:** all results are read-only views or value copies; the caller cannot mutate the underlying graph.

## Related use cases

- Includes: none directly; UC-S01 (Validate graph) is performed at write time, not at query time.
- Related: UC-U11 (Trace provenance), UC-S10 (Select a frontier), UC-S11 (Apply projection spec).
