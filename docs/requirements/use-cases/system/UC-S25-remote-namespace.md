# UC-S25: Bind and resolve names over a network

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `namespace.Store` (via `namespace.HTTPStore` client and `namespace.NewHTTPHandler` server) |
| Primary actor | `namespace.Store` client |
| Stakeholders & interests | A distributed repository host: run the mutable namespace as a shared service that many clients bind/resolve against. |
| Preconditions | A running HTTP server (`NewHTTPHandler` wrapping any `Store`) reachable at a base URL. |
| Trigger | A client calls a `Store` method on an `HTTPStore`. |
| Success postcondition | The bind/resolve is performed against the server's backing store; the result matches what a local `Store` would return. The caller's `context.Context` is attached to the HTTP request. |
| Failure postcondition | A bind returns a transport/HTTP error. A resolve, which has no error return, surfaces a transport failure as `ok == false`. |

## Main success scenario

1. A host wraps any `Store` (e.g. `FileStore` or `memStore`) with `namespace.NewHTTPHandler(store)` and serves it over HTTP.
2. A client constructs `namespace.NewHTTPStore(baseURL, httpClient)` — itself a `Store`.
3. The client calls `BindRef` / `BindAlias` / `BindProjection`; the client `POST`s `/bind` with `{kind, name, hex-id}` and the server delegates to the wrapped store, returning `204`.
4. The client calls `ResolveRef` / `ResolveAlias` / `ResolveProjection`; the client `GET`s `/resolve?kind=..&name=..` and the server returns `{found, id}`; the client returns `(id, found)`.

## Extensions

### Successful variations

- **3a. Names with reserved characters:** names are URL-escaped, so `feature/x y&z` binds and resolves correctly.
- **4a. Unbound name:** the server returns `{found:false}` and the client returns `ok == false`.

### Failure paths

- **\*. `ctx` cancelled or transport error on bind:** `Bind*` returns the underlying error (the ctx is threaded onto the request via `http.NewRequestWithContext`).
- **\*. `ctx` cancelled or transport error on resolve:** because the `Store` resolve methods have no error return, the failure surfaces as `ok == false`.
- **3b. Malformed id / unknown kind:** the server responds `400`; the client's bind returns an error.

## Sub-variations

- **Backing store:** the handler wraps any `Store`, so the remote namespace can itself be durable (`FileStore`) or in-memory (`memStore`).
- **Wire format:** JSON over HTTP; vertex IDs are hex-encoded (the same encoding `FileStore` uses on disk).

## Related use cases

- Same operations as UC-S15 (Bind a name) and UC-S16 (Resolve a binding), exercised across a network boundary; realizes the remote/persistent backing the `Store` interface's `context.Context` parameter anticipates.
- The wrapped store may be UC-S22's `FileStore`, giving a durable remote namespace.
