# Use cases

Cockburn-style use cases for the system. **These are the primary
requirements.** Every public method on every internal `Engine` and
`Service` is reachable from at least one UC here. New features begin as a
new UC in this catalogue, before any code is written.

Two layers:

- `user/` — user goals (sea level): a stakeholder wants the system to
  accomplish something observable.
- `system/` — sub-function use cases (fish level): an internal operation
  that supports one or more user use cases.

Two siblings track plan state:

- `ledger.md` — what is done and where each UC stands.
- `roadmap.md` — what to do next and why that order.

Together they encode the full plan at any moment. The `/use-case` slash
command (see `.claude/skills/use-case/SKILL.md`) automates the common
operations on this catalogue, including `/use-case roadmap` and
`/use-case next`.

Every use case has a stable ID:

- `UC-U<NN>` for user (sea) use cases.
- `UC-S<NN>` for system (sub-function) use cases.

Numbering is allocated in the catalogue (`index.md`) and never reused. If a
use case is retired, mark it `Retired` in the index — do not renumber.

## Template

Use `template.md` verbatim when adding a new use case. Required sections:

1. **Header table** — Goal level, Scope, Primary actor, Stakeholders &
   Interests, Preconditions, Trigger, Postconditions (success and failure).
2. **Main success scenario** — numbered steps, primary path, written from
   the actor-system interaction perspective.
3. **Extensions** — branches off main steps. Sub-divide into:
   - **Successful variations** — alternate paths that still reach a success
     postcondition.
   - **Failure paths** — paths that end in a failure postcondition.
4. **Sub-variations** — orthogonal variations applicable to multiple steps
   (e.g. "any input could come from API or CLI").
5. **Related use cases** — IDs of UCs that this one includes, extends, or
   that include this one.

## Writing rules

- Step text describes interaction at the goal level, not implementation.
  System steps name the package or interface that fulfills them in
  parentheses, e.g. "(governance.Engine.Check)".
- Extensions are numbered against the step they branch from: `3a`, `3b`,
  etc. Sub-steps under an extension use `3a1`, `3a2`.
- Failure paths end with the failure postcondition explicitly, citing the
  sentinel error if one exists (e.g. `governance.ErrPolicyViolation`).
- Keep each use case self-contained — a reader should be able to follow it
  without jumping to other files. Refer to other UCs by ID, not by
  paraphrasing them.
- One file per use case. Filename: `UC-U<NN>-<slug>.md` or
  `UC-S<NN>-<slug>.md`.

## Maintenance

When adding or changing an interface in `internal/`, update any UC whose
Main Success Scenario or Extensions cite that interface. The use case
catalogue is part of the design surface; out-of-date entries are bugs.

## Ledger

`ledger.md` records the implementation/verification status of every UC.
Update it in the same commit that changes implementation or test coverage:

1. Move the UC's row to the new status (`Specified` → `Partial` →
   `Implemented` → `Verified`, or `Retired`).
2. Bump `Last reviewed` to the commit's UTC date.
3. Cite the implementing package or file in `Implementation`.
4. Cite the test file(s) in `Tests`.

Status definitions live in `ledger.md`. The only status that satisfies the
test-gating rule in `/docs/design-rules.md` is `Verified`.

## Roadmap

`roadmap.md` records the optimal dependency-ordered implementation chain
for moving every UC from `Specified` to `Verified`. Update it when:

1. A package's UCs all move to `Verified` in the ledger — refresh
   "Current focus" to point at the next active phase.
2. The dependency graph changes (new package added, deps revised) —
   re-stratify the phases and update the diagram.
3. A phase completes — move that block above "Active phase".

Do not re-order phases without a corresponding architecture change. The
order is determined by package imports, not preference.
