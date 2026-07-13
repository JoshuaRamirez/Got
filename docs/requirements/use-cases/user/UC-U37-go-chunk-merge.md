# UC-U37: Merge Go files by declaration, and refuse structurally invalid results

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`gochunk.go`: Go AST chunker + validity gate; `chunkmerge.go`: chunker selection + gate) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: merge Go files at real declaration granularity, and never have a merge silently produce a file that does not compile. |
| Preconditions | The contested file's path ends in `.go` and parses. |
| Trigger | `got merge <branch>` where a `.go` file changed on both sides. |
| Success postcondition | Disjoint declaration changes merge cleanly; a merge that would introduce a duplicate top-level symbol is refused and surfaced as a conflict instead. |
| Failure postcondition | The file is left to the file-level merge to flag; nothing invalid is committed. |

## Main success scenario

1. Developer runs `got merge <branch>`. For each contested `.go` file, the
   chunk-merge pre-pass (UC-U36) selects the Go chunker (`chunkerFor`).
2. System parses each version with `go/ast` and chunks it at top-level
   declaration boundaries, keyed by symbol (`func:Name`, `method:Type.Name`,
   `type:`/`var:`/`const:` name). Content is spliced from the original bytes, so
   Split→Join is verbatim (no reformatting).
3. System runs the chunk sets through the graph three-way merge engine
   (`repo.MergeStates`).
4. System reassembles the merged declarations and runs the structural-validity
   gate (`goValidityOK`): the result must parse and declare no top-level symbol
   twice.
5. If the gate passes, System rewrites both sides to the merged file, so the
   file-level merge sees agreement and the file merges cleanly.

## Extensions

### Successful variations

- **2a. Body/format immunity:** because a declaration is keyed by its symbol, an
  edit or reindent of a function body keeps it aligned; a change on only one side
  is taken.
- **2b. Parse fallback:** a file that does not parse as Go degrades to the
  block chunker (UC-U36); it never fails hard.

### Failure paths

- **4a. Structural-validity gate:** if the merged result would not parse, or
  would declare a top-level symbol twice (e.g. one side adds `func Size`, the
  other adds `var Size` in a different place — a clean chunk merge, distinct
  keys), the gate rejects it. The pre-pass leaves the file untouched and the
  file-level merge conflicts, resolvable with `merge --ours`/`--theirs`
  (UC-U32). This is the case git merges silently into code that does not compile.

## Sub-variations

- **Method namespacing:** two types may declare a method of the same name; the
  gate keys methods by receiver type, so they do not count as a collision.
- **Whole-result check:** the gate inspects the merged file as a whole, not
  chunk by chunk — it catches collisions between chunks that merged
  independently and cleanly. git has no parser and cannot make this check at all.

## Related use cases

- Extends: UC-U36 (chunk-level merge — this supplies the Go chunker behind the
  same `chunker` interface, plus the validity gate). Uses: UC-U32
  (`--ours`/`--theirs` for the refused merges), UC-U35 (versioned files).
