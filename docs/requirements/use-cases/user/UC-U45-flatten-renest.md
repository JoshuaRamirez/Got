# UC-U45: Flatten and re-nest directory mapping

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`rename.go` flatten / one-level re-nest in `matchDirRenames` / `matchFlattenRenest`; pre-pass in `reconcileRenames`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: a tree that drops one common prefix (flatten) or gains one extra parent on every dest (re-nest) should still carry the other side's per-file edit to the new paths. Never invent a mapping on a split, and never pair a path that remained on the renaming side. |
| Preconditions | UC-U44's basename-weighted scoring and immediate-parent directory mapping are in place. One side deleted a tree of paths and added them flattened or re-nested; the other side edited a surviving path. Immediate-parent voting does not recover `srcdir/basename → destdir/basename` (empty dest parent, or files under different immediate parents). |
| Trigger | `got merge <branch>` after UC-U44's unique-majority immediate-parent mapping leaves a flatten or one-level re-nest unmatched. |
| Success postcondition | A unique flatten or one-level re-nest relocates unmatched files by the recovered mapping so an edit still merges at the new path. The result is still validity-gated. |
| Failure postcondition | An ambiguous flatten/re-nest split is not treated as a tree rename; a low-similarity same-basename dest is not a rename; nothing invalid is committed. |

## Main success scenario

1. Developer runs `got merge <branch>`. Contested files still go through the
   structural merge, the line-level diff3 fallback (UC-U42), adjacent hunk
   refinement, content-similarity rename pairing (UC-U43), and
   basename-weighted / immediate-parent directory mapping (UC-U44).
2. After those passes, System looks at each non-root source prefix that has
   at least two deleted files under it (`matchFlattenRenest`). Already-matched
   pairs vote, and unmatched files vote only when exactly one unused added
   path shares their basename. A unique strict majority for one transform
   is a tree rename:
   - **flatten:** dest is the path with that common source prefix dropped
     (`pkg/a.go` + `pkg/sub/b.go` → `a.go` + `sub/b.go`);
   - **one-level re-nest:** dest is one extra parent plus the original path
     (`pkg/a.go` + `pkg/sub/b.go` → `lib/pkg/a.go` + `lib/pkg/sub/b.go`).
3. Remaining unmatched files pair by that transform when the dest path
   exists as an add and the pair still meets the combined similarity gate
   (content `shared/max(n,m)` plus 1/10 basename bonus, threshold 3/5).
4. A 1–1 pair still runs the ordinary content merge (structural, then
   diff3, then hunk refinement) at the new path. The per-file language gate
   and the whole-package semantic gate (UC-U40) still run.
5. On a clean result System records the merge commit; `got extract` renders
   the merged files at their new paths.

## Extensions

### Successful variations

- **2a. Flatten plus edit:** files under a base prefix land at the path
  with that prefix dropped (including nested relatives). A file whose
  per-file match ties — typically a competing same-basename add — follows
  the flatten; an edit of that file on the other side merges at the
  flattened path.
- **2b. One-level re-nest plus edit:** every dest is the original path
  under one extra parent. An unmatched file follows that extra parent; an
  edit merges at the re-nested path.

### Failure paths

- **2c. Ambiguous flatten / re-nest split:** files from one source prefix
  land under two or more transforms (or two dests of the same kind) with
  no unique strict majority. System does not invent a winner; unmatched
  paths stay delete+add / modify-delete.
- **3a. Low-similarity same basename:** a flatten or re-nest proposes a
  dest that shares the basename but content similarity is too low for the
  combined score to meet 3/5 — not a rename. A concurrent edit of the old
  path remains a modify/delete conflict.
- **4a. Invalid result:** a rename-relocated merge that would produce
  invalid Go is refused by the language gate; a result that does not
  type-check is refused by the semantic gate (UC-U40).

## Sub-variations

- **Layering:** structural first (UC-U36/U37/U41), then coarse diff3
  (UC-U42), then hunk refinement and content pairing (UC-U43), then
  basename weighting and immediate-parent directory mapping (UC-U44), then
  this flatten / one-level re-nest. Each layer only accepts a result the
  next gate will still validate. `--ours`/`--theirs` still resolve genuine
  remaining conflicts.
- **Transforms:** flatten drops one common prefix of any depth
  (`src/pkg/...` → `...`). Re-nest adds exactly one extra parent
  (`pkg/...` → `lib/pkg/...`). Immediate-parent sibling moves stay UC-U44.
- **Directory mapping:** only non-root prefixes; at least two deleted
  files under that prefix; unique destination transform with
  `votes > n/2`. Destination collision (two olds proposing the same new
  path, or a dest already taken by per-file matching) refuses that dest.
  Each directory-proposed pair still has to meet the combined similarity
  gate. Root-level files rely on per-file + basename matching only.
- **Similarity:** same combined score as UC-U43/U44. Empty files are
  never matched. Ties on the combined score still refuse.

## Known limitations (honest scope)

- **No copy detection:** a path that remains on the renaming side while a
  similar path is added is not paired. Copies stay as independent adds.
- **Deeper prefix replacement and N-level re-nest are UC-U46.** Arbitrary
  reshapes that are not a unique prefix strip+add still need a unique
  per-file match.
- Limits inherited from UC-U43 and UC-U44 still apply (divergent rename
  destinations are not unified; large-file bag overlap is
  reorder-insensitive; same-region overlaps still conflict; near-miss
  floor is 1/2).

## Related use cases

- Extends: UC-U44 (basename-weighted scoring and immediate-parent
  directory mapping — this recovers flatten and one-level re-nest).
  Extended by: UC-U46 (prefix replacement / N-level re-nest). Uses:
  UC-U40 (semantic gate), UC-U32 (`--ours`/`--theirs`).
