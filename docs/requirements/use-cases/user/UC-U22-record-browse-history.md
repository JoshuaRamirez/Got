# UC-U22: Record and browse repository history

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo` (via `Commit`, `LoadHistory`, `SaveHistory`) + `cmd/got` (`commit`, `log`) |
| Primary actor | Author / Integrator |
| Stakeholders & interests | Author: record the current state as a commit and see the history of a branch. Auditor: recover the operation behind each commit and walk ancestry — non-lossily, unlike git. |
| Preconditions | An initialized repository directory. |
| Trigger | Actor commits the current state, or asks for a branch's log. |
| Success postcondition | A content-addressed commit recording the current graph state and its operation delta is appended to the history DAG and persisted; the branch's commit pointer advances to it. `log` prints the branch's ancestry newest-first. |
| Failure postcondition | An error is returned; the history is unchanged. |

## Main success scenario

1. Actor mutates the graph (`add-vertex`, `add-edge`, …), which updates `graph.json`.
2. Actor runs `got commit -m <message> [--branch <name>] [--actor <name>]`.
3. System loads the branch's current commit (the `commit:<branch>` namespace ref) as the parent, snapshots the current graph, computes the operation delta (vertices added/removed since the parent), and records a new commit in the log (`repo.Commit`, UC-S26).
4. System persists the log to `history.json` and advances the branch's commit pointer to the new commit.
5. Actor runs `got log [<branch>]`; System walks the branch head's ancestry (`history.Log.Ancestors`) and prints each commit — short id, author, message — newest first.

## Extensions

### Successful variations

- **3a. Root commit:** the first commit on a branch has no parent; its delta is the full vertex set produced.
- **3b. Merge commit:** a commit may be given multiple parents; ancestry reaches shared history once.
- **5a. No commits yet:** `log` on a branch with no commit pointer reports "no commits".

### Failure paths

- **2a. Missing message:** `commit` without `-m` exits non-zero.
- **3c. Unknown parent commit:** if the recorded parent is absent from the log, `repo.Commit` returns `history.ErrUnknownCommit`.
- **\*a. Before init:** any command before `got init` reports "run 'got init'".

## Sub-variations

- **Non-lossy history:** each commit stores the operation delta (`Consumed`/`Produced`) alongside the resulting snapshot, so the history records *how* the state was reached, not only the bytes — the git-loses-information critique addressed.
- **Branch commit pointer:** a branch's advancing commit tip is kept as the `commit:<branch>` namespace ref (a `CommitID` round-tripped through the 32-byte ref value), distinct from the branch's vertex tip (UC-U21). This is the one mutable, advancing pointer.

## Related use cases

- Includes: UC-S26 (operation-first commit DAG), UC-S23 (snapshot codec).
- Complements: UC-U21 (first-class branches) — `branch-log` shows fork ancestry, `log` shows commit ancestry.
