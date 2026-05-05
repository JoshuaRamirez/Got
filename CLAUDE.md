# Claude Code Notes

This file is the root index. Scoped rules live in per-folder `CLAUDE.md`
files; consult the one closest to the code you are touching. For doc
navigation, run `/librarian`.

## Use cases are the primary requirements

The Cockburn-style use case catalogue under `docs/requirements/use-cases/` is the
canonical statement of what this system does. Every public method on every
internal `Engine` and `Service` is reachable from at least one user use
case (sea level) and exercised by at least one system use case (fish
level).

**Working rules**:

- A new feature begins as a new use case in the catalogue, before any code
  is written. Use the `/use-case` skill or copy `docs/requirements/use-cases/template.md`.
- A change to existing behavior begins as an edit to the relevant UC.
- A UC is not "done" until its row in `docs/requirements/use-cases/ledger.md` reads
  `Verified` — implementation plus behavioral tests covering the main
  success path and at least one failure path per extension group.
- The ledger is updated in the same commit that changes implementation or
  test coverage. Out-of-date ledger rows are bugs.

## Folder map

- `/internal/CLAUDE.md` — API rules for internal packages (context, errors,
  struct-vs-interface, imports, tests).
- `/docs/CLAUDE.md` — documentation layout and writing rules.
- `/docs/requirements/CLAUDE.md` — requirements layout (currently use cases).
- `/docs/requirements/use-cases/CLAUDE.md` — use case catalogue convention and ledger
  protocol.
- `/docs/devlog/CLAUDE.md` — devlog convention (one file per UTC day).
- `/.github/CLAUDE.md` — CI pipeline and workflow rules.

## Slash commands

- `/librarian` — navigate all project documentation.
- `/requirements` — top-level requirements navigator.
- `/use-case` — manage the use case catalogue.
- `/devlog` — manage the chronological session journal.

## Canonical references

- `/docs/requirements/use-cases/index.md` — full UC catalogue index.
- `/docs/requirements/use-cases/ledger.md` — UC implementation/verification status.
- `/docs/design-rules.md` — full API design rules with rationale.
- `/docs/devlog/YYYY-MM-DD.md` — chronological journal.
- `/README.md` — user-facing setup / build / test.

## Outstanding manual tasks

None at the moment. CI's `Format check` (`gofmt -l .`) is live in
`.github/workflows/ci.yml`; the prior "needs workflow OAuth scope" warning
no longer applies.
