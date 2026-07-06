# UC-U29: Cherry-pick and amend commits

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`cherry-pick`, `amend`) over `history`, `graph.Diff` |
| Primary actor | Developer |
| Stakeholders & interests | Developer: apply a single commit's change onto the current branch, and fix up the most recent commit. |
| Preconditions | An initialized repository with commits. |
| Trigger | `got cherry-pick <commit-ish>` or `got amend [-m <message>]`. |
| Success postcondition | `cherry-pick` records a new commit on the current branch containing the target commit's change. `amend` replaces the current branch tip with a new commit whose snapshot is the working state (keeping the original parents), with an optional new message. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer runs `got cherry-pick <commit-ish>`. System computes the target commit's forward change (the delta from its parent to itself), applies it to the current working graph, and records a new "cherry-pick …" commit on the current branch, advancing the tip and working graph.
2. Developer runs `got amend [-m <message>]`. System records a new commit that keeps the current tip's parents but uses the current working graph as its snapshot (folding in uncommitted changes) and an optional new message, then moves the branch tip to it. The previous commit becomes unreferenced.

## Extensions

### Successful variations

- **2a. Message only:** `amend` with no working changes and `-m` just rewrites the message.

### Failure paths

- **1a. Unknown commit-ish:** `cherry-pick` of a ref resolving to nothing reports "unknown commit-ish".
- **2b. Nothing to amend:** `amend` on a branch with no commit reports an error.

## Sub-variations

- **Semantic apply:** cherry-pick applies the target's *structural* forward delta (added/removed/changed nodes and edges) to the current graph — last-write-wins on overlap, rather than a textual patch. (It does not yet surface conflicts the way `merge` does; that is a possible enhancement.)
- **Amend is a rewrite:** like git, amend produces a new commit id and leaves the old commit unreferenced in the log.

## Related use cases

- Uses: UC-S27 (structural diff / apply), UC-U22 (commit), UC-U25 (commit-ish resolution).
