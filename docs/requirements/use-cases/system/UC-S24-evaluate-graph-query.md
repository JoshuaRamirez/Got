# UC-S24: Evaluate a graph query

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `graph.Graph` (via `Graph.Query` and the `Query` types in `query.go`) |
| Primary actor | `graph.Graph` |
| Stakeholders & interests | Reader / tool: select a subgraph by declarative criteria (type, attribute) without scanning the whole graph by hand. |
| Preconditions | A graph and a `Query` value. |
| Trigger | A caller invokes `Graph.Query(q)` (reachable from UC-U10). |
| Success postcondition | A `Subgraph` induced on the matching vertices is returned — every edge and hyperedge whose endpoints all lie in the matched set is included. |
| Failure postcondition | `graph.ErrQueryUnsupported` is returned for an unrecognized `Query` type. |

## Main success scenario

1. Caller builds a `Query`: `graph.ByType{Type}` (vertices of a type) or `graph.ByAttr{Key, Value}` (vertices whose attribute equals a value, by deep equality).
2. Caller invokes `Graph.Query(q)`.
3. System resolves the query to the matching vertex-ID set (`matchVertices`).
4. System induces the subgraph on that set (`Graph.Induce`) — including edges/hyperedges wholly within the set — and returns it.

## Extensions

### Successful variations

- **1a. Composite query:** `graph.And{Queries}` (set intersection) and `graph.Or{Queries}` (set union) compose sub-queries; they nest arbitrarily. An empty `And` or `Or` matches no vertices.
- **3a. No matches:** the query matches nothing and an empty subgraph is returned (not an error).

### Failure paths

- **2a. Unknown query type:** a `Query` whose concrete type the engine does not recognize yields `graph.ErrQueryUnsupported`. Inside a composite, the error propagates out.

## Sub-variations

- **Attribute equality:** `ByAttr` uses deep equality on the stored value; a vertex missing the key does not match.
- **Induced closure:** the returned subgraph is always vertex-induced, so it is internally consistent (no dangling edges).

## Related use cases

- Includes: UC-S01 semantics implicitly (the induced subgraph is well-formed by construction; `Induce` never yields dangling edges).
- Included by: UC-U10 (Query the graph) — the user-level goal this sub-function fulfills.
