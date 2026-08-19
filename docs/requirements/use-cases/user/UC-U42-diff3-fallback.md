# UC-U42: Fall back to a line-level merge

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`diff3.go`: `diff3`/`diff3Merge`; fallback in `reconcileFilesByChunk`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: a merge should be at least as good as git's line merge on every file — including non-Go text and Go changes the structural chunker is too coarse for. |
| Preconditions | A file changed on both sides and the structural merge did not reconcile it. |
| Trigger | `got merge <branch>` where the structural chunk merge conflicts or is inapplicable. |
| Success postcondition | Non-conflicting line-level changes merge (git parity); the result is still validity-gated. |
| Failure postcondition | Genuinely overlapping changes conflict; nothing invalid is committed. |

## Main success scenario

1. `got merge <branch>` reconciles each contested file. For a Go file it first
   tries the structural (symbol/statement) merge (UC-U37/UC-U41).
2. If the structural merge does not apply (a non-Go file) or conflicts (e.g. a
   statement inserted on one side shifts the positional alignment), System runs a
   line-level three-way merge (`diff3`): stable lines matched in all three
   versions anchor the merge; each region between anchors is taken from whichever
   side changed it, or accepted if both changed it identically.
3. If every region is non-conflicting, System validity-gates the merged text
   (`diff3Merge` → the same language gate) and, if it passes, uses it.
4. The whole-package semantic gate (UC-U40) still runs on the assembled merge.

## Extensions

### Successful variations

- **2a. Non-Go files:** any text file gets git-quality line merging — the block
  chunker's content-keyed structural merge is *not* used for non-Go text (it
  would reorder edited lines), so `diff3` is the merge.
- **2b. Intra-function insert/delete:** a statement added or removed on one side
  while a distant statement is edited on the other merges here, where the
  positional structural merge (UC-U41) conflicts on the shift.

### Failure paths

- **2c. Overlapping change:** both sides change the same lines differently —
  a conflict, exactly as git reports (resolvable with `merge --ours`/`--theirs`,
  UC-U32). Adjacent but non-overlapping edits (an insertion immediately next
  to the other side's edit) are refined by UC-U43 rather than left as a
  conflict here.
- **3a. Invalid result:** a line merge that would produce invalid Go (e.g. a
  duplicate declaration) is refused by the gate.

## Sub-variations

- **Layering:** structural first (for the graph advantages — import union,
  symbol identity, both-add-at-same-location, semantic validity), then diff3 (for
  git-parity textual merging), then a real conflict. Got is therefore never
  *worse* than git at text merging, and better where structure or types help.

## Known limitations (honest scope)

- **Adjacent changes and renames:** coarse diff3 still treats an insertion
  immediately next to the other side's edit as one conflict region, and it
  does not pair a path change with an edit of the old path. UC-U43 closes
  both: hunk refinement of non-overlapping adjacent edits, and
  similarity-based rename/move detection. Remaining limits live in UC-U43.

## Related use cases

- Extends: UC-U36 (chunk merge), UC-U41 (intra-function merge — diff3 covers the
  insert/delete cases positional keying cannot). Extended by: UC-U43 (adjacent
  hunk refinement and rename detection). Uses: UC-U40 (semantic gate
  still validates the result), UC-U32 (`--ours`/`--theirs`).
