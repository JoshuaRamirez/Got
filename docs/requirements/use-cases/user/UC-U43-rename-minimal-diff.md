# UC-U43: Refine adjacent edits and detect renames

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`diff3.go` hunk refinement; `rename.go` similarity match; pre-pass in `reconcileFilesByChunk`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: a merge should not conflict on adjacent but non-overlapping line edits, and a file renamed or moved on one side should still receive the other side's content edit at the new path. |
| Preconditions | A file changed on both sides, or one side deleted a base path and added another while the other side edited the base path; the structural merge (if any) and coarse diff3 did not already reconcile it. |
| Trigger | `got merge <branch>` after UC-U42's structural-then-diff3 pipeline still sees a conflict that this refinement can shrink, or a delete+add that is the same file. |
| Success postcondition | Non-overlapping adjacent line edits merge; a detected rename carries the three-way content merge to the new path; the result is still validity-gated. |
| Failure postcondition | A same-region overlap still conflicts; a low-similarity delete+add is not treated as a rename; nothing invalid is committed. |

## Main success scenario

1. Developer runs `got merge <branch>`. Contested files still go through the
   structural merge and then the line-level diff3 fallback (UC-U42).
2. When a diff3 region between two three-way anchors would conflict, System
   computes pairwise edit hunks of that region against the base (`refineGap`).
   Hunks whose base ranges do not overlap are applied independently: an
   insertion immediately next to the other side's edit is taken together with
   that edit, not reported as one conflict.
3. Separately, for a path present in the base but missing on one side, System
   looks for a path added on that side whose line-LCS similarity to the base
   file is at least the match threshold (`matchRenames`). A 1–1 unique-best
   pair is treated as a rename: the content merge (structural, then diff3,
   then step 2) runs against the surviving side and the result is placed at
   the new path on both sides so the file-level merge no longer sees an
   unrelated delete+add.
4. The per-file language gate (`diff3Merge` / `chunkMerge`) and the
   whole-package semantic gate (UC-U40) still run on the assembled merge.
5. On a clean result System records the merge commit; `got extract` renders
   the merged files (the renamed file at its new path).

## Extensions

### Successful variations

- **2a. Adjacent insert versus edit:** one side inserts a line immediately
  before or after a line the other side edited; the hunks do not share a
  base line, so both apply.
- **3a. Rename plus edit:** one side moves or renames a file (old path gone,
  new path added) while the other side edits the old path; the merged
  content is stored at the new path and the old path is dropped.
- **3b. Both sides rename to the same path:** both sides delete the old path
  and add the same new path with disjoint content edits; those edits merge
  at the new path (the file is not in the base under the new name, so the
  ordinary same-path pre-pass would have skipped it).

### Failure paths

- **2b. Overlapping change:** both sides change the same base line(s)
  differently, or both insert different text at the same point — a conflict
  (resolvable with `merge --ours`/`--theirs`, UC-U32).
- **3c. Unrelated delete+add:** the added file's line-LCS similarity to the
  deleted base file is below the threshold, or two added files tie for best
  match — not a rename. A concurrent edit of the old path remains a
  modify/delete conflict.
- **3d. Divergent rename:** both sides move the same base file to *different*
  new paths. System does not pick a winner; the two adds stand and the old
  path is gone on both sides.
- **4a. Invalid result:** a refined or rename-relocated merge that would
  produce invalid Go is refused by the language gate; a result that does
  not type-check is refused by the semantic gate (UC-U40).

## Sub-variations

- **Layering:** structural first (UC-U36/U37/U41), then coarse diff3
  (UC-U42), then hunk refinement, then rename pairing. Each layer only
  accepts a result the next gate will still validate. `--ours`/`--theirs`
  still resolve genuine remaining conflicts.
- **Similarity:** a rename matches only when `shared / max(n, m)` is at
  least 3/5 and the pair is each side's unique best (compared as integers
  so 3/5 and 6/10 tie). Empty files are never matched. Destination
  collision (the new path already exists on the other side as a different
  file) is refused. Pairs whose line-count product exceeds 2e6 are scored
  by bag-of-lines overlap instead of LCS so scoring stays linear-memory.

## Known limitations (honest scope)

- **Not a git-identical rename detector:** there is no copy detection.
  Basename-weighted scoring and directory-as-a-unit moves are UC-U44.
  Flatten and one-level re-nest mapping are UC-U45. Prefix replacement
  and N-level re-nest are UC-U46. Remaining ties and sub-threshold pairs
  (after those refinements) are left as delete+add.
- **Large-file scoring:** when `n×m > 2e6` lines, similarity is bag-of-lines
  overlap (reorder-insensitive; can score a reshuffle higher than LCS).
  Diff3's merge-time LCS is unchanged and still quadratic-memory per file.
- **Same-point divergent inserts still conflict:** two different insertions
  at the same base index overlap as competing inserts.
- **LCS mis-alignment:** repeated identical lines can align the wrong
  counterparts, same as the existing diff3 (never silently overlapping).
- **Divergent destinations are not unified** (extension 3d): both new paths
  survive; System does not invent a single winner.

## Related use cases

- Extends: UC-U42 (diff3 fallback — this refines the adjacent-conflict and
  missing-rename limits). Extended by: UC-U44 (basename-weighted scoring
  and directory-as-a-unit move), UC-U45 (flatten / one-level re-nest),
  UC-U46 (prefix replacement / N-level re-nest). Uses: UC-U40 (semantic
  gate), UC-U32 (`--ours`/`--theirs`).
