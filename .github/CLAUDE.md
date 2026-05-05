# .github/

CI and repository automation. The only file here is `workflows/ci.yml`.

## CI pipeline (`workflows/ci.yml`)

Runs on push and PR against `main`. Steps, in order:

1. `actions/checkout@v4`
2. `actions/setup-go@v5` (Go version pinned via `go.mod`)
3. `gofmt -l .` — must produce empty output
4. `go build ./...`
5. `go test -v -race -coverprofile=coverage.out ./...`
6. `go vet ./...`

All four checks must pass. Permissions are scoped to `contents: read`.

## Working rules

- Keep CI fast. Do not add steps that don't enforce a rule.
- If a check is added here, document the corresponding rule in
  `/docs/design-rules.md` (or in `/internal/CLAUDE.md` if package-scoped).
- Do not skip the format check. If `gofmt` complains, run `gofmt -w` and
  commit; do not add `// nolint`-style suppressions.

## Token scope caveat (historical)

The root `CLAUDE.md` previously warned that workflow edits required a
`workflow` OAuth scope the sandbox token lacked. As of `91f6072`
(2026-05-05) that warning is removed — the sandbox now permits workflow
pushes. If a future session sees a 403 on workflow files specifically,
note it in the devlog and fall back to applying the change manually.
