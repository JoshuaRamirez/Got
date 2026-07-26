# UC-U39: Merge Go imports as a set, and gate on import names

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`gochunk.go`: import decomposition + `importLocalNames`; `chunkmerge.go`: `groupImportChunks`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: two branches each adding a different import should merge; a merge must not produce an import whose name collides with a package-scope declaration. |
| Preconditions | The contested file is `.go` and parses. |
| Trigger | `got merge <branch>` where both sides changed a Go file's imports. |
| Success postcondition | Distinct added imports are unioned into one valid import block; a merge whose import name duplicates another import or a package-scope symbol is refused. |
| Failure postcondition | The file is left to the file-level merge to flag; nothing invalid is committed. |

## Main success scenario

1. Developer runs `got merge <branch>`. For a `.go` file, the Go chunker
   decomposes a parenthesized `import (...)` block into a head (`import (`), one
   chunk per spec (keyed `import:<path>`), and a tail (`)`).
2. The graph three-way merge unions the specs: a spec added on only one side is
   an independent chunk addition, so two branches adding different imports do
   not conflict.
3. On reassembly, `groupImportChunks` keeps the decomposed import chunks
   contiguous and correctly bracketed (head, specs, tail), positioned where the
   block was — so an added spec lands inside the parens, not after them.
4. The validity gate includes each import's bound name (alias, or the package
   name), so the merged file is rejected if an import name is duplicated or
   collides with a package-scope declaration.

## Extensions

### Successful variations

- **2a. Same import both sides:** identical spec chunks (same key, same content)
  merge to one — no duplicate.
- **1a. Single-line import:** `import "x"` stays one chunk keyed `import:x`; it
  is a complete statement needing no head/tail.

### Failure paths

- **4a. Import-name collision:** an `import "fmt"` on one side and a
  `var fmt = 1` on the other reassemble cleanly but do not compile (`fmt`
  redeclared); the gate now surfaces the collision and the merge is refused
  (resolvable with `merge --ours`/`--theirs`, UC-U32). Duplicate imports and
  same-alias/different-path clashes are refused the same way.

## Sub-variations

- **Alias/blank/dot:** an explicit alias is the bound name; `_` and `.` imports
  bind no checkable name and are skipped by the gate (an alias therefore avoids
  a collision a bare import would cause).

## Known limitations (honest scope)

- **Package-name heuristic:** without an alias the bound name is guessed with
  `path.Base`, which is wrong for paths like `gopkg.in/yaml.v2` (→ `yaml.v2`).
  The `go/types` gate (UC-U40) reads the real package name and supersedes the
  heuristic for the cases it can resolve.
- **Ordering:** unioned imports are not re-sorted (gofmt would); the result is
  valid but may not be in canonical import order.

## Related use cases

- Extends: UC-U37 (Go declaration merge), UC-U38 (order-preserving merge). Uses:
  UC-U32 (`--ours`/`--theirs`) for refused collisions.
