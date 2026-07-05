# UC-U20: Persist and reload a repository

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `repo` (via `repo.SaveState` / `repo.LoadState`) |
| Primary actor | Repository host (a server, daemon, or CLI) |
| Stakeholders & interests | Host: keep a repository across process restarts without replaying the ingest history. Operator: an inspectable on-disk repository directory. |
| Preconditions | A directory path and an `ontology.Schema`. |
| Trigger | The host loads a repository at startup and saves it after mutating operations. |
| Success postcondition | After `SaveState`, the on-disk directory reflects the in-memory `State`. A later `LoadState` on the same directory returns a `State` whose graph and namespace bindings match what was saved. |
| Failure postcondition | An error is returned; a corrupt or ill-formed on-disk graph is rejected at load rather than surfacing later. |

## Main success scenario

1. Host calls `repo.LoadState(dir, schema)`. The graph is read from `dir/graph.json` (an absent file yields an empty graph, validated on decode — UC-S23); the namespace is backed by a durable `namespace.FileStore` over `dir/namespace.json` (UC-S22).
2. Host drives operations on the returned `State` through `repo.Service` (Ingest, Revise, Branch, ...). Each graph-mutating operation returns a new `State` carrying a new immutable graph value; each namespace bind is flushed to disk immediately by the FileStore.
3. Host calls `repo.SaveState(dir, state)` to persist the current graph value with an atomic write-then-rename.
4. On a later run, `repo.LoadState(dir, schema)` reconstructs a `State` whose graph and namespace match the saved repository.

## Extensions

### Successful variations

- **1a. Empty directory:** `LoadState` on a directory with no `graph.json` returns an empty graph and an empty (but durable) namespace.
- **2a. Namespace durability without SaveState:** because the namespace is a `FileStore`, binds persist the moment they happen; a crash between binds and the next `SaveState` loses no namespace state (only unsaved graph mutations are lost).
- **3a. Repeated saves:** a later `SaveState` overwrites `graph.json`, so `LoadState` always sees the newest graph value.

### Failure paths

- **1b. Corrupt or ill-formed graph file:** `LoadState` returns an error wrapping `graph.ErrNotWellFormed` (or a JSON error), because the snapshot codec validates on decode (UC-S23).
- **3b. Disk write fails:** `SaveState` returns the underlying I/O error; the previous `graph.json` is left intact (the write goes to a temp file that is only renamed on success).
- **\*. Non-`FileStore` namespace:** if a caller constructs a `State` with a namespace other than a `FileStore`, `SaveState` still persists the graph, but namespace durability is the caller's responsibility. (A `State` from `LoadState` always uses a `FileStore`.)

## Sub-variations

- **Two persistence rhythms:** the immutable graph is written explicitly by `SaveState`; the mutable namespace persists continuously via its `FileStore`. This mirrors the architecture's split between the append-only graph and the single mutable namespace shell.

## Related use cases

- Composes UC-S23 (graph snapshot codec) and UC-S22 (durable namespace `FileStore`).
- Serves any operation performed through `repo.Service` (UC-U01–UC-U06) by making its result durable.
