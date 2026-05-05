# Claude Code Notes

## Outstanding Manual Tasks

### CI: gofmt format check

`.github/workflows/ci.yml` includes a `Format check` step that runs
`test -z "$(gofmt -l .)"`. If a sandboxed agent without `workflow` OAuth
scope edits the workflow file, it will need to be applied manually.

## Design rules

API shape (context, errors, struct-vs-interface, `Ingest` typing) and the
test-gating policy are documented in `docs/design-rules.md`. Apply those
rules to all new and refactored code.
