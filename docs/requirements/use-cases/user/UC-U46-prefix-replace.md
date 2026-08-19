# UC-U46: Prefix replacement and N-level re-nest

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`rename.go` prefix strip+add in `matchDirRenames` / `matchFlattenRenest`; pre-pass in `reconcileRenames`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: a tree that replaces one common prefix with another, or gains two or more extra dest parents on every path, should still carry the other side's per-file edit to the new paths. Never invent a mapping on a split, and never pair a path that remained on the renaming side. |
| Preconditions | UC-U45's flatten and one-level re-nest mapping are in place. One side deleted a tree of paths and added them under a replaced prefix or an N-level re-nest; the other side edited a surviving path. Flatten / one-level re-nest voting does not recover `pkg/…` → `lib/…` (drop and add) or `pkg/…` → `vendor/lib/pkg/…` (two or more extra parents). |
| Trigger | `got merge <branch>` after UC-U45's flatten / one-level re-nest mapping leaves a prefix replacement or N-level re-nest unmatched. |
| Success postcondition | A unique prefix replacement or N-level re-nest relocates unmatched files by the recovered mapping so an edit still merges at the new path. The result is still validity-gated. |
| Failure postcondition | An ambiguous prefix-replace / N-level split is not treated as a tree rename; a low-similarity same-basename dest is not a rename; nothing invalid is committed. |

## Main success scenario

1. Developer runs `got merge <branch>`. Contested files still go through the
   structural merge, the line-level diff3 fallback (UC-U42), adjacent hunk
   refinement, content-similarity rename pairing (UC-U43), basename-weighted
   / immediate-parent directory mapping (UC-U44), and flatten / one-level
   re-nest (UC-U45).
2. After those passes, System looks at each non-root source prefix that has
   at least two deleted files under it (`matchFlattenRenest`). Already-matched
   pairs vote, and unmatched files vote only when exactly one unused added
   path shares their basename. A unique strict majority for one
   `(oldPrefix → newPrefix)` strip+add is a tree rename:
   dest = `join(newPrefix, rel(old, oldPrefix))`.
   - **flatten** (UC-U45): newPrefix empty (`pkg/a.go` + `pkg/sub/b.go` →
     `a.go` + `sub/b.go`);
   - **one-level re-nest** (UC-U45): oldPrefix empty and newPrefix one
     component (`pkg/a.go` + `pkg/sub/b.go` → `lib/pkg/a.go` +
     `lib/pkg/sub/b.go`);
   - **N-level re-nest:** oldPrefix empty and newPrefix two or more
     components (`pkg/a.go` + `pkg/sub/b.go` → `vendor/lib/pkg/a.go` +
     `vendor/lib/pkg/sub/b.go`);
   - **prefix replacement:** both prefixes non-empty (`pkg/a.go` +
     `pkg/sub/b.go` → `lib/a.go` + `lib/sub/b.go`).
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

- **2a. N-level re-nest plus edit:** every dest is the original path under
  two or more extra parents. A file whose per-file match ties — typically
  a competing same-basename add — follows that extra prefix; an edit of
  that file on the other side merges at the re-nested path.
- **2b. Prefix replacement plus edit:** files under a source prefix land
  at the path with that prefix replaced by a new prefix (including nested
  relatives). An unmatched file follows the replacement; an edit merges
  at the new path.

### Failure paths

- **2c. Ambiguous prefix-replace / N-level split:** files from one source
  prefix land under two or more transforms (or two dests of the same kind)
  with no unique strict majority. System does not invent a winner;
  unmatched paths stay delete+add / modify-delete.
- **3a. Low-similarity same basename:** a prefix replacement or N-level
  re-nest proposes a dest that shares the basename but content similarity
  is too low for the combined score to meet 3/5 — not a rename. A
  concurrent edit of the old path remains a modify/delete conflict.
- **4a. Invalid result:** a rename-relocated merge that would produce
  invalid Go is refused by the language gate; a result that does not
  type-check is refused by the semantic gate (UC-U40).

## Sub-variations

- **Layering:** structural first (UC-U36/U37/U41), then coarse diff3
  (UC-U42), then hunk refinement and content pairing (UC-U43), then
  basename weighting and immediate-parent directory mapping (UC-U44), then
  flatten / one-level re-nest (UC-U45), then this general prefix strip+add.
  Each layer only accepts a result the next gate will still validate.
  `--ours`/`--theirs` still resolve genuine remaining conflicts.
- **Transforms:** dest = `join(newPrefix, rel(old, oldPrefix))`. Flatten
  and one-level re-nest are the empty-prefix special cases already
  recovered by UC-U45; this use case is the general unique strip+add.
  Immediate-parent sibling moves stay UC-U44.
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
- **Arbitrary reshapes that are not a unique prefix strip+add stay out.**
  File-order reshuffles and sibling swaps that do not share one
  `(oldPrefix → newPrefix)` map still need a unique per-file match.
- Limits inherited from UC-U43, UC-U44, and UC-U45 still apply (divergent
  rename destinations are not unified; large-file bag overlap is
  reorder-insensitive; same-region overlaps still conflict; near-miss
  floor is 1/2).

## Related use cases

- Extends: UC-U45 (flatten and one-level re-nest — this generalizes that
  vote to a unique `(oldPrefix → newPrefix)` strip+add). Uses: UC-U40
  (semantic gate), UC-U32 (`--ours`/`--theirs`).
