# UC-S06: Issue a certificate for a frontier

| Field | Value |
|---|---|
| Goal level | Sub-function (fish) |
| Scope | `internal/verification` |
| Primary actor | `verification.Engine` |
| Stakeholders & interests | Caller: receive a `Certificate` strong enough for downstream gating. Compliance: certificates are only issued when policies are satisfied. |
| Preconditions | A frontier, a list of `Evaluation` values, and a policy set are supplied. |
| Trigger | A merge, evaluate, or release flow needs evidence that policies are satisfied. |
| Success postcondition | A non-nil `Certificate` is returned whose `Target() == frontier` and `Policies()` covers the supplied policies. |
| Failure postcondition | `verification.ErrCertificationFailed` is returned. |

## Main success scenario

1. System invokes `governance.Engine.Check` for the frontier and policy set (UC-S12).
2. System verifies the aggregate decision is `Sat` and obligations are empty.
3. System constructs the certificate with target, evidence, and policy IDs.
4. System returns the certificate.

## Extensions

### Successful variations

- **1a. Empty policy set:**
  - 1a1. System issues a trivial certificate with `Policies()` empty.
- **3a. Multiple evidence sources reinforce the same claim:**
  - 3a1. System aggregates them into the certificate's evidence list.

### Failure paths

- **2a. Aggregate decision is `Unsat`:**
  - 2a1. System returns `verification.ErrCertificationFailed` listing the unmet policies.
- **2b. Aggregate decision is `Unknown` (insufficient evidence):**
  - 2b1. System returns `verification.ErrCertificationFailed` per the strict policy on `Unknown`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Certificate strength:** the fiber over a frontier may have multiple incomparable certificates; the engine may return the strongest available.

## Related use cases

- Includes: UC-S12 (Check policy aggregate).
- Included by: UC-S03 (Pushout), UC-S04 (Resolve conflicts), UC-S13 (Gate release).
