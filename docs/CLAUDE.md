# docs/

Documentation layout and writing rules.

## Layout

- `design-rules.md` — canonical API rules. This is the rules-of-record for
  new and refactored code. Cite section numbers when applying a rule.
- `devlog/` — chronological journal, one file per UTC day. Convention lives
  in `devlog/CLAUDE.md`; do not change the format without updating that file.
- `requirements/` — primary requirements. Currently houses the
  Cockburn-style use case catalogue under `requirements/use-cases/`
  (user goals in `user/`, sub-function operations in `system/`,
  convention in `requirements/use-cases/CLAUDE.md`, ledger in
  `requirements/use-cases/ledger.md`, template in
  `requirements/use-cases/template.md`). When adding or changing a
  public method on any internal Engine or Service, update the relevant
  UC in the same change.

## Where things go

| Want to record | Put it in |
|---|---|
| A new design rule or change to an existing one | `design-rules.md` |
| A decision made during a session, with rationale | `devlog/YYYY-MM-DD.md` |
| A user-observable goal the system serves | `requirements/use-cases/user/UC-U<NN>-...md` |
| An internal sub-function operation | `requirements/use-cases/system/UC-S<NN>-...md` |
| User-facing setup or build instructions | `/README.md` |
| Sandbox-only manual tasks Claude can't do | `/CLAUDE.md` |
| Per-package categorical role / allowed imports | the `package X` doc-comment in code |

## Writing style

- No marketing language. State what is, not what is impressive about it.
- Reference files as `path:line`, commits as short SHAs, packages as
  `internal/<pkg>`.
- Devlog entries are append-only. Corrections are new entries that
  reference the original, not edits to it.
- Keep rules in `design-rules.md` short and citable. Long explanations
  belong in the devlog entry that introduced the rule.
