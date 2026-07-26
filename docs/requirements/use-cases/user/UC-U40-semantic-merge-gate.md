# UC-U40: Refuse a merge that does not type-check

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`gotypes.go`: `semanticGateOK`/`typeCheckPackage`; hook in `merge`) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: a default merge must not silently commit Go that no longer type-checks — a symbol duplicated across files, or a reference to a symbol the other branch deleted. |
| Preconditions | The merge touches one or more `.go` files that parse. |
| Trigger | `got merge <branch>` (without `--ours`/`--theirs`). |
| Success postcondition | If the merged package type-checks (within-package), the merge proceeds; otherwise it is refused and the developer resolves it deliberately. |
| Failure postcondition | The merge is aborted before commit; nothing is written. |

## Main success scenario

1. Developer runs `got merge <branch>`. After the merged graph is produced (and
   the per-file structural gates have passed), System groups the merged `.go`
   file vertices by directory and restricts to packages a changed file belongs
   to.
2. For each such package, System parses all its merged files and type-checks
   them together with `go/types` (`typeCheckPackage`).
3. If type-checking reports an in-package redeclaration or an undefined name,
   System refuses the merge (`semanticGateOK` → false) and prints the offending
   message; nothing is committed.
4. Otherwise the merge proceeds to commit.

## Extensions

### Successful variations

- **3a. Override:** `merge --ours`/`--theirs` is the explicit override — it does
  not run the semantic gate, so a developer who has chosen a side is never
  blocked (UC-U32).
- **1a. No Go files:** a merge touching no parseable `.go` package skips the gate
  entirely.

### Failure paths

- **3b. Cross-file redeclaration:** one branch adds a definition of a symbol that
  already exists in another file of the package — refused (`X redeclared`), a
  case the per-file structural gate cannot see and git commits silently.
- **3c. Dangling reference:** one branch removes a symbol while another adds a
  reference to it — refused (`undefined: X`).

## Sub-variations

- **Tolerances (honest scope):** the gate is a *within-package* net, not a
  whole-program build. It tolerates — and never refuses on — parse errors (the
  structural gate owns syntax), imports it cannot resolve (the sandbox lacks the
  module's own `internal/...` and external deps), and unused-import/variable
  warnings. It refuses only on unambiguous merge hazards (redeclaration,
  undefined name). When it cannot make a confident negative judgement it passes.

## Related use cases

- Extends: UC-U37 (Go declaration merge), UC-U39 (import merge — this supersedes
  the `path.Base` import-name heuristic for packages it can resolve). Uses:
  UC-U32 (`--ours`/`--theirs`) as the override.
