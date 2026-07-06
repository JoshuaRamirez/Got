# UC-U27: Delete and rename branches

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`branch -d`, `branch -m`) over `namespace.Store.DeleteRef` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: remove branches that are no longer needed and rename branches, without leaving stale pointers. |
| Preconditions | An initialized repository. |
| Trigger | `got branch -d <name>` or `got branch -m <old> <new>`. |
| Success postcondition | Delete removes the branch's commit pointer (and its first-class vertex, if any). Rename moves the commit pointer and, if it is the current branch, HEAD, to the new name. |
| Failure postcondition | An error is returned; branches are unchanged. |

## Main success scenario

1. Developer runs `got branch -d <name>`: System removes the branch's commit pointer (`namespace.Store.DeleteRef` on `commit:<name>`) and, if a first-class `BranchSelector` vertex exists, removes it from the graph.
2. Developer runs `got branch -m <old> <new>`: System rebinds the commit pointer from `commit:<old>` to `commit:<new>`, deletes the old pointer, updates HEAD if the current branch was renamed, and drops the old first-class vertex if present.

## Extensions

### Successful variations

- **1a. Pointer-only branch:** deleting a branch that has no first-class vertex removes just its commit pointer.

### Failure paths

- **1b. Delete current branch:** deleting the current (HEAD) branch is refused.
- **1c. Unknown branch:** deleting/renaming a branch that does not exist reports "no such branch".
- **2a. Target exists:** renaming to an existing branch name is refused.

## Sub-variations

- **Two branch notions:** a branch's VCS identity is its `commit:<name>` pointer; a first-class `BranchSelector` vertex (UC-U21) is optional richer metadata. Delete removes both; rename moves the pointer and drops the old vertex (re-create with `got branch <new>` to restore metadata/lineage under the new name). Branch vertices are excluded from content diff/status, so removing one does not register as an uncommitted change.

## Related use cases

- Uses: `namespace.Store.DeleteRef` (added for this UC; also usable to delete any ref/tag).
- Complements: UC-U21 (first-class branches), UC-U23 (current branch / HEAD).
