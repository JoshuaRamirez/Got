# UC-U21: Manage first-class branches

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo` (via `CreateBranch`, `Branches`, `BranchLineage`) |
| Primary actor | Integrator / repository host |
| Stakeholders & interests | Integrator: create branches that carry identity, metadata, and fork lineage, and reason about where a branch came from. Auditor: branches persist in history rather than vanishing when a pointer is deleted. |
| Preconditions | A repository state. A named parent branch, if given, already exists. |
| Trigger | An actor creates a branch, lists branches, or asks for a branch's fork ancestry. |
| Success postcondition | The branch exists as a `BranchSelector` vertex in the append-only graph, carrying its name, optional parent (as a `forks_from` edge), and metadata; its tip, when supplied, is bound in the namespace. |
| Failure postcondition | An error is returned and the graph is unchanged. |

## Main success scenario

1. Actor calls `CreateBranch(state, name, parent, tip, meta)`.
2. System records the branch as a `BranchSelector` vertex whose content-addressed id is derived from the name, carrying `branch.name`, an optional `branch.parent`, and any metadata attributes (`repo.Service.Ingest`, UC-U01).
3. When a parent is given, System links the branch to its parent with a `forks_from` edge (admissible: `{BranchSelector, ForksFrom, BranchSelector}`).
4. When a tip is given, System binds the namespace ref `name -> tip` (UC-U03) — the one mutable, advancing part of the branch.
5. System returns the new state and the `Branch`.
6. Actor calls `Branches(state)` to list every branch (a graph query over `BranchSelector` vertices, UC-S24-style), or `BranchLineage(state, name)` to walk the `forks_from` chain from a branch up to its root.

## Extensions

### Successful variations

- **1a. Root branch:** a branch created with no parent has no `forks_from` edge; its lineage is just itself.
- **4a. No tip:** a branch may exist before it points anywhere; the namespace has no binding for it until a tip is set (via this call or `repo.Service.Branch`).

### Failure paths

- **1b. Branch already exists:** System returns `repo.ErrBranchExists` and does not modify the graph.
- **2a. Unknown parent:** a named parent branch not present in the graph yields `repo.ErrUnknownBranch`.
- **4b. Unknown tip:** a tip vertex not in the graph yields `graph.ErrVertexNotFound`.
- **\*. `ctx` cancelled:** System returns `ctx.Err()`.

## Sub-variations

- **First-class vs. pointer:** unlike a git branch (a bare mutable ref with no identity, metadata, or history), a branch here is a real graph object — content-addressed, attributed, permanent, queryable, with traceable ancestry. The mutable tip is deliberately the *only* pointer-like part.
- **Channel:** reachable from the `cmd/got` CLI (`branch`, `branches`, `branch-log`) or the library.

## Related use cases

- Includes: UC-U01 (Ingest), UC-U03 (Branch/bind the tip), UC-S18 (admissibility of the `forks_from` edge).
- Complements: UC-U04/UC-U18 (merge), where branch tips are the frontiers being merged.
