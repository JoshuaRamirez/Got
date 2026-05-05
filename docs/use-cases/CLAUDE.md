# Use cases

Cockburn-style use cases for the system. Two layers:

- `user/` — user goals (sea level): a stakeholder wants the system to
  accomplish something observable.
- `system/` — sub-function use cases (fish level): an internal operation
  that supports one or more user use cases.

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
