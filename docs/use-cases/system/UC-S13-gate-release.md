# UC-S13: Gate a frontier for release

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/governance` |
| Primary actor | `governance.Engine` |
| Stakeholders & interests | Release manager: yes/no eligibility plus outstanding obligations if no. Compliance: a `true` result implies `Check` would have returned `Sat`. |
| Preconditions | A frontier and a policy set are supplied. |
| Trigger | `release.Service.Promote` (UC-U07) calls down. |
| Success postcondition | `(true, nil, nil)` is returned. |
| Failure postcondition | `(false, obligations, nil)` or an error. |

## Main success scenario

1. System invokes UC-S12 to compute the aggregate decision and obligations.
2. If the decision is `Sat` and obligations are empty, system returns `(true, nil, nil)`.

## Extensions

### Successful variations

- **1a. Empty policy set:**
  - 1a1. System returns `(true, nil, nil)`.

### Failure paths

- **2a. Decision is `Unsat`:**
  - 2a1. System returns `(false, obligations, nil)` listing the unmet obligations.
- **2b. Decision is `Unknown`:**
  - 2b1. System returns `(false, obligations, nil)` per the strict policy on `Unknown`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Strictness on `Unknown`:** strict (treat as fail) or lenient (treat as pass) — strict is the default.

## Related use cases

- Includes: UC-S12 (Check policy aggregate).
- Included by: UC-U07 (Promote release).
