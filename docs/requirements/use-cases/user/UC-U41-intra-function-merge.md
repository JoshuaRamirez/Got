# UC-U41: Merge disjoint edits within one function

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`gochunk.go`: statement-level sub-chunking of function bodies) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: two branches editing different statements of the same function should merge — the one case where declaration-level chunking was coarser than git. |
| Preconditions | The contested `.go` file parses and the edits are within a function body. |
| Trigger | `got merge <branch>` where both sides changed the same function differently. |
| Success postcondition | Edits to different statements merge, preserving statement order; edits to the same statement still conflict. |
| Failure postcondition | The file is left to the file-level merge to flag; the go/types gate (UC-U40) backstops any non-type-checking result. |

## Main success scenario

1. Developer runs `got merge <branch>`. The Go chunker decomposes each function
   body into a header chunk, one chunk per top-level statement (keyed
   positionally under the function), and a tail chunk for the closing brace.
2. The graph three-way merge aligns statements by position: a statement edited on
   only one side is a one-sided modification (its key is stable), so two branches
   editing different statements do not conflict.
3. System reassembles the statements in order (order-preserving merge, UC-U38)
   and the merge proceeds.

## Extensions

### Successful variations

- **2a. Disjoint statement edits:** edits to statement *i* on one side and
  statement *j* on the other both apply, in order.

### Failure paths

- **2b. Same statement:** both sides editing the same statement differently
  conflict (resolvable with `merge --ours`/`--theirs`, UC-U32).
- **2c. Statement insert/delete:** adding or removing a statement shifts the
  positional alignment; a concurrent edit near the shift surfaces as a conflict
  rather than a silent scramble. The go/types gate (UC-U40) refuses any merged
  result that would not type-check.

## Sub-variations

- **Positional keys:** statements are keyed by index within the function
  (`<func-key>\x1f<i>`), so an edit is a modification rather than a delete+add.
  This is the deliberate trade: it makes concurrent *edits* merge, and makes
  concurrent structural *changes* (insert/delete) tend toward conflict — not a
  full statement-level diff3 alignment.

## Known limitations (honest scope)

- **Not full alignment:** positional keying merges disjoint edits but does not
  track a statement moved or inserted the way a real statement-level diff3 would.
  Insert/delete combinations conflict rather than merge; they are never silently
  corrupted (the content three-way tends to conflict on the shift, and the
  go/types gate backstops non-compiling results).
- **One level deep:** only top-level statements of a function body are
  sub-chunked; nested blocks are merged as part of their enclosing statement.

## Related use cases

- Extends: UC-U37 (Go declaration merge — this recurses one level deeper). Uses:
  UC-U38 (order-preserving reassembly), UC-U40 (go/types gate backstop), UC-U32
  (`--ours`/`--theirs`).
