# UC-U28: Blame a node and query its history

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`blame`, `log --touching`) over `history`, `graph.Diff` |
| Primary actor | Developer / Auditor |
| Stakeholders & interests | Developer: find which commit introduced a node and which last changed it, and list every commit that touched it — precisely, per node, rather than by line heuristic. |
| Preconditions | An initialized repository with commits on the current branch. |
| Trigger | `got blame <name>` or `got log --touching <name>`. |
| Success postcondition | `blame` reports the introducing and last-changing commits for the node; `log --touching` lists the branch commits that added, removed, or changed it. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer runs `got blame <name>`. System walks the current branch's commit ancestry chronologically and reports the commit that first contained the node (introduced) and the last commit whose content for the node differs from the prior one (last changed), each with author and message.
2. Developer runs `got log --touching <name>`. System lists the branch's commits whose structural diff against their parent added, removed, or changed the node (`graph.Diff`).

## Extensions

### Successful variations

- **1a. Never modified:** if a node was added and never changed, "last changed" equals "introduced".
- **2a. No touching commits:** `log --touching` prints nothing if the node was never involved.

### Failure paths

- **1b. Absent node:** `blame` of a node not present in the branch's history reports "not present".
- **\*a. No commits:** `blame` on a branch with no commits reports an error.

## Sub-variations

- **Better than git:** blame here is **per node**, computed from the commit DAG's snapshots — it names the exact commit that introduced or changed a specific entity, rather than git's per-line, rename-guessing heuristic. `log --touching` is the node-level analogue of `git log -- <path>`.

## Related use cases

- Uses: UC-S26 (commit DAG / ancestry), UC-S27 (structural diff), UC-U22 (commit metadata).
