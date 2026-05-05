---
name: devlog
description: Manage the chronological developer log under docs/devlog/. One file per UTC day, append-only entries. Subcommands: today, show, append, list, latest.
---

# /devlog — manage the developer log

The developer log under `docs/devlog/` is the project's chronological
journal. Convention lives in `docs/devlog/CLAUDE.md`. One file per UTC
day named `YYYY-MM-DD.md`; entries are append-only with `## HH:MM UTC —
Topic` headings.

Parse the user's input as `<subcommand> [args...]`. Default: `today`.

## Project paths

- Convention: `docs/devlog/CLAUDE.md`
- Daily files: `docs/devlog/YYYY-MM-DD.md`

## Subcommands

### `today` (default)

Read today's UTC date with `date '+%Y-%m-%d'` and display
`docs/devlog/<today>.md`. If the file does not exist yet, report that no
entries have been logged today and offer `append` as the next step.

### `show <YYYY-MM-DD>`

Display the log file for the given date.

### `append [<topic>]`

Append a new entry to today's UTC file:

1. Compute today's date (`%Y-%m-%d`) and current UTC time (`%H:%M`) via
   `date`.
2. If `docs/devlog/<today>.md` does not exist, create it with a top-level
   `# YYYY-MM-DD` heading.
3. Append a new entry: `## HH:MM UTC — <topic>` followed by the body.
4. If `<topic>` was not supplied, ask the user what to log; they can
   provide multi-line content.
5. Show the diff for confirmation. Do not commit unless the user asks.

Past entries are append-only — never edit them. Corrections are new
entries that reference the original.

### `list [<N>]`

List the most recent N daily files (default 7). Show each filename and
the count of entries it contains.

### `latest`

Show the most recent entry across all files (the last `## HH:MM UTC —`
heading plus its body in the most recent file).

## Working rules

- UTC for all timestamps. Use `date -u '+%Y-%m-%d %H:%M'` if the host's
  default zone differs.
- Touch only `docs/devlog/`. Do not commit unless the user asks.
- Keep entries tight. Reference commits, files, and PRs by stable
  identifier rather than pasting long output.
