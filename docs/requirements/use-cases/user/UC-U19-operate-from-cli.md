# UC-U19: Operate the repository from the command line

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (composes `repo.Service`, `namespace.Store`, `provenance.Engine`) |
| Primary actor | Operator (a human at a shell, or a script) |
| Stakeholders & interests | Operator: drive the library end-to-end without writing Go. Auditor: a persisted, inspectable repository state file. |
| Preconditions | A working directory. For every command except `init`, a repository state file exists (created by `init`). |
| Trigger | Operator runs `got <command> [args]`. |
| Success postcondition | The command performs its operation against the persisted repository, prints a result, persists any state change, and exits 0. |
| Failure postcondition | The command prints a diagnostic to stderr and exits non-zero. The persisted state is unchanged on a rejected mutation. |

## Main success scenario

1. Operator runs `got init`, creating an empty repository state file under the state directory (`$GOT_DIR`, default `.got`).
2. Operator runs `got add-vertex <name> --type <VertexType> [--attr k=v ...]`. The CLI loads the state, ingests the vertex through `repo.Service.Ingest` (UC-U01), persists the new state, and confirms.
3. Operator runs `got add-edge <name> --type <EdgeType> --from <v> --to <v>`. The CLI ingests the edge through `repo.Service.Ingest`; admissibility is enforced by the graph's `Validate` (UC-S01/UC-S18).
4. Operator runs `got bind <ref> <vertex>` to point a branch ref at a vertex through `repo.Service.Branch` (UC-U03), persisting the binding.
5. Operator runs `got resolve <ref>` to print the vertex a ref points to via `namespace.Store.ResolveRef` (UC-U09).
6. Operator runs `got list vertices|edges` to print the graph contents (UC-U10).
7. Operator runs `got trace <from> <to>` to print whether two vertices are causally connected and the simple causal paths between them via `provenance.Engine` (UC-U11), and `got cone <name>` to print a vertex's provenance cone (UC-S08).

## Extensions

### Successful variations

- **1a. `init` over an existing repository:** the CLI reports that a repository already exists and leaves it unchanged (exit 0).
- **7a. `trace` between unconnected vertices:** the CLI reports no causal connection and prints no paths (exit 0).

### Failure paths

- **\*a. Command run before `init`:** the CLI prints "no repository; run 'got init'" and exits non-zero.
- **2a. Unknown vertex type:** `add-vertex` with a type not in the ontology prints a diagnostic and exits non-zero; state unchanged.
- **3a. Inadmissible edge:** `add-edge` whose `(from-type, edge-type, to-type)` triple is not admissible is rejected by `Ingest` (wrapping `graph.ErrNotWellFormed`); the CLI exits non-zero and state is unchanged.
- **3b. Missing endpoint:** `add-edge` referencing an unknown `--from`/`--to` vertex prints a diagnostic and exits non-zero.
- **4a. Bind to unknown vertex:** `bind` to a vertex name not in the graph is rejected (`graph.ErrVertexNotFound`); exit non-zero.
- **5a. Resolve unbound ref:** `resolve` of a ref with no binding prints "unbound" and exits non-zero.
- **\*b. Unknown command / missing arguments:** the CLI prints usage and exits non-zero.

## Sub-variations

- **Identity:** a vertex's content-addressed `VertexID` is `sha256(name)`, the same convention the library's tests use; edges and refs reference vertices by name.
- **Persistence:** repository state is a single JSON file under `$GOT_DIR`. Each mutating command loads, applies through the library, and saves; read commands load and report.

## Related use cases

- Channel for: UC-U01 (Ingest), UC-U03 (Branch), UC-U09 (Resolve name), UC-U10 (Query graph), UC-U11 (Trace provenance), UC-S08 (Provenance cone).
- This UC adds no new engine behavior; it is a new delivery channel (the "input could come from API or CLI" sub-variation) over existing operations.
