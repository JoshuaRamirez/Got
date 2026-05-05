# UC-S18: Check whether an edge or hyperedge is admissible

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/ontology` |
| Primary actor | `ontology.Schema` |
| Stakeholders & interests | Caller (typically `graph.Validate`): yes/no admissibility for a given type signature. |
| Preconditions | The schema is in scope. |
| Trigger | A validate or write path needs to confirm a type signature is allowed. |
| Success postcondition | A boolean is returned: true if the signature is in the admissibility table, false otherwise. |
| Failure postcondition | None — this UC has no error path; the schema is a pure value. |

## Main success scenario

1. Caller invokes `Schema.EdgeAllowed(srcType, edgeType, dstType)` or `Schema.HyperedgeAllowed(inputs, edgeType, outputs)`.
2. System looks up the signature in its admissibility table.
3. System returns the boolean.

## Extensions

### Successful variations

- **1a. Type unknown to the schema:**
  - 1a1. `KnownVertexType` / `KnownEdgeType` returns false; this can be checked separately by the caller.

### Failure paths

- None.

## Sub-variations

- **Schema source:** `ontology.NewDefaultSchema()` or caller-supplied.

## Related use cases

- Included by: UC-S01 (Validate graph), UC-U01 (Ingest), UC-U02 (Revise).
