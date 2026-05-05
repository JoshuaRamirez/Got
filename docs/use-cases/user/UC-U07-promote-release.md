# UC-U07: Promote a frontier to a release alias

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `release.Service` (composes `governance`, `verification`, `namespace`, `projection`) |
| Primary actor | Release manager |
| Stakeholders & interests | Release manager: a release alias points at a certified frontier. Compliance: only frontiers passing all policies can be promoted. Consumers: the alias resolves deterministically once promoted. |
| Preconditions | A certified `Frontier` and `Certificate` are available. A non-empty policy set `[]governance.Policy` is supplied. |
| Trigger | Release manager triggers a promotion. |
| Success postcondition | The `namespace.Store` binds the alias to the frontier's vertex. The promotion is auditable via the underlying graph. |
| Failure postcondition | The namespace is unchanged. An error is reported. |

## Main success scenario

1. Actor invokes `release.Service.Promote(ctx, alias, frontier, certificate, policies)`.
2. System checks that `certificate.Target()` matches the supplied `frontier`.
3. System invokes `governance.Engine.GateRelease` to confirm the frontier is admissible (UC-S13).
4. System invokes `namespace.Store.BindAlias` (UC-S15) to bind the alias to the frontier's witness vertex.
5. System returns nil.

## Extensions

### Successful variations

- **4a. Alias already bound (re-promotion):**
  - 4a1. System overwrites the existing binding. The prior alias→vertex mapping is not retained except via the devlog or audit trail.
- **3a. Policy set empty:**
  - 3a1. System treats the gate as trivially `Sat` and proceeds.

### Failure paths

- **2a. Certificate target does not match frontier:**
  - 2a1. System returns `verification.ErrEnvironmentMismatch` (or a release-scoped error).
- **3b. Policy gate returns false (`Unsat` or unmet `Obligation`s):**
  - 3b1. System returns `release.ErrPolicyGate` carrying the unmet obligations.
- **4a. Bind fails at the underlying store:**
  - 4a1. System returns the store's error unchanged.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Alias lifecycle:** first-time promotion, version bump, hotfix re-promotion.
- **Certificate strength:** any object in the certificate fiber — caller may pass the strongest available.

## Related use cases

- Includes: UC-S13 (Gate release), UC-S15 (Bind name).
- Related: UC-U08 (Rollback release).
