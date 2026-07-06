# UC-U25: Inspect, tag, and revert commits

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`show`, `tag`/`tags`, `revert`) over `history`, `graph.Diff` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: inspect a commit and its change, name commits, and undo a commit safely with a new commit. |
| Preconditions | An initialized repository with at least one commit for `show`/`revert`. |
| Trigger | `got show`, `got tag`/`got tags`, or `got revert`. |
| Success postcondition | `show` prints a commit's metadata and its diff against its parent; `tag` names a commit; `revert` records a new commit that undoes the target and updates the working graph. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer runs `got show [<commit-ish>]` (default HEAD). System resolves the commit-ish — a branch name (its tip), a tag, or a commit-id hex prefix — and prints the commit id, merge parents (if a merge), author, message, and the structural diff (UC-S27) against its first parent.
2. Developer runs `got tag <name> [<commit-ish>]` (default HEAD) to name a commit; `got tags` lists all tags.
3. Developer runs `got revert <commit-ish>`. System computes the reverse of the target commit's change (the delta from the commit back to its parent), applies it to the current working graph, and records a new "Revert: …" commit on the current branch, advancing its tip and working graph.

## Extensions

### Successful variations

- **1a. Commit-ish forms:** a branch name, a tag, or a ≥4-char commit-id prefix all resolve.
- **3a. Revert of an addition:** reverting a commit that added elements removes them; reverting a removal restores them; reverting a change restores the parent's version.

### Failure paths

- **1b. Unknown commit-ish:** `show`/`revert`/`tag` of a ref that resolves to nothing reports "unknown commit-ish".
- **2a. Duplicate tag:** `tag` with an existing name is refused.
- **3b. Endpoint drop:** a reverted edge whose endpoints do not survive is dropped so the result stays well-formed.

## Sub-variations

- **Semantic revert:** the revert is computed as a structural reverse-delta and applied to the graph, not a textual patch — it can incorporate later independent changes on the branch without a textual conflict.
- **Tags are a lightweight file:** tags are stored in `tags.json` (name → commit id), enumerable for `tags`.

## Related use cases

- Includes: UC-S27 (diff for `show`), UC-U22 (commit for `revert`), UC-S26 (history).
