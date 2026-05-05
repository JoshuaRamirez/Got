# Internal packages

Working rules for code under `internal/`. The full rationale lives in
`/docs/design-rules.md`; this file is the operational summary.

## API shape

- **`context.Context`**: first parameter on every `Engine`/`Service`
  interface method, plus every `namespace.Store` method (the named
  exception). Do not add `ctx` to pure value-type accessors
  (`Graph`/`Subgraph`/`Frontier`/`View`/...), to `identity.Hasher`/`Factory`,
  to `ontology.Schema`/`Registry`, or to `governance.Policy.Check`.
  Implementations honor `ctx.Err()` at every loop iteration that does real
  work.

- **Errors**: each package declares sentinels at package scope (`var ErrX =
  errors.New("pkg: ...")`) and wraps them via `fmt.Errorf("%w: ...", ErrX,
  detail)`. Callers use `errors.Is`. Do not return bare `errors.New(...)`
  from a public surface unless the failure mode is genuinely caller-opaque.

- **Struct vs interface**: single-getter "data holder" types are structs.
  Interfaces are reserved for multi-method contracts, opaque computation
  results, or types with real implementation alternatives.

- **No `any` in public signatures**: when a parameter is genuinely
  polymorphic, define a typed interface with a string discriminator (see
  `repo.Payload`, `graph.Query`).

## Imports

Each package's `package X` doc-comment declares its allowed imports and the
packages it must not import. Do not violate these — the dependency graph is
deliberately a DAG, with `repo` at the top and `identity`/`ontology` at the
leaves. If you need a new edge, change the doc-comment in the same commit.

## Tests

- Concrete-impl packages (`graph`, `identity`, `namespace`, `ontology`,
  `provenance`) ship behavioral tests covering happy path and error paths
  for every public API.
- Interface-only packages ship minimum tests for exported constants, struct
  round-trips, and sentinel identity. Behavioral tests land with the first
  concrete implementation.
- CI runs `go test -race ./...` plus `gofmt -l .` (must be empty) and
  `go vet ./...`. All three must stay green.

## When you add a new package here

1. Write the `package X` doc-comment with categorical role and import rules.
2. Define interfaces, structs, and sentinel errors per the rules above.
3. Add a `*_test.go` (minimum or behavioral, per the rule).
4. Update `/docs/design-rules.md` if the new package introduces a sentinel
   that should be in the canonical list.
