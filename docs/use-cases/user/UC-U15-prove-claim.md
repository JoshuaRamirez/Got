# UC-U15: Prove a claim with a proof

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `verification.Engine` |
| Primary actor | Verifier |
| Stakeholders & interests | Verifier: get a yes/no answer about whether a proof validates a claim. Auditor: every (claim, proof) interaction is reproducible. |
| Preconditions | The `Claim` and `Proof` reference vertices that exist in the graph. |
| Trigger | Verifier asks whether a proof discharges a claim. |
| Success postcondition | A boolean is returned. The graph is not mutated by this call. |
| Failure postcondition | An error is returned. |

## Main success scenario

1. Actor calls `Prove(ctx, g, claim, proof)`.
2. System retrieves the claim and proof vertices and any linked evidence.
3. System runs the verification check appropriate to the claim's kind.
4. System returns the boolean.

## Extensions

### Successful variations

- **3a. Proof refutes the claim:**
  - 3a1. System returns `(false, nil)` — the absence of validation is not an error.

### Failure paths

- **2a. Claim or proof vertex missing:**
  - 2a1. System returns `graph.ErrVertexNotFound`.
- **3b. Verification logic itself fails (e.g., proof corrupt):**
  - 3b1. System returns the error wrapped with its origin.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Claim kinds:** safety, liveness, schema-conformance, evaluation-threshold, etc. — each with its own proof shape.

## Related use cases

- Includes: none directly.
- Related: UC-U05 (Evaluate frontier), UC-S06 (Issue certificate).
