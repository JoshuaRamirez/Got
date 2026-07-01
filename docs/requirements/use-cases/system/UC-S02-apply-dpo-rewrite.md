# UC-S02: Apply a DPO rewrite

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/revision` |
| Primary actor | `revision.Engine` |
| Stakeholders & interests | `repo.Service.Revise`: receive a rewritten graph and a `ChangeCapsule`. Auditor: capsule records consumed and produced vertices for replay. |
| Preconditions | A `Rule`, a `Match`, and the host graph are supplied. The match's mapping points only into the host graph. |
| Trigger | `repo.Service.Revise` (UC-U02) calls down to apply a rewrite. |
| Success postcondition | A new `Graph` and a populated `ChangeCapsule` are returned. The new graph extends the input graph by removing the consumed pattern and inserting the right-hand side. |
| Failure postcondition | An error is returned. The input graph is unchanged. |

## Main success scenario

1. System checks the match embeds: every value in `match.Mapping()` is a vertex in `g`.
2. System verifies each `rule.SideConditions()` predicate holds against `(g, match)`.
3. System computes the pushout complement (removes left-context vertices/edges).
4. System inserts the right-hand side via `WithVertex` / `WithEdge` / `WithHyperedge`.
5. System emits a `ChangeCapsule` recording `Consumed`, `Produced`, `Kind`, `Actor`, `Environment`, `Policies`, `Metadata`.
6. System returns `(newGraph, capsule, nil)`.

## Extensions

### Successful variations

- **2a. Side conditions empty:**
  - 2a1. System skips predicate evaluation and proceeds to step 3.
- **3a. Identity rewrite (left == right):**
  - 3a1. System returns the input graph and a capsule with empty `Consumed` and `Produced`.
- **4b. Produced element re-states existing identical content (Strict mode):**
  - 4b1. A produced (R-side) vertex or edge declares an ID already present in the post-deletion graph with structurally identical content.
  - 4b2. System treats the insertion as idempotent and proceeds to step 5.
- **4c. Produced element replaces a consumed ID (Strict mode):**
  - 4c1. A produced ID equals an ID deleted earlier in the same rewrite (`L\K`).
  - 4c2. Because the ID is absent from the post-deletion graph, this is a legitimate delete-then-add and System proceeds to step 5.

### Failure paths

- **1a. Match does not embed:**
  - 1a1. System returns `revision.ErrNoMatch`.
- **2b. A side condition fails:**
  - 2b1. System returns `revision.ErrSideConditionFailed`.
- **3b. Pushout complement does not exist (Strict mode):**
  - 3b1. Deleting a consumed vertex would orphan an incident edge or hyperedge not also consumed.
  - 3b2. System returns `revision.ErrDanglingEdge` and leaves the input graph unchanged. (Lenient mode instead silently drops the orphaned edge.)
- **4a. Inserting the right-hand side violates schema admissibility:**
  - 4a1. System returns `graph.ErrNotWellFormed` and discards the partial graph.
- **4d. Produced element collides with different host content (Strict mode):**
  - 4d1. A produced (R-side) vertex, edge, or hyperedge declares an ID already present in the post-deletion graph but with structurally different content, so the caller-declared ID is not a faithful content address.
  - 4d2. System returns `revision.ErrIdentityCollision` and leaves the input graph unchanged. (Lenient mode instead silently overwrites the existing element via `graph.WithVertex`/`WithEdge`/`WithHyperedge`.)
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Capsule metadata richness:** caller may supply minimal or fully-populated metadata.

## Related use cases

- Includes: UC-S01 (Validate graph), UC-S18 (Check ontology admissibility).
- Included by: UC-U02 (Revise), UC-U14 (Replay capsule).
