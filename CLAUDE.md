# Claude Code Notes

## Outstanding Manual Tasks

### CI: Add gofmt format check (requires `workflow` OAuth scope)

The following step needs to be added to `.github/workflows/ci.yml` after the
"Set up Go" step and before the "Build" step. It could not be pushed
automatically because the OAuth token lacks `workflow` scope.

```yaml
      - name: Format check
        run: test -z "$(gofmt -l .)"
```

### P2: Open Design Questions (Go Quality)

These inconsistencies were identified during the quality review but require
design decisions before fixing:

1. **No `context.Context` on any interface method** — Every Engine/Service
   method that may involve I/O or cancellation should accept `context.Context`
   as its first parameter per Go convention. This is a pervasive API change
   across all 17 packages.

2. **No custom error types** — The codebase has zero typed errors or sentinel
   errors. Domain errors like `ErrVertexNotFound`, `ErrPolicyViolation`,
   `ErrConflictUnresolvable` would let callers programmatically distinguish
   error conditions.

3. **Struct vs interface inconsistency for data types** — Simple data holders
   are sometimes structs (`Interval`, `Obligation`, `ChangeCapsule`) and
   sometimes interfaces with a single getter (`Trace`, `Outcome`,
   `FidelityContract`, `Witness`). A documented rationale for when to use each
   approach would improve consistency.

4. **`Ingest(State, any)` loose typing** — The `any` parameter in
   `repo.Service.Ingest` provides no type safety. Every other Service method
   uses typed parameters.

5. **Zero test files** — CI runs `go test -v -race` but there are no
   `*_test.go` files. Tests require implementations to exist first.
