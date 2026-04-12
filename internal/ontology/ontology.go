// Package ontology implements the OntologyKernel specification.
//
// It defines the type system for vertices and edges, and the admissibility
// rules that govern which edge and hyperedge signatures are well-formed.
//
// Imports: none (identity only if IDs are embedded in schema metadata).
// Must not import: graph or any upper layer.
package ontology

// VertexType classifies a vertex within the ontology.
//
// Known vertex types from the specification:
//
//	Artifact, Revision, Agent, Human, Prompt, Model, Tool, Execution,
//	Observation, Evaluation, Policy, Dataset, Environment, Intent,
//	Claim, Proof, Task, BranchSelector, MergeWitness, Release,
//	Capability, Certificate, Conflict, ProjectionSpec
type VertexType string

// EdgeType classifies an edge or hyperedge within the ontology.
//
// Known edge types from the specification:
//
//	depends_on, derived_from, executes, authored_by, approved_by,
//	evaluated_by, constrains, supersedes, observes, proves, refutes,
//	calls, materializes, targets, belongs_to, forks_from, merges_into,
//	selects, binds, traces_to, justifies, violates, satisfies
type EdgeType string

// RoleType classifies the trust or authority role of an agent.
type RoleType string

// Schema encodes the admissibility rules of the ontology. Every edge or
// hyperedge in any valid graph must satisfy one of these relations.
type Schema interface {
	// KnownVertexType returns true if the given type is in the ontology.
	KnownVertexType(VertexType) bool

	// KnownEdgeType returns true if the given type is in the ontology.
	KnownEdgeType(EdgeType) bool

	// EdgeAllowed returns true if an edge of the given type may connect
	// a source vertex of type src to a destination vertex of type dst.
	EdgeAllowed(src VertexType, et EdgeType, dst VertexType) bool

	// HyperedgeAllowed returns true if a hyperedge of the given type may
	// connect the given input vertex types to the given output vertex types.
	HyperedgeAllowed(inputs []VertexType, et EdgeType, outputs []VertexType) bool
}

// Registry provides access to the ontology schema in effect.
type Registry interface {
	DefaultSchema() Schema
}
