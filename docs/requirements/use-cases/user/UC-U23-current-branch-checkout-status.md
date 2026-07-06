# UC-U23: Switch branches and see working status

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`checkout`/`switch`, `status`; HEAD) over `repo`, `history`, `graph.Diff` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: a current branch (HEAD) so everyday commands need no `--branch`; switch branches with the working graph following along; see what is uncommitted. |
| Preconditions | An initialized repository (`init` sets HEAD to `main`). |
| Trigger | Developer runs `status`, `checkout`/`switch`, or a command that defaults to the current branch. |
| Success postcondition | `status` reports the current branch and uncommitted content changes; `checkout` moves HEAD and updates the working graph to the target branch's committed state; `commit`/`log`/`diff` default to HEAD. |
| Failure postcondition | An error is returned; HEAD and the working graph are unchanged. |

## Main success scenario

1. `init` records HEAD = `main` (a `HEAD` file naming the current branch).
2. Developer runs `status`: System prints the current branch and, comparing the working graph against the branch's head commit snapshot (content only — first-class branch vertices are excluded), either "nothing to commit, working graph clean" or the list of uncommitted changes.
3. Developer runs `checkout <branch>` (alias `switch`): System refuses if there are uncommitted content changes (unless `--force`), then updates the working graph (`graph.json`) to the target branch's committed snapshot and points HEAD at it.
4. Developer runs `commit`, `log`, or `diff` with no branch argument: System uses HEAD.

## Extensions

### Successful variations

- **3a. Create a branch:** `checkout -b <name>` creates a new branch at the current branch's commit (the working graph is kept) and switches to it.
- **3b. Empty target:** switching to a branch with no commits yields an empty working graph.
- **2a. First-class branch metadata excluded:** creating a first-class branch (UC-U21) adds a `BranchSelector` vertex to the graph, but `status`/`diff` treat branch vertices as metadata, not content, so they do not register as uncommitted changes.

### Failure paths

- **3c. Unknown branch:** `checkout` of a branch that has no commit pointer and no first-class vertex (and is not the current HEAD) reports "no such branch (use -b to create)".
- **3d. Dirty working graph:** `checkout` with uncommitted content changes is refused unless `--force`.
- **3e. Existing branch with -b:** `checkout -b` on an existing branch is refused.
- **\*a. Before init:** any command before `got init` reports "run 'got init'".

## Sub-variations

- **HEAD is a file:** the current branch is stored in a `HEAD` file in the repository directory (git-style), naming the branch. It is the one process-independent notion of "where I am".
- **Working graph follows HEAD:** `graph.json` is the working tree; `checkout` rewrites it to the target branch's committed state.

## Related use cases

- Builds on UC-U22 (commit history) and UC-U21 (first-class branches); consumes UC-S27 (structural diff) for `status`.
