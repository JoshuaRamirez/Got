# UC-S22: Persist namespace bindings to durable storage

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `namespace.Store` (via `namespace.FileStore`) |
| Primary actor | `namespace.Store` |
| Stakeholders & interests | Any long-lived repository host (server, daemon): keep ref/alias/projection bindings across process restarts. Concurrent clients: bind and resolve safely from multiple goroutines. |
| Preconditions | A filesystem path for the store file. |
| Trigger | A host opens a durable namespace via `namespace.NewFileStore(path)` and issues binds/resolves. |
| Success postcondition | Bindings written via `Bind*` are flushed to disk and are visible to any later `NewFileStore` opened on the same path. Resolution returns the durable value. |
| Failure postcondition | An I/O or corruption error is returned; the in-memory state is not left partially applied for the failed operation. |

## Main success scenario

1. Host calls `namespace.NewFileStore(path)`. If the file exists it is loaded; otherwise the store starts empty.
2. Host calls `BindRef` / `BindAlias` / `BindProjection`. System updates the in-memory map under a mutex and flushes the whole state to disk with an atomic write-then-rename.
3. Host calls `ResolveRef` / `ResolveAlias` / `ResolveProjection`. System returns the bound vertex ID (or `ok == false` if unbound).
4. On a later run, `NewFileStore` on the same path reloads all bindings, so resolution returns the previously-persisted values.

## Extensions

### Successful variations

- **2a. Rebind:** binding an existing name overwrites the previous target and persists the new one.
- **3a. Concurrent access:** every method holds a mutex, so concurrent binds and resolves are safe (unlike the in-memory `memStore`, which is documented as unsafe for concurrent writers). Verified under `go test -race`.

### Failure paths

- **1a. Corrupt store file:** `NewFileStore` returns an error rather than silently starting empty, so a damaged file is surfaced.
- **2b. Disk write fails:** `Bind*` returns the underlying I/O error.

## Sub-variations

- **Durability boundary:** only the namespace is persisted. The graph is content-addressed and reconstructable, so the mutable namespace is the meaningful state to make durable; a host reconstructs or separately snapshots the graph.
- **Encoding:** the store file is JSON mapping each name to a hex-encoded vertex ID.

## Related use cases

- Alternative implementation of the same interface exercised by UC-S15 (Bind a name) and UC-S16 (Resolve a binding); `FileStore` adds durability and concurrency-safety to those operations.
- Enables durable variants of UC-U03 (Branch), UC-U07/UC-U08 (Promote/Rollback release aliases), and UC-U09 (Resolve a name).
