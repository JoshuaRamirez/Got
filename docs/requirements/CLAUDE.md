# Requirements

This directory holds the project's primary requirements. Today the only
requirement format is the Cockburn-style use case catalogue under
`use-cases/`. Other formats (PRDs, ADRs, threat models) may live alongside
in their own subdirectories if they appear later.

## Layout

- `use-cases/` — Cockburn use case catalogue. The full convention and
  ledger live in `use-cases/CLAUDE.md`. Every public method on every
  internal `Engine` and `Service` is covered by at least one UC; the
  ledger at `use-cases/ledger.md` records implementation/verification
  status of each.

## Slash commands

- `/requirements` — top-level navigation across requirement formats.
- `/use-case` — manage the use case catalogue specifically.

## Working rule

A new feature begins as a new requirement here, before any code is
written. The catalogue is the source of truth for what the system does.
Out-of-date entries are bugs.
