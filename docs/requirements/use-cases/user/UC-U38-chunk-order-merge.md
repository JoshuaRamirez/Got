# UC-U38: Preserve chunk order across a merge

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`chunkmerge.go`: `mergeChunkOrder` and order-aware reassembly) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: when one branch reorders chunks in a file and the other edits them, keep the reorder — never silently drop it. |
| Preconditions | A contested file merges cleanly at the chunk-content level (UC-U36/UC-U37). |
| Trigger | `got merge <branch>` where one side reordered a file's chunks. |
| Success postcondition | The merged file's chunk order reflects the reordering side; concurrent content edits are applied; a two-sided incompatible reorder surfaces as a conflict rather than a silent choice. |
| Failure postcondition | The file is left to the file-level merge to flag; nothing is silently reordered or dropped. |

## Main success scenario

1. Developer runs `got merge <branch>`. The chunk merge (UC-U36) reconciles each
   contested file's chunk *content* through the graph engine.
2. Separately, System three-way merges the chunk *key sequence*
   (`mergeChunkOrder`), recomputed from the ordered `Split` outputs of
   base/ours/theirs — never from a per-chunk attribute.
3. If neither side reordered the shared chunks, System emits base order followed
   by each side's additions (identical to the content-only behavior). If exactly
   one side reordered, that side's sequence is the authority and the other side's
   additions are appended. If both reordered identically, System accepts it.
4. System reassembles the surviving chunks in the merged order and validity-gates
   the result.

## Extensions

### Successful variations

- **3a. Reorder + disjoint edit:** one side reorders while the other edits an
  unaffected chunk — both survive (the reorder is kept, the edit applied).
- **2a. Deletion:** a chunk deleted by the content merge simply drops out of the
  sequence.

### Failure paths

- **3b. Two-sided incompatible reorder:** if both sides reorder the shared chunks
  into different sequences, `mergeChunkOrder` returns no result, the chunk merge
  declines, and the file-level merge conflicts (resolvable with
  `merge --ours`/`--theirs`, UC-U32).

## Sub-variations

- **Order is not vertex state:** the merge order is recomputed from the ordered
  chunk slices at merge time. It is deliberately *not* stored on the chunk
  vertices, because `repo.MergeStates` compares whole attribute maps — an order
  attribute would turn an independent reorder into a false content conflict.

## Known limitations (honest scope)

- **Trailing-separator coupling (Go chunker):** because the Go chunker splices a
  declaration together with the blank line that follows it, moving a declaration
  to the *last* position changes its trailing whitespace. If the same
  declaration is edited on the other side, that surfaces as a conflict rather
  than merging. This is a byte-fidelity artifact of the parser-free splicing, not
  a silent loss — the reorder is surfaced, never dropped. Reorders that keep each
  moved declaration followed by another declaration merge cleanly.

## Related use cases

- Extends: UC-U36 (chunk-level merge), UC-U37 (Go declaration merge). Uses:
  UC-U32 (`--ours`/`--theirs`) for the residual two-sided-reorder conflict.
