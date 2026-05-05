---
name: use-case
description: Manage the Cockburn-style use case catalogue under docs/requirements/use-cases/. Use cases are this project's primary requirements. Subcommands: list, show, new, status, set-status, roadmap, next, audit.
---

# /use-case — manage use case catalogue

The use case catalogue under `docs/requirements/use-cases/` is this project's primary
requirements document. Every public method on every internal `Engine` and
`Service` is reachable from at least one UC. Two siblings track plan
state:

- `ledger.md` — what is done and where each UC stands.
- `roadmap.md` — what to do next and why that order.

This skill performs the common operations on the catalogue. Parse the
user's input as `<subcommand> [args...]`. If no subcommand is supplied,
default to `list`.

## Project paths

- Catalogue index: `docs/requirements/use-cases/index.md`
- Ledger: `docs/requirements/use-cases/ledger.md`
- Roadmap: `docs/requirements/use-cases/roadmap.md`
- Convention: `docs/requirements/use-cases/CLAUDE.md`
- Template: `docs/requirements/use-cases/template.md`
- User UCs: `docs/requirements/use-cases/user/UC-U<NN>-<slug>.md`
- System UCs: `docs/requirements/use-cases/system/UC-S<NN>-<slug>.md`

## Subcommands

### `list` (default)

Show the ledger's two summary tables (User and System), then the totals
table. Read `docs/requirements/use-cases/ledger.md` and render the relevant sections.
Do not re-read every UC file; the ledger is the source of truth for
status.

### `show <UC-ID>`

Open `docs/requirements/use-cases/{user|system}/<UC-ID>-*.md` and display it. Resolve
the file by globbing the ID prefix; the suffix slug is part of the
filename but not the ID.

### `new <user|system> <slug>`

Create a new UC:

1. Find the next free numeric ID for the requested layer by reading the
   matching tables in `docs/requirements/use-cases/index.md` and `docs/requirements/use-cases/ledger.md`.
2. Copy `docs/requirements/use-cases/template.md` to the new file path
   (`docs/requirements/use-cases/<layer>/UC-<U|S><NN>-<slug>.md`).
3. Replace `UC-X<NN>: <Title>` with the actual ID and a placeholder title
   the user can refine.
4. Append a row to the relevant section of `docs/requirements/use-cases/ledger.md`
   with status `Specified`, today's UTC date, and `—` placeholders for
   Implementation/Tests/Notes.
5. Append a row to the relevant section of `docs/requirements/use-cases/index.md`.
6. Tell the user the new file path so they can fill in details.

Do not commit unless the user asks. Do not bump the totals table; the
list output should compute totals on the fly when the user later runs
`/use-case status`.

### `status [<UC-ID>]`

Without an ID: same as `list` plus the `Summary` and `Next-bite candidates`
sections from the ledger.

With an ID: read the ledger row for that UC and report Status,
Implementation, Tests, Last reviewed, Notes.

### `set-status <UC-ID> <new-status> [<notes>]`

Update the ledger row for one UC. Valid statuses: `Specified`, `Partial`,
`Implemented`, `Verified`, `Retired` (definitions in `ledger.md`).

1. Edit the matching row in `docs/requirements/use-cases/ledger.md`.
2. Bump `Last reviewed` to today's UTC date.
3. If notes were supplied, replace the `Notes` cell.
4. If the new status is `Verified`, also confirm with the user that the
   UC has tests covering the main success path and at least one failure
   path per extension group, per the design-rules test-gating rule.
5. Recompute the totals table at the bottom of the ledger.
6. If this status change completes a phase per `roadmap.md`, remind the
   user to update the roadmap's "Current focus" section.
7. Show the diff for confirmation. Do not commit unless the user asks.

### `roadmap`

Render `docs/requirements/use-cases/roadmap.md`. The roadmap shows the
optimal dependency-ordered implementation chain and which packages
unlock which UCs. Use this when planning what to work on next.

### `next`

Recommend the single next-best package to implement, by cross-referencing
the ledger and the roadmap:

1. Read the ledger to find which packages are already `Verified`.
2. Read the roadmap to find the active phase.
3. Within the active phase, prefer:
   a. Packages on the critical path (`projection`, `governance`,
      `verification`, `composition`, `repo`).
   b. Packages that unlock the most UCs.
   c. Packages with no remaining-phase dependencies.
4. Report the chosen package, the UCs it will unlock when verified, and
   any prerequisites already met.

If no work remains in the active phase, advance to the next phase and
repeat.

### `audit`

Verify catalogue consistency. Report — do not modify — the following:

1. **Orphan files**: any `docs/requirements/use-cases/{user,system}/UC-*.md` that has
   no row in `ledger.md` or `index.md`.
2. **Missing files**: any ledger row whose UC file does not exist on disk.
3. **Stale `Verified`**: any UC marked `Verified` whose cited test file
   does not exist or no longer references the implementing package.
4. **Coverage gaps**: any public method on an `Engine` or `Service` in
   `internal/` that is not cited by any UC's Main Success Scenario or
   Extensions. Use `grep -rn "func (.*) [A-Z]" internal/` to enumerate
   public methods, and grep the UC files for citations like
   `repo.Service.Foo` or `(governance.Engine.Bar)`.
5. **Totals drift**: recompute the totals table from the rows and report
   any mismatch with the file's recorded totals.

Output is a short bulleted report under each heading; nothing if a
heading has no findings.

## Working rules

- Touch only `docs/requirements/use-cases/` and the ledger unless the subcommand
  explicitly requires editing other files.
- Do not commit anything from this skill. The user runs `git commit`.
- When in doubt about which UC ID applies to a code change, prefer the
  most specific (lowest-level) UC. The user-level UC will then transitively
  include it through the `Related use cases` section.
- IDs are stable. Never renumber.
