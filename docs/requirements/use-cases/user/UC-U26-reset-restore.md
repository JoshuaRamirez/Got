# UC-U26: Reset a branch and restore the working graph

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`reset`, `restore`) over `repo`, `history` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: move a branch tip to an earlier (or other) commit, and discard working changes back to a committed state. |
| Preconditions | An initialized repository with commits. |
| Trigger | `got reset [--hard] <commit-ish>` or `got restore [<commit-ish>]`. |
| Success postcondition | `reset` repoints the current branch tip to the target commit (and, with `--hard`, rewrites the working graph to it). `restore` rewrites the working graph to a commit's state, discarding uncommitted changes. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer runs `got reset <commit-ish>`: System repoints the current branch's commit tip to the resolved commit, keeping the working graph as-is (so any difference becomes uncommitted changes).
2. With `--hard`, System also rewrites the working graph (`graph.json`) to the target commit's snapshot.
3. Developer runs `got restore [<commit-ish>]`: System rewrites the working graph to the resolved commit's snapshot (default: the current branch tip), discarding uncommitted changes.

## Extensions

### Successful variations

- **1a. Soft reset keeps work:** without `--hard`, the working graph is unchanged; `status` then shows the difference against the new tip.
- **3a. Restore to a specific commit:** `restore <commit-ish>` sets the working graph to any commit's state, not only HEAD.

### Failure paths

- **1b. Unknown commit-ish:** `reset`/`restore` of a ref resolving to nothing reports "unknown commit-ish".
- **3b. No commits:** `restore` with no argument on a branch with no commits reports an error.

## Sub-variations

- **Reset moves the tip, restore moves the tree:** `reset` changes which commit the branch points at; `restore` changes the working graph — the two axes git splits across `reset`/`restore`/`checkout -- .`.

## Related use cases

- Includes: UC-U22 (commit tips), UC-U23 (working graph), UC-S26 (history), UC-U25 (commit-ish resolution).
