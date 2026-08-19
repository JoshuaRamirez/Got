# UC-U44: Weight basename and detect directory moves

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`rename.go` basename-weighted scoring and directory-as-a-unit mapping; pre-pass in `reconcileRenames`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: a same-basename path change (`src/foo.go` → `pkg/foo.go`) should still uniquely match when content similarity alone would tie or miss; a directory moved as a unit should carry the other side's per-file edit to the new paths. Never silently pair unrelated files that merely share a name. |
| Preconditions | UC-U43's rename pre-pass is in place. One side deleted a base path (or a tree of paths) and added another; the other side edited a surviving path. Content similarity alone does not uniquely pair at the 3/5 threshold, or identical files under one directory would tie. |
| Trigger | `got merge <branch>` after UC-U43's unique-best content pairing leaves a same-basename move unmatched, or leaves a whole-directory move as per-file ties. |
| Success postcondition | A unique same-basename pair that meets the combined threshold merges at the new path; a unique-majority directory move relocates unmatched files by relative path so an edit still merges there. The result is still validity-gated. |
| Failure postcondition | A same-basename pair with low content similarity is not a rename; an ambiguous directory split is not treated as a tree rename; nothing invalid is committed. |

## Main success scenario

1. Developer runs `got merge <branch>`. Contested files still go through the
   structural merge, the line-level diff3 fallback (UC-U42), adjacent hunk
   refinement, and content-similarity rename pairing (UC-U43).
2. When scoring a delete+add pair, System adds a small basename bonus when
   the two paths share a final component (`renameScore`). Combined similarity
   is still compared as a rational against the 3/5 threshold. A unique
   same-basename candidate can therefore win a content-score tie, or uniquely
   match when content alone sits just under 3/5 (`matchRenames` /
   `uniqueBest`).
3. After per-file pairing, System looks at each non-root source directory
   whose files disappeared on that side (`matchDirRenames`). If a unique
   strict majority of those files land under one destination directory
   (matched pairs vote by their dest parent; unmatched files vote only when
   exactly one added path shares their basename), that mapping is a tree
   rename. Remaining unmatched files pair as `srcdir/rel` → `destdir/rel`
   when the dest path exists as an add and the pair still meets the
   similarity gate.
4. A 1–1 pair still runs the ordinary content merge (structural, then
   diff3, then hunk refinement) at the new path. The per-file language gate
   and the whole-package semantic gate (UC-U40) still run.
5. On a clean result System records the merge commit; `got extract` renders
   the merged files at their new paths.

## Extensions

### Successful variations

- **2a. Basename breaks a content tie:** two added files have the same
  line-LCS score at or above 3/5; exactly one shares the deleted file's
  basename, so that one is the unique match.
- **2b. Basename rescues a near miss:** content similarity is below 3/5 but
  at least 1/2, the added path uniquely shares the basename, and the
  combined score meets 3/5 — treated as a rename.
- **3a. Directory move plus edit:** most files under a base directory land
  under one new directory (preserving relative names). A file whose
  per-file match ties — typically a competing same-basename add — follows
  the majority dest; an edit of that file on the other side merges at the
  new path.

### Failure paths

- **2c. Low-similarity same basename:** two paths share a basename but
  content similarity is too low for the combined score to meet 3/5 — not a
  rename. A concurrent edit of the old path remains a modify/delete
  conflict.
- **3b. Ambiguous directory split:** files from one source directory land
  under two or more destinations with no unique strict majority. System
  does not invent a winner; unmatched paths stay delete+add /
  modify-delete.
- **4a. Invalid result:** a rename-relocated merge that would produce
  invalid Go is refused by the language gate; a result that does not
  type-check is refused by the semantic gate (UC-U40).

## Sub-variations

- **Layering:** structural first (UC-U36/U37/U41), then coarse diff3
  (UC-U42), then hunk refinement and content pairing (UC-U43), then this
  basename weighting and directory mapping. Each layer only accepts a
  result the next gate will still validate. `--ours`/`--theirs` still
  resolve genuine remaining conflicts.
- **Basename scoring:** combined similarity is content `shared/max(n,m)`
  plus 1/10 when `path.Base` matches (uncapped, so a same-basename 1.0
  beats a different-basename 1.0). Threshold remains 3/5, so a
  same-basename pair needs content ≥ 1/2. Different-basename pairs still
  need content ≥ 3/5. LCS is skipped when even a perfect overlap cannot
  meet that floor (`pairScore`: 1/2 if basenames match, 3/5 otherwise).
  Empty files are never matched. Ties on the combined score still refuse.
- **Directory mapping:** only non-root parents; at least two deleted files
  from that parent; unique destination with `votes > n/2`. Destination
  collision (two olds proposing the same new path, or a dest already taken
  by per-file matching) refuses that dest. Each directory-proposed pair
  still has to meet the combined similarity gate. Root-level files rely on
  per-file + basename matching only.

## Known limitations (honest scope)

- **No copy detection:** a path that remains on the renaming side while a
  similar path is added is not paired. Copies stay as independent adds.
- **Directory mapping follows immediate parents:** a flatten or a
  one-level re-nest that does not preserve `parent/basename` is UC-U45,
  not this use case. Prefix replacement and N-level re-nest are UC-U46.
  Arbitrary reshapes that are not a unique prefix strip+add still need a
  unique per-file match.
- **Near-miss floor is 1/2:** a same-basename pair below 50% content
  similarity is refused even if a human would call it the same file.
- Limits inherited from UC-U43 still apply (divergent rename destinations
  are not unified; large-file bag overlap is reorder-insensitive;
  same-region overlaps still conflict).

## Related use cases

- Extends: UC-U43 (content-similarity rename pairing — this weights
  basename and adds directory-as-a-unit mapping). Extended by: UC-U45
  (flatten / one-level re-nest), UC-U46 (prefix replacement / N-level
  re-nest). Uses: UC-U40 (semantic gate), UC-U32 (`--ours`/`--theirs`).
