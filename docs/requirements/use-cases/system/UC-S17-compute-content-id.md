# UC-S17: Compute a content-addressed identifier

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/identity` |
| Primary actor | `identity.Factory` |
| Stakeholders & interests | Caller: deterministic identity for vertices, edges, and hyperedges. Auditor: identical canonical bytes always yield identical IDs. |
| Preconditions | A value implementing `identity.Canonical` is supplied. |
| Trigger | A write or query path needs an ID. |
| Success postcondition | A typed identifier (`VertexID`, `EdgeID`, `HyperedgeID`) equal to `Hasher.Sum(canonical.CanonicalBytes())` is returned. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. System calls `canonical.CanonicalBytes()` to obtain the canonical encoding.
2. System invokes `hasher.Sum(bytes)` to compute the fixed-size hash.
3. System wraps the hash in the typed identifier and returns it.

## Extensions

### Successful variations

- **1a. Cached canonical encoding:**
  - 1a1. Caller may pre-compute and pass the canonical bytes; the factory honors them.

### Failure paths

- **1b. `CanonicalBytes` returns an error (encoding failed):**
  - 1b1. System returns the error wrapped with the value's type.

## Sub-variations

- **Hash algorithm:** SHA-256 by default (`SHA256Hasher`); the `Hasher` interface allows alternates.

## Related use cases

- Included by: UC-U01 (Ingest content), UC-U02 (Revise), UC-S02 (Apply DPO rewrite), UC-S03 (Compute pushout).
