# Use case catalogue

Stable IDs. Do not renumber. Retired entries keep their ID and gain a
`Retired` status.

## User use cases (sea level)

| ID | Title | Primary actor | Scope |
|---|---|---|---|
| [UC-U01](user/UC-U01-ingest-content.md) | Ingest content into repository | Author / Tool | `repo.Service` |
| [UC-U02](user/UC-U02-revise-graph.md) | Revise the graph via a rewrite rule | Author / Tool | `repo.Service` |
| [UC-U03](user/UC-U03-create-or-update-branch.md) | Create or update a branch | Author | `repo.Service`, `namespace.Store` |
| [UC-U04](user/UC-U04-merge-frontiers.md) | Merge two frontiers | Integrator | `repo.Service` |
| [UC-U05](user/UC-U05-evaluate-frontier.md) | Evaluate a frontier in an environment | Reviewer / CI | `repo.Service` |
| [UC-U06](user/UC-U06-materialize-bundle.md) | Materialize a bundle from a projection | Build system / Consumer | `repo.Service` |
| [UC-U07](user/UC-U07-promote-release.md) | Promote a frontier to a release alias | Release manager | `release.Service` |
| [UC-U08](user/UC-U08-rollback-release.md) | Rollback a release alias | Release manager | `release.Service` |
| [UC-U09](user/UC-U09-resolve-name.md) | Resolve a name to a vertex | Reader | `namespace.Store` |
| [UC-U10](user/UC-U10-query-graph.md) | Query the graph | Reader / Tool | `graph.Graph` |
| [UC-U11](user/UC-U11-trace-provenance.md) | Trace causal provenance | Auditor | `provenance.Engine` |
| [UC-U12](user/UC-U12-trace-authorship.md) | Trace authorship and responsibility | Auditor | `multiagent.Engine` |
| [UC-U13](user/UC-U13-check-freshness.md) | Check temporal freshness of a vertex | Reader / CI | `temporal.Engine` |
| [UC-U14](user/UC-U14-replay-capsule.md) | Replay a change capsule | CI / Auditor | `replay.Engine` |
| [UC-U15](user/UC-U15-prove-claim.md) | Prove a claim with a proof | Verifier | `verification.Engine` |
| [UC-U16](user/UC-U16-detect-emergent-capability.md) | Detect an emergent capability | Capability monitor | `capability.Engine` |
| [UC-U17](user/UC-U17-resolve-merge-conflicts.md) | Resolve merge conflicts | Integrator | `repo.Service`, `composition.Engine` |
| [UC-U18](user/UC-U18-three-way-merge.md) | Three-way merge against a common ancestor | Integrator | `composition.Engine` |
| [UC-U19](user/UC-U19-operate-from-cli.md) | Operate the repository from the command line | Operator | `cmd/got` |

## System use cases (sub-function level)

| ID | Title | Primary engine | Scope |
|---|---|---|---|
| [UC-S01](system/UC-S01-validate-graph.md) | Validate graph well-formedness | `graph.Graph` | `internal/graph` |
| [UC-S02](system/UC-S02-apply-dpo-rewrite.md) | Apply a DPO rewrite | `revision.Engine` | `internal/revision` |
| [UC-S03](system/UC-S03-compute-pushout.md) | Compute the guarded pushout of two frontiers | `composition.Engine` | `internal/composition` |
| [UC-S04](system/UC-S04-resolve-conflicts.md) | Apply conflict resolutions | `composition.Engine` | `internal/composition` |
| [UC-S05](system/UC-S05-evaluate-in-environment.md) | Evaluate a frontier in a given environment | `verification.Engine` | `internal/verification` |
| [UC-S06](system/UC-S06-issue-certificate.md) | Issue a certificate for a frontier | `verification.Engine` | `internal/verification` |
| [UC-S07](system/UC-S07-compute-provenance-closure.md) | Compute the provenance closure of a seed set | `provenance.Engine` | `internal/provenance` |
| [UC-S08](system/UC-S08-compute-provenance-cone.md) | Compute the provenance cone of a vertex | `provenance.Engine` | `internal/provenance` |
| [UC-S09](system/UC-S09-enumerate-causal-traces.md) | Enumerate causal traces between two vertices | `provenance.Engine` | `internal/provenance` |
| [UC-S10](system/UC-S10-select-frontier.md) | Select a frontier from the graph | `projection.Engine` | `internal/projection` |
| [UC-S11](system/UC-S11-apply-projection-spec.md) | Apply a full projection spec | `projection.Engine` | `internal/projection` |
| [UC-S12](system/UC-S12-check-policy-aggregate.md) | Check the aggregate decision over a policy set | `governance.Engine` | `internal/governance` |
| [UC-S13](system/UC-S13-gate-release.md) | Gate a frontier for release | `governance.Engine` | `internal/governance` |
| [UC-S14](system/UC-S14-materialize-for-target.md) | Materialize a view for a specific target | `realization.Engine` | `internal/realization` |
| [UC-S15](system/UC-S15-bind-name.md) | Bind a name to a vertex | `namespace.Store` | `internal/namespace` |
| [UC-S16](system/UC-S16-resolve-binding.md) | Resolve a name binding | `namespace.Store` | `internal/namespace` |
| [UC-S17](system/UC-S17-compute-content-id.md) | Compute a content-addressed identifier | `identity.Factory` | `internal/identity` |
| [UC-S18](system/UC-S18-check-ontology-admissibility.md) | Check whether an edge or hyperedge is admissible | `ontology.Schema` | `internal/ontology` |
| [UC-S19](system/UC-S19-check-replay-feasibility.md) | Check whether a change capsule is replayable | `revision.Engine` | `internal/revision` |
| [UC-S20](system/UC-S20-check-temporal-validity.md) | Check the temporal validity of a vertex | `temporal.Engine` | `internal/temporal` |
| [UC-S21](system/UC-S21-audit-frontier-wellformedness.md) | Audit a frontier for structural and temporal well-formedness | `composition.Engine` | `internal/composition` |
| [UC-S22](system/UC-S22-persist-namespace.md) | Persist namespace bindings to durable storage | `namespace.Store` | `internal/namespace` |

## Coverage

Every public method on every Service or Engine in `internal/` is covered by
at least one user use case (directly or transitively through includes) and
at least one system use case (which exercises the implementing engine).
When a new public method is added, the catalogue must be updated in the
same change.
