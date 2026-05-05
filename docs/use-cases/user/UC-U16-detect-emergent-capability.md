# UC-U16: Detect an emergent capability

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `capability.Engine` |
| Primary actor | Capability monitor |
| Stakeholders & interests | Monitor: identify capabilities that arise from the joint action of governance, verification, and projection. Auditor: emergence is named and witnessed. |
| Preconditions | A `Frontier`, a policy set, and a list of certificates are supplied. |
| Trigger | Monitor evaluates the current configuration for capability emergence. |
| Success postcondition | `(true, Witness{Name: ...}, nil)` is returned. |
| Failure postcondition | `(false, Witness{}, nil)` (no emergence — not an error) or an error. |

## Main success scenario

1. Actor calls `Emerges(ctx, g, frontier, policies, certificates)`.
2. System checks the frontier against the policy set (UC-S12).
3. System inspects each certificate's targets and policies.
4. System evaluates the configured emergence predicates for the (frontier, policies, certificates) triple.
5. If any predicate fires, system returns `(true, witness, nil)` with the named capability.

## Extensions

### Successful variations

- **5a. Multiple predicates fire:**
  - 5a1. System returns the highest-precedence witness per the engine's configuration; remaining witnesses are reported via the engine's audit channel.

### Failure paths

- **4a. No predicate fires:**
  - 4a1. System returns `(false, Witness{}, nil)`. The non-emergent path is not an error.
- **2a. Policy aggregate fails (`Unsat`):**
  - 2a1. System short-circuits and returns `(false, Witness{}, capability.ErrNoEmergence)` indicating emergence is unreachable.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Predicate set:** configurable per engine instance; may evolve as new emergent capabilities are recognized.

## Related use cases

- Includes: UC-S12 (Check policy aggregate).
- Related: UC-S06 (Issue certificate).
