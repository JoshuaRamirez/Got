# UC-U30: Stash uncommitted working changes

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`stash push`/`pop`/`list`) over `repo`, `graph.Diff` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: set uncommitted changes aside to get a clean working graph, then restore them later. |
| Preconditions | An initialized repository. |
| Trigger | `got stash [push|pop|list]`. |
| Success postcondition | `push` saves the current working state onto a stash stack and resets the working graph to HEAD; `pop` restores the most recent stashed state; `list` shows the stack. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer runs `got stash` (or `got stash push`). If the working graph differs from HEAD's commit, System pushes the current working snapshot onto the stash stack (`stash.json`) and resets the working graph to the HEAD commit's state.
2. Developer runs `got stash pop`. System restores the most recently stashed working snapshot into the working graph and removes it from the stack.
3. Developer runs `got stash list`. System lists the stash stack, newest first, with the branch each was taken on.

## Extensions

### Successful variations

- **1a. Nothing to stash:** if the working graph is already clean relative to HEAD, `push` reports "nothing to stash".

### Failure paths

- **2a. Empty stack:** `pop` with no stashes reports "no stashes".

## Sub-variations

- **Stack semantics:** stashes form a LIFO stack persisted in `stash.json`; each records the branch it was taken on.
- **Restore, not merge:** `pop` restores the stashed working snapshot; it does not three-way-merge it against intervening changes (a possible enhancement).

## Related use cases

- Uses: UC-U23 (working graph / HEAD), UC-S27 (clean detection via diff).
