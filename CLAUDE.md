# Claude Code Notes

This file is the root index. Scoped rules live in per-folder `CLAUDE.md`
files; consult the one closest to the code you are touching.

## Folder map

- `/internal/CLAUDE.md` — API rules for internal packages (context, errors,
  struct-vs-interface, imports, tests).
- `/docs/CLAUDE.md` — documentation layout and writing rules.
- `/docs/devlog/CLAUDE.md` — devlog convention (one file per UTC day).
- `/.github/CLAUDE.md` — CI pipeline and workflow rules.

## Canonical references

- `/docs/design-rules.md` — full API design rules with rationale.
- `/docs/devlog/YYYY-MM-DD.md` — chronological journal.
- `/README.md` — user-facing setup / build / test.

## Outstanding manual tasks

None at the moment. CI's `Format check` (`gofmt -l .`) is live in
`.github/workflows/ci.yml`; the prior "needs workflow OAuth scope" warning
no longer applies.
