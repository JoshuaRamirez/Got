# UC-U08: Rollback a release alias

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `release.Service` |
| Primary actor | Release manager |
| Stakeholders & interests | Release manager: revert a release alias to a previously named state. Compliance: rollback target was itself once promoted (or recorded). Consumers: alias resolves to the previous state after rollback. |
| Preconditions | The alias has at least one historical binding identified by the `version` string. |
| Trigger | Release manager initiates rollback. |
| Success postcondition | `namespace.Store.ResolveAlias(alias)` returns the historical vertex. |
| Failure postcondition | The namespace is unchanged. An error is reported. |

## Main success scenario

1. Actor invokes `release.Service.Rollback(ctx, alias, version)`.
2. System looks up the historical binding for `(alias, version)` in the release ledger.
3. System invokes `namespace.Store.BindAlias` to overwrite the current binding with the historical vertex (UC-S15).
4. System returns nil.

## Extensions

### Successful variations

- **2a. Version refers to the current binding:**
  - 2a1. System completes as a no-op success.

### Failure paths

- **2b. No record of `(alias, version)` in the ledger:**
  - 2b1. System returns `release.ErrUnknownVersion`.
- **3a. Bind fails at the underlying store:**
  - 3a1. System returns the store's error unchanged.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Source of historical bindings:** in-graph release vertices, namespace-store history, external ledger.

## Related use cases

- Includes: UC-S15 (Bind name).
- Related: UC-U07 (Promote a frontier to a release alias).
