---
name: librarian
description: Navigate all project documentation. Indexes every docs/ folder and the slash command that manages it. Use when you need to find where to read or write a particular kind of doc.
---

# /librarian — navigate project documentation

The librarian is the front door for everything under `docs/`. It does not
edit content directly — it routes you to the right doc folder and the
slash command that manages it.

Parse the user's input as `<subcommand> [args...]`. Default: `list`.

## Doc-folder skill map

| Folder | Purpose | Slash command | Convention file |
|---|---|---|---|
| `docs/requirements/` | Primary requirements (currently use cases) | `/requirements` | `docs/requirements/CLAUDE.md` |
| `docs/requirements/use-cases/` | Cockburn use case catalogue | `/use-case` | `docs/requirements/use-cases/CLAUDE.md` |
| `docs/devlog/` | Chronological session journal | `/devlog` | `docs/devlog/CLAUDE.md` |
| `docs/` (root file) | API design rules | — (read `docs/design-rules.md`) | `docs/CLAUDE.md` |

## Subcommands

### `list` (default)

Show the doc-folder skill map above. Then list any top-level docs files
that are not folder-managed (`docs/design-rules.md`, etc.) and the
canonical-references list from the root `CLAUDE.md`.

### `where <topic>`

Given a topic, recommend where to read or write. Match the topic against
these heuristics:

- "requirement", "use case", "UC", "user goal", "system function" →
  `/use-case` for the catalogue, `/requirements` for the broader format
  index.
- "design rule", "API rule", "ctx", "errors", "struct vs interface",
  "Payload" → `docs/design-rules.md`.
- "decision", "session log", "what changed today" → `/devlog`.
- "package conventions", "imports", "internal" → `internal/CLAUDE.md`.
- "CI", "workflow", "gofmt" → `.github/CLAUDE.md`.

Cite the most specific destination first; mention the convention file
for that folder.

### `audit`

Run the audits of every doc-folder skill that supports it and summarize:

1. `/use-case audit` — orphan files, missing files, stale `Verified`,
   coverage gaps, totals drift.
2. (Future) `/devlog audit` — gaps in daily coverage, malformed entries.
3. (Future) `/requirements audit` — cross-format consistency.

For each audited skill, show its findings under a heading. Report
nothing for an audited skill if it has no findings.

### `tree`

Render a compact tree view of `docs/` showing folders, their convention
files, and the managing slash command. Skip individual content files.

## Working rules

- The librarian never edits content. For any write operation, route to
  the specific skill responsible for that folder.
- When a new top-level docs folder is added, update the table in this
  file and add a new skill under `.claude/skills/<name>/SKILL.md`. The
  root `CLAUDE.md` folder map should also be updated in the same change.
- Stay short. The librarian's value is fast routing, not long-form
  documentation.
