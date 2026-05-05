# Developer log

A chronological record of work done on this repo, one file per day.

## Layout

- One file per UTC day: `YYYY-MM-DD.md` (e.g. `2026-05-05.md`).
- The date-prefixed filename sorts lexicographically, so `ls docs/devlog` is also a chronological listing.
- This `CLAUDE.md` is the convention file; do not log entries here.

## Entry format

Inside each day file, append entries in chronological order. Each entry starts with an `##` heading containing a 24-hour UTC timestamp and a short topic, followed by the body:

```
## HH:MM UTC — Topic

Body. Plain prose, lists, or fenced code as needed.
Reference files as `path:line` and commits as short SHAs.
```

Rules:
- Use UTC for timestamps so entries from any environment sort correctly.
- Always append; never rewrite past entries. If a prior entry was wrong, add a new one that corrects it and reference the original.
- One entry per logical chunk of work (a merge, an investigation, a decision), not per command.
- Keep bodies tight. Link to commits, PRs, and files instead of pasting long output.

## When to write

Write an entry when any of these happen:
- A branch is merged, created, or deleted.
- A non-trivial decision is made (architecture, dependency, process).
- An investigation produces a finding worth remembering (root cause, dead end, gotcha).
- A failed attempt is abandoned — record why so it isn't retried.

Routine commits do not need entries; the git log already covers them.

## Starting a new day

If today's file does not exist yet, create it with a top-level `# YYYY-MM-DD` heading and then append the first entry. No other boilerplate.
