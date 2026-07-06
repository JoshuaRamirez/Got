# UC-U24: Merge a branch semantically

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo` (`MergeStates`), `history` (`MergeBase`), `cmd/got` (`merge`, `merge-base`) |
| Primary actor | Developer / Integrator |
| Stakeholders & interests | Developer: combine another branch's work into the current one without spurious textual conflicts; only genuine same-target divergence should stop the merge. |
| Preconditions | Both branches have commits. |
| Trigger | `got merge <branch>` merges the named branch into the current branch (HEAD). |
| Success postcondition | Either the current branch fast-forwards, or a merge commit with two parents is recorded whose state is the semantic three-way merge of the two branch tips against their common ancestor; the working graph and branch tip advance to it. |
| Failure postcondition | On genuine conflicts, the merge is aborted with typed conflicts and nothing changes. |

## Main success scenario

1. Developer runs `got merge <branch>`.
2. System resolves the current branch's tip commit and the other branch's tip commit.
3. System computes their nearest common ancestor commit (`history.Log.MergeBase`).
4. If the ancestor equals the other tip, System reports "already up to date". If it equals the current tip, System **fast-forwards** the current branch to the other tip (no merge commit) and updates the working graph.
5. Otherwise System performs a semantic three-way merge of the two tip states against the ancestor state (`repo.MergeStates`, via UC-U18): per-vertex and per-edge reconciliation, taking one-sided changes automatically.
6. On a clean merge, System records a merge commit with **two parents** (the two tips), advances the current branch to it, and updates the working graph to the merged state.

## Extensions

### Successful variations

- **4a. Fast-forward:** the current branch has no unique commits, so it simply moves to the other tip.
- **3a. Unrelated histories:** with no common ancestor, the ancestor state is empty and every element is treated as an addition (add/add).

### Failure paths

- **5a. Conflicts:** if the two sides changed the same vertex or edge to different values, System aborts with typed `composition.Conflict`s (Textual/Schema/Trust/Temporal/Structural) and records nothing.
- **2a. No commits:** merging when either branch has no commit reports an error.
- **1a. Self-merge:** merging the current branch into itself is refused.

## Sub-variations

- **Semantic, not textual:** reconciliation is over typed vertices and edges (UC-U18/UC-D edge reconciliation), so independent edits on the two branches merge cleanly where git's line diff would often conflict.
- **`merge-base`:** `got merge-base <a> <b>` prints the nearest common commit — the point the three-way merge diffs against.

## Related use cases

- Includes: UC-U18 (three-way merge engine), UC-S26 (commit DAG / ancestry), UC-U22 (commit), UC-U23 (current branch / working graph).
