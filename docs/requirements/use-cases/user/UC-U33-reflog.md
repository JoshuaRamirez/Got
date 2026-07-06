# UC-U33: Review and recover ref movements with the reflog

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`reflog`, plus ref-move journaling in `commit`/`checkout`/`reset`/`merge`/`rebase`/`amend`/`revert`/`cherry-pick`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: see where HEAD and each branch pointed over time, and recover a commit that a history-rewriting operation (reset, rebase, amend) left unreferenced. |
| Preconditions | The repository is initialized. |
| Trigger | `got reflog [<ref>|--all]`. |
| Success postcondition | The journal of ref movements is printed newest-first; every prior tip — including commits no branch still points at — is listed with the action that moved the ref, so it stays reachable. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer performs ref-moving operations (`commit`, `checkout`, `reset`,
   `merge`, `rebase`, `amend`, `revert`, `cherry-pick`). Each appends an entry
   to the append-only reflog recording the ref, old tip, new tip, action, and
   message; operations on the current branch also mirror an entry under `HEAD`.
2. Developer runs `got reflog` to review activity.
3. System prints the `HEAD` entries newest-first, numbered `HEAD@{0}` (newest)
   upward, each as `<short-new> <ref>@{i}: <action>: <message>`.

## Extensions

### Successful variations

- **2a. Per-ref view:** `got reflog <branch>` filters to a single branch's tip
  movements, numbered `<branch>@{i}`.
- **2b. All refs:** `got reflog --all` interleaves every ref's movements in one
  chronological stream.
- **2c. Recovery:** after a `reset --hard` (or `rebase`/`amend`) drops a commit
  from a branch, the dropped commit's hash still appears in the reflog, so the
  developer can `reset`/`checkout` back to it by that id.

### Failure paths

- **1a. No repository:** running `reflog` before `got init` reports the
  `run 'got init'` hint and fails.
- **2d. Empty journal:** a ref with no recorded movements prints
  `(no reflog entries)` and succeeds.

## Sub-variations

- **Best-effort journaling:** a reflog write failure never aborts the operation
  that moved the ref — the journal is a convenience log, not part of the commit
  DAG's integrity.
- **HEAD vs branch:** unlike git, branch tips are namespace refs
  (`commit:<branch>`) and HEAD is a branch name; the reflog records both layers,
  commit-addressed, so a checkout between branches is journaled under `HEAD`
  with the old and new branch tips.

## Related use cases

- Uses: UC-U22 (commit), UC-U26 (reset/restore), UC-U31 (rebase), UC-U29
  (cherry-pick/amend). The reflog observes the ref movements these produce.
