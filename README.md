# Got

An immutable directed-hypergraph engine with mutable namespace control, graph rewriting via decorated DPO, governance, verification, and deterministic replay.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.24 or later

### Clone

```bash
git clone https://github.com/JoshuaRamirez/Got.git
cd Got
```

### Build

```bash
go build ./...
```

### Test

```bash
go test -v -race ./...
```

### Command-line shell

`cmd/got` is a thin CLI over the library. It persists a single JSON state
file under `$GOT_DIR` (default `.got`) and drives the library engines for
each subcommand (see `docs/requirements/use-cases/user/UC-U19-operate-from-cli.md`).

```bash
go build -o got ./cmd/got

./got init
./got add-vertex exec --type Execution
./got add-vertex art  --type Artifact
./got add-edge   e1   --type materializes --from exec --to art
./got list vertices
./got bind main art
./got resolve main
./got branch release-2 --from main --desc "the 2.x line"
./got branch hotfix    --from release-2
./got branches            # each branch is a real object with metadata + parent
./got branch-log hotfix   # fork ancestry: hotfix <- release-2 <- main
./got trace exec art      # causal paths via the provenance engine
./got cone exec           # provenance cone
```

Unlike a git branch — a bare mutable pointer with no identity or history — a
branch here is a first-class `BranchSelector` vertex: it carries metadata, records
its fork parent, persists in the graph, and has traceable ancestry (`branch-log`).

Inadmissible edges are rejected by the graph's well-formedness check, so the
CLI surfaces the same ontology guarantees the library enforces.

State is a repository directory (`repo.SaveState` / `repo.LoadState`): `graph.json`
(the graph snapshot) plus `namespace.json` (the durable namespace). Human names
are carried in a reserved `got.name` attribute so they survive the graph codec.

## Project Structure

```
.
├── internal/
│   ├── identity/        # Content-addressable hashing and typed IDs
│   ├── ontology/        # Type system and admissibility rules
│   ├── graph/           # Immutable typed attributed hypergraph (append-only)
│   ├── namespace/       # Mutable ref/alias/projection-handle bindings
│   ├── provenance/      # Causal cone and trace computation
│   ├── temporal/        # Time-interval queries and freshness checks
│   ├── multiagent/      # Authorship and responsibility tracing
│   ├── revision/        # DPO graph rewriting and change capsules
│   ├── replay/          # Deterministic capsule re-execution
│   ├── projection/      # Frontier selection and subgraph views
│   ├── governance/      # Policy evaluation and release gating
│   ├── verification/    # Evaluation, claim proving, certification
│   ├── capability/      # Emergent capability detection
│   ├── composition/     # Guarded-pushout merge with conflict monad
│   ├── realization/     # Materialization of views into target bundles
│   ├── release/         # Named release promotion and rollback
│   └── repo/            # Top-level facade composing all modules
├── cmd/
│   └── got/             # Command-line shell over the library (UC-U19)
├── .github/
│   └── workflows/
│       └── ci.yml       # GitHub Actions CI pipeline
├── go.mod
├── .gitignore
├── LICENSE
└── README.md
```

## License

Licensed under the [Apache License 2.0](LICENSE).