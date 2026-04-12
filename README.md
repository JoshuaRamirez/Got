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