# UC-U12: Trace authorship and responsibility

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `multiagent.Engine` |
| Primary actor | Auditor |
| Stakeholders & interests | Auditor: identify which agents authored a vertex and the full responsibility chain. Compliance: every artifact has at least one accountable agent (when authorship edges are required by policy). |
| Preconditions | The target vertex exists in the graph. |
| Trigger | Auditor asks who authored or is responsible for a vertex. |
| Success postcondition | A list of agent vertex IDs (`Authorship`) or a `Responsibility` value with a `Path` (`ResponsibilityTrace`) is returned. |
| Failure postcondition | `multiagent.ErrNoAuthorship` or another error is returned. |

## Main success scenario

1. Actor calls `Authorship(ctx, g, target)` or `ResponsibilityTrace(ctx, g, target)`.
2. System traverses `authored_by` edges (and any other configured authorship edges) from the target.
3. System returns the agent IDs or the full responsibility path.

## Extensions

### Successful variations

- **2a. Multiple authors:**
  - 2a1. System returns all agent IDs in deterministic order.
- **2b. Responsibility path passes through delegation edges:**
  - 2b1. System follows the configured delegation edge types and returns the full path.

### Failure paths

- **1a. Target vertex not in graph:**
  - 1a1. System returns `graph.ErrVertexNotFound`.
- **2c. No authorship edges from the target:**
  - 2c1. `Authorship` returns the empty slice (no error). `ResponsibilityTrace` returns `multiagent.ErrNoAuthorship`.
- **\*. `ctx` cancelled:**
  - System returns `ctx.Err()`.

## Sub-variations

- **Authorship edge set:** typically `authored_by`; configurable per engine.
- **Path depth:** unbounded by default; callers may filter post-hoc.

## Related use cases

- Includes: none directly.
- Related: UC-U11 (Trace causal provenance).
