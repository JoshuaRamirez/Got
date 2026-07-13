# UC-U36: Merge a file at chunk granularity

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`chunk.go` block chunker, `chunkmerge.go` reconcile pre-pass in `merge`) over `repo.MergeStates` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: merge two branches that changed the *same file* in different places without a manual conflict, when the changes are structurally disjoint. |
| Preconditions | Both branches have committed a file at the same path, both changed it relative to the merge base, and the two versions differ. |
| Trigger | `got merge <branch>` where a contested file's changes are chunk-disjoint. |
| Success postcondition | The file merges cleanly: both sides' chunk-level changes are present; no conflict is raised for that file. |
| Failure postcondition | If the two sides changed the same chunk differently, the file still conflicts (raised by the file-level merge); nothing is silently dropped. |

## Main success scenario

1. Developer runs `got merge <branch>`. Before the file-level three-way merge,
   a pre-pass (`reconcileFilesByChunk`) examines every file changed on both
   sides relative to the base.
2. For each such file, System decomposes the base/ours/theirs versions into
   chunks (`blockChunker`: top-level brace blocks and standalone lines, each
   keyed by its signature line so a body edit keeps the chunk's identity).
3. System runs the three chunk sets through the *same* graph three-way merge
   engine used for vertices (`repo.MergeStates` — the composition pushout), at
   chunk granularity.
4. If the chunk merge is clean, System rewrites both sides' file content to the
   reassembled merged file, so the file-level merge now sees the two sides
   agree and raises no conflict.
5. System records the merge commit; `got extract` renders the merged file.

## Extensions

### Successful variations

- **3a. Same-location additions:** two branches each add a new block (function,
  import, method) at the same point get distinct chunk keys, so both are taken —
  the case git's line merge reports as a conflict on the shared insertion point.
- **1a. One-sided files:** a file changed on only one side is taken as-is by the
  file-level merge; the pre-pass ignores it.

### Failure paths

- **3b. Same-chunk divergence:** if both sides changed the *same* chunk in
  different ways, the chunk merge reports a conflict and the pre-pass leaves the
  file untouched; the file-level merge then conflicts (resolvable with
  `merge --ours`/`--theirs`, UC-U32). Nothing is auto-merged.

## Sub-variations

- **Engine reuse, not diff3:** the chunk merge is the vertex merge applied to
  throwaway chunk vertices — the same associativity/typed-conflict machinery,
  one level finer. No line-based patching is involved.

## Known limitations (honest scope)

- **Where it helps vs. git:** git already merges *disjoint body edits* to
  different functions cleanly. The demonstrated win is **same-location
  structural additions** (both sides append/insert a new block), which git
  conflicts on. Reorders and moves would also merge once the chunker tracks
  identity across position — the block chunker keys by signature, so a moved
  block already aligns, but insertion *ordering* when both sides insert at the
  same point is a deterministic-but-arbitrary heuristic (base order, then each
  side's additions).
- **Parser-free tier:** brace counting ignores braces inside strings/comments,
  and a chunk is a top-level block, not a semantic unit. A language-aware
  chunker (`go/ast`, or tree-sitter for many languages) would slot behind the
  same `chunker` interface for finer, semantics-correct granularity.

## Related use cases

- Extends: UC-U24 (semantic merge), UC-U35 (version real files). Uses:
  UC-U32 (`--ours`/`--theirs` for the residual same-chunk conflicts).
