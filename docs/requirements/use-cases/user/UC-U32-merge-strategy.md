# UC-U32: Resolve a merge with a strategy

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo` (`MergeStatesStrategy`), `cmd/got` (`merge --ours`/`--theirs`) |
| Primary actor | Developer / Integrator |
| Stakeholders & interests | Developer: complete a conflicting merge by choosing a side per conflict, rather than aborting. |
| Preconditions | Both branches have commits and a semantic three-way merge would conflict. |
| Trigger | `got merge <branch> --ours` or `got merge <branch> --theirs`. |
| Success postcondition | A merge commit is recorded whose conflicting nodes/edges take the current branch's version (`--ours`) or the merged branch's version (`--theirs`); non-conflicting changes from both sides still merge. |
| Failure postcondition | An error is returned; nothing changes. |

## Main success scenario

1. Developer runs `got merge <branch>` and hits conflicts; the plain merge aborts and lists the typed conflicts with a hint.
2. Developer re-runs with `--ours` (keep the current branch's side on conflict) or `--theirs` (take the other branch's side).
3. System performs the three-way merge with a tiebreaker: agreed changes are taken; one-sided changes are taken; on a genuine same-target conflict it picks the chosen side (`repo.MergeStatesStrategy`).
4. System records a two-parent merge commit and advances the branch and working graph.

## Extensions

### Successful variations

- **3a. Non-conflicting still merges:** independent changes on both sides are merged regardless of strategy; the strategy only decides genuine conflicts.
- **3b. Modify/delete:** `--ours` keeps our action (our modification, or our deletion), `--theirs` keeps theirs.

### Failure paths

- **2a. Both flags:** `--ours` and `--theirs` together are rejected.

## Sub-variations

- **Per-node/edge resolution:** the strategy applies per conflicting element (vertex or edge), not to the whole tree — so a single `--ours` merge still incorporates the other branch's independent work.

## Related use cases

- Extends: UC-U24 (semantic merge). Uses: UC-U18 (three-way reconciliation semantics).
