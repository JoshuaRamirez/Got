# UC-S27: Compute a structural diff between two graph states

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `graph` (via `graph.Diff`, `graph.Delta`) |
| Primary actor | `graph.Diff` |
| Stakeholders & interests | Reviewer: see what changed between two commits at the level of vertices and edges, not lines. Integrator: drive a merge or a review from a structure-aware delta. |
| Preconditions | Two graph snapshots (UC-S23), typically the resulting states of two commits (UC-S26). |
| Trigger | A caller asks for the difference from an old snapshot to a new one. |
| Success postcondition | A `Delta` listing added, removed, and changed vertices and edges is returned. |
| Failure postcondition | None — `Diff` is total; an all-empty `Delta` means the snapshots are structurally identical. |

## Main success scenario

1. Caller invokes `graph.Diff(old, new)`.
2. System indexes each snapshot's vertices and edges by ID.
3. System classifies each element: present only in `new` → **added**; only in `old` → **removed**; in both but with differing content → **changed** (`Delta.ChangedVertices` / `ChangedEdges` carry the old and new values).
4. System returns the `Delta`.

## Extensions

### Successful variations

- **3a. Changed element:** because a `VertexID` is `sha256(name)` in this system (not a full-content hash), the same ID can carry different `Type`/`Attrs`/`Time`/`Trust` across snapshots — so a content change is reported as `Changed`, not as a remove+add pair. (Under a pure full-content hash, equal IDs would imply equal content and `Changed` would be empty.)
- **3b. No differences:** identical snapshots yield `Delta.Empty() == true`.

### Failure paths

- None. `Diff` never errors.

## Sub-variations

- **Semantic, not textual:** the diff is over graph structure (typed vertices and edges), so moving or reordering does not register as change and independent edits do not collide — the opposite of git's line diff.
- **Reversibility:** `Diff(a, b)` and `Diff(b, a)` swap added and removed.

## Related use cases

- Consumes: UC-S23 snapshots.
- Included by: UC-U22 (`got diff <branch>` diffs a branch's last commit against its parent, or two branch heads against each other).
