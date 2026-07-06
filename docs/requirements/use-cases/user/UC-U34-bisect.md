# UC-U34: Bisect history to find the first bad commit

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`bisect start`/`good`/`bad`/`run`/`reset`/`status`) over `history` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: locate the exact commit at which a property flipped from good to bad, in `O(log n)` tests rather than checking every commit. |
| Preconditions | The repository is initialized and the good commit is an ancestor of the bad commit. |
| Trigger | `got bisect start <bad> <good>`. |
| Success postcondition | The first bad commit is reported; the working graph and starting branch are restorable with `bisect reset`. |
| Failure postcondition | An error is returned; no bisect session is started, or the in-progress session is left unchanged. |

## Main success scenario

1. Developer runs `got bisect start <bad> <good>`. System resolves both
   commit-ishes, verifies `good` is an ancestor of `bad`, records the session,
   and checks out the middle suspect's snapshot into the working graph.
2. Developer inspects the working graph (or runs a test) and reports the verdict
   with `got bisect good` or `got bisect bad`.
3. System narrows the suspect set — ancestors of `bad` that are not ancestors of
   `good`, minus `bad` — and checks out the next optimal candidate (the one whose
   in-set ancestor count most evenly splits the remaining suspects).
4. Steps 2–3 repeat until the suspect set is empty. System reports the current
   `bad` as the first bad commit.
5. Developer runs `got bisect reset`; System restores the working graph to the
   branch the session started on and clears the session.

## Extensions

### Successful variations

- **2a. Automated run:** `got bisect run <cmd> [args...]` drives the loop: for
  each candidate it runs the command in the repo (exit 0 = good, non-zero = bad)
  and advances automatically until the first bad commit is found.
- **1a. Status:** `got bisect status` reports the current boundary and the
  candidate under test; with no session it prints `no bisect in progress`.

### Failure paths

- **1b. Not an ancestor:** if `good` is not an ancestor of `bad`, the session is
  refused.
- **1c. Same commit / unknown ref:** `bad == good`, or an unresolvable
  commit-ish, is refused.
- **2b. No session:** `good`/`bad`/`run`/`reset` with no session in progress
  reports `no bisect in progress` and fails.

## Sub-variations

- **Detached working graph:** unlike git, bisect does not move HEAD or any branch
  pointer; it only restores each candidate's snapshot to the working graph, so
  branch tips are untouched and `bisect reset` simply reloads the origin branch.
- **DAG-correct suspect set:** the suspect set is computed from ancestor sets
  (`history.Log.Ancestors`), so merges in the range are handled — not just linear
  history.

## Related use cases

- Uses: UC-U22 (commit), UC-S26 (commit history DAG / ancestors), UC-U20
  (repository state). Complements UC-U33 (reflog) for post-hoc investigation.
