# UC-S26: Record operation-first commit history

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `history` (via `Commit`, `Log`, `NewCommit`, `Marshal`/`Unmarshal`) |
| Primary actor | `history.Log` |
| Stakeholders & interests | A repository host: keep a non-lossy, walkable history of how the graph reached each state. Auditor: recover the operation (not just the resulting bytes) behind every change, and the ancestry of any state. |
| Preconditions | A resulting graph snapshot (UC-S23) and the parent commit ids. |
| Trigger | A caller records a commit, walks ancestry, or persists/loads the log. |
| Success postcondition | The commit is content-addressed and appended to the DAG; its ancestry is walkable; the log round-trips through JSON. |
| Failure postcondition | An error is returned for an unknown parent or an unknown commit. |

## Main success scenario

1. Caller builds a `Commit` with `NewCommit(parents, message, actor, consumed, produced, snapshot)`.
2. System computes a content-addressed `CommitID` from the parents, message, actor, and the resulting state's element ids — so equal commits share an id (`history.computeID`).
3. Caller appends the commit with `Log.Add`; System verifies every parent is already present.
4. Caller walks history with `Log.Ancestors(id)` — the commit first, then its ancestors in breadth-first order, deduped across merge parents.
5. Caller persists the log with `Marshal` and later restores it with `Unmarshal`.

## Extensions

### Successful variations

- **1a. Root commit:** a commit with no parents begins a history.
- **1b. Merge commit:** a commit with two (or more) parents joins branches; `Ancestors` reaches shared history once (deduped).
- **2a. Operation is annotation, not identity:** `Consumed`/`Produced` record the delta (the non-lossy "how"), but do not affect the `CommitID` — two commits reaching the same state from the same parents are the same commit (mirrors git: `id = hash(tree, parents, author, message)`).

### Failure paths

- **3a. Unknown parent:** `Log.Add` on a commit whose parent is absent returns `history.ErrUnknownParent`.
- **4a. Unknown commit:** `Log.Ancestors` of an id not in the log returns `history.ErrUnknownCommit`.
- **5a. Corrupt log:** `Unmarshal` of malformed JSON or a bad id returns an error.

## Sub-variations

- **Non-lossy vs. git:** git records a snapshot and reconstructs intent heuristically; a `Commit` records the operation delta plus the resulting snapshot, so the "how" is preserved, not inferred.
- **Storage note:** each commit currently carries a full `graph.Snapshot`; deduplicating shared state (git's tree/blob sharing) is a future optimization, not required for correctness.

## Related use cases

- Includes: UC-S23 (graph snapshot codec) — the resulting state each commit carries.
- Included by: UC-U22 (record and browse repository history), which advances branch tips (UC-U21) along commits.
