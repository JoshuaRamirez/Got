# UC-U11: Trace causal provenance

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `provenance.Engine` |
| Primary actor | Auditor |
| Stakeholders & interests | Auditor: identify the causal cone of a vertex or determine whether one vertex causes another. Reviewer: trust the closure operator's algebraic guarantees. |
| Preconditions | The vertices of interest exist in the graph. The engine is configured with a causal-edges set (typically `ontology.CausalEdges`). |
| Trigger | Auditor wants to know what caused or what was caused by a vertex. |
| Success postcondition | One of: a boolean (`Causes`), a vertex set (`Cone`, `Close`), or a list of `Trace` values (`TraceSet`). |
| Failure postcondition | An error is returned. |

## Main success scenario

1. Actor calls one of:
   - `Causes(ctx, g, from, to)` — boolean reachability over causal edges.
   - `Cone(ctx, g, seed)` — full causal cone of a single seed (UC-S08).
   - `Close(ctx, g, seed)` — closure of a seed set (UC-S07).
   - `TraceSet(ctx, g, from, to)` — enumerate causal traces (UC-S09).
2. System computes the result via undirected BFS or DFS over the causal-edge subgraph.
3. System returns the result.

## Extensions

### Successful variations

- **1a. Reflexive query (`from == to` on `Causes`):**
  - 1a1. System returns `(true, nil)` immediately per the reflexivity axiom.
- **1b. Empty seed set on `Close`:**
  - 1b1. System returns an empty slice and nil error per the extensiveness axiom (`Close({}) = {}`).
- **2a. Path to target found early in `Causes`:**
  - 2a1. System short-circuits the BFS and returns true.

### Failure paths

- **1c. Seed or endpoint vertex not in graph:**
  - 1c1. System returns `provenance.ErrUnknownVertex` wrapping the offending ID.
- **\*. `ctx` cancelled mid-traversal:**
  - System returns `ctx.Err()` from the loop iteration that observed the cancellation. No partial result is returned.

## Sub-variations

- **Engine configuration:** which `ontology.EdgeType` values count as causal — typically `ontology.CausalEdges`, but configurable.
- **Path semantics:** `TraceSet` returns simple paths only (no cycles).

## Related use cases

- Includes: UC-S07 (Compute provenance closure), UC-S08 (Compute provenance cone), UC-S09 (Enumerate causal traces).
- Related: UC-U12 (Trace authorship and responsibility).
