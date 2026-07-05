# UC-S23: Serialize and deserialize a graph

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `graph` (via `graph.EncodeSnapshot` / `Snapshot.Build`, `graph.Marshal` / `Unmarshal`) |
| Primary actor | `graph` codec |
| Stakeholders & interests | A repository host: persist or transport a graph without replaying the ingest history. Auditor: an inspectable, canonical on-disk form. |
| Preconditions | For decode, an `ontology.Schema` to validate against. Attribute values must be JSON-serializable for the JSON path. |
| Trigger | A host encodes a graph to bytes/snapshot, or decodes one back. |
| Success postcondition | Encoding produces a `Snapshot` (or JSON) carrying every vertex, edge, and hyperedge with all fields. Decoding reconstructs a structurally identical, well-formed graph (equal IDs ⇒ equal content, since the graph is content-addressed). |
| Failure postcondition | Decoding returns an error for a malformed ID, a missing edge/hyperedge endpoint, or a graph that is not well-formed under the schema. |

## Main success scenario

1. Host calls `graph.EncodeSnapshot(g)`, producing a `Snapshot` with hex-encoded IDs and every vertex field (type, attrs, `TimeTriple`, `TrustAnnotation`), edge (type, endpoints, attrs), and hyperedge (type, ordered inputs/outputs, attrs).
2. Host serializes the `Snapshot` (e.g. `graph.Marshal(g)` returns JSON).
3. Later, host calls `graph.Unmarshal(schema, data)` (or `Snapshot.Build(schema)`), which rebuilds the graph via `graph.Builder` — adding all vertices, then edges, then hyperedges — and runs `Validate` (UC-S01).
4. The reconstructed graph is structurally identical to the original.

## Extensions

### Successful variations

- **1a. Empty graph:** encodes and decodes to an empty graph.
- **3a. Snapshot value reuse:** a caller may hold the `Snapshot` value directly (for inspection or a non-JSON transport) and call `Snapshot.Build` without going through JSON.

### Failure paths

- **3b. Malformed ID:** a snapshot ID that is not 32-byte hex is rejected with an error.
- **3c. Missing endpoint:** an edge or hyperedge referencing a vertex absent from the snapshot is rejected (`graph.ErrMissingEndpoint`) at build time.
- **3d. Inadmissible graph:** a snapshot whose edges/hyperedges violate schema admissibility is rejected on load (`graph.ErrNotWellFormed`) — validation runs at decode, so a corrupt or hand-edited snapshot cannot produce an ill-formed graph.
- **2a. Non-serializable attribute:** `graph.Marshal` returns the underlying `encoding/json` error if an attribute value cannot be marshalled.

## Sub-variations

- **Content-addressing:** because identity is the hash of canonical content, a decoded snapshot shares IDs with the original; round-tripping is lossless by construction.

## Related use cases

- Includes: UC-S01 (Validate graph well-formedness) — run on every decode.
- Enables: repository-level persist/reload (see UC-U20), which composes this codec with the durable namespace store (UC-S22).
