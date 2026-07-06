# UC-U31: Rebase a branch onto another

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`rebase`) over `history`, `graph.Diff` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: move the current branch's commits so they sit on top of another branch, producing linear history. |
| Preconditions | Both branches have commits and share a common ancestor. |
| Trigger | `got rebase <onto>`. |
| Success postcondition | The current branch's commits above the merge base are replayed, in order, on top of `<onto>`'s tip as new commits; the branch tip and working graph advance to the last replayed commit. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer runs `got rebase <onto>`.
2. System resolves the current branch tip and `<onto>`'s tip and computes their merge base (`history.Log.MergeBase`).
3. System collects the current branch's commits above the base, oldest-first.
4. Starting from `<onto>`'s tip, System replays each commit: it applies the commit's forward structural delta (`graph.Diff` of the commit against its old parent) to the running state and records a new commit with the running tip as parent.
5. System points the current branch at the final replayed commit and sets the working graph to it.

## Extensions

### Successful variations

- **2a. Fast-forward:** if the current branch is an ancestor of `<onto>`, System fast-forwards the current branch to `<onto>` (no replay).
- **2b. Already up to date:** if `<onto>` is an ancestor of the current tip, there is nothing to do.

### Failure paths

- **2c. No common ancestor:** rebasing between unrelated histories is refused.
- **1a. No commits:** rebasing a branch (or onto a branch) with no commits reports an error.
- **1b. Onto self:** rebasing a branch onto itself is refused.

## Sub-variations

- **Linear replay:** rebase flattens the replayed range against each commit's first parent (merge commits in the range are linearized), producing new commit ids — a history rewrite, like git.
- **Semantic apply:** each step applies the structural delta (nodes/edges), last-write-wins on overlap, rather than a textual patch.

## Related use cases

- Uses: UC-U24 (`merge-base`), UC-S27 (structural diff / apply), UC-U22 (commit).
