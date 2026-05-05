---
name: requirements
description: Top-level navigator for project requirements under docs/requirements/. Today the only requirement format is the use case catalogue; delegates to /use-case for that work.
---

# /requirements — navigate project requirements

The `docs/requirements/` directory holds the project's primary
requirements. Today the only format in active use is the Cockburn-style
use case catalogue under `docs/requirements/use-cases/`. This skill is
the front door — it tells the user what formats exist and which slash
command manages each.

Parse the user's input as `<subcommand> [args...]`. Default: `list`.

## Project paths

- Convention: `docs/requirements/CLAUDE.md`
- Use case catalogue: `docs/requirements/use-cases/`
- Use case ledger: `docs/requirements/use-cases/ledger.md`

## Subcommands

### `list` (default)

Show the available requirement formats and the slash command that
manages each:

| Format | Path | Slash command |
|---|---|---|
| Use cases (Cockburn) | `docs/requirements/use-cases/` | `/use-case` |

Then briefly summarize the working rule from `docs/requirements/CLAUDE.md`:
"A new feature begins as a new requirement here, before any code is
written."

### `status`

Delegate to `/use-case status` since use cases are currently the only
requirement format with a tracked ledger. If new formats are added that
also have ledgers, aggregate their summaries here.

### `new <format>`

Route to the appropriate creator:

- `new use-case` → instruct the user to run `/use-case new <user|system> <slug>`.
- Other formats: report that no other requirement format is registered
  yet; suggest editing this skill to add one.

### `audit`

Delegate to `/use-case audit`. Aggregate findings from any future
requirement-format audits.

## Working rules

- Do not duplicate logic that lives in a more specific skill (`/use-case`).
  Always delegate to the most specific skill.
- When a new requirement format is introduced, add it to the table in
  `list` and to the routing in `new` and `audit`.
- Touch only `docs/requirements/`.
