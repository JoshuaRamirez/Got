# UC-U01: Ingest content into repository

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo.Service` |
| Primary actor | Author / ingest tool |
| Stakeholders & interests | Author: content lands in the graph with stable identity. Auditor: every ingested vertex carries provenance. Operator: ingest does not corrupt the existing graph. |
| Preconditions | The supplied `State` is well-formed (`graph.Graph.Validate()` would return nil). The payload implements `repo.Payload` and declares a known `PayloadKind`. |
| Trigger | The author has new content to add to the repository. |
| Success postcondition | A new `State` is returned whose graph extends the input graph with the payload's vertices, edges, and hyperedges. The original `State` is unchanged. |
| Failure postcondition | The original `State` is returned unchanged and an error is reported. |

## Main success scenario

1. Actor invokes `repo.Service.Ingest(ctx, state, payload)`.
2. System inspects `payload.PayloadKind()` and confirms the kind is registered.
3. System derives vertex/edge IDs from canonical bytes of the payload (UC-S17).
4. For each vertex, system applies `graph.Graph.WithVertex` to extend the graph.
5. For each edge and hyperedge, system applies `WithEdge` / `WithHyperedge`.
6. System validates the resulting graph (UC-S01).
7. System returns a new `State` wrapping the extended graph.

## Extensions

### Successful variations

- **2a. Empty payload (zero vertices, zero edges):**
  - 2a1. System returns the input `State` unchanged with `nil` error.
- **3a. Payload supplies pre-computed IDs:**
  - 3a1. System verifies the supplied IDs match the canonical-byte hash; on match, skips re-derivation and proceeds to step 4.
- **4a. Vertex already present (same ID):**
  - 4a1. System replaces the existing vertex per `WithVertex` semantics. (Append-only is preserved at the graph level — the ID is unchanged.)

### Failure paths

- **2b. Unknown `PayloadKind`:**
  - 2b1. System returns `repo.ErrIngestRejected` wrapped with the offending kind.
- **3b. Canonical encoding fails (`identity.Canonical.CanonicalBytes` returns error):**
  - 3b1. System returns the error wrapped with `repo.ErrIngestRejected`.
- **5b. Edge endpoint missing in the resulting graph:**
  - 5b1. System returns `graph.ErrMissingEndpoint`. The intermediate state is discarded.
- **6b. Schema admissibility violated (edge or hyperedge type not allowed for endpoint types):**
  - 6b1. System returns `graph.ErrNotWellFormed`.
- **\*. `ctx` cancelled at any step:**
  - System returns `ctx.Err()` immediately. No partial state is exposed.

## Sub-variations

- **Payload origin:** CLI, library call, API gateway.
- **Payload size:** single vertex, batch of N vertices, full subgraph.
- **Identity supplied:** derived (canonical-bytes hash) or pre-computed and verified.

## Related use cases

- Includes: UC-S17 (Compute content-addressed identifier), UC-S01 (Validate graph well-formedness), UC-S18 (Check ontology admissibility).
- Extended by: none.
