package ontology

// edgeTriple is the lookup key for the admissibility relation.
type edgeTriple struct {
	Src VertexType
	ET  EdgeType
	Dst VertexType
}

// knownVertexTypes enumerates every vertex type in the ontology.
var knownVertexTypes = map[VertexType]bool{
	Artifact: true, Revision: true, Agent: true, Human: true,
	Prompt: true, Model: true, Tool: true, Execution: true,
	Observation: true, Evaluation: true, Policy: true, Dataset: true,
	Environment: true, Intent: true, Claim: true, Proof: true,
	Task: true, BranchSelector: true, MergeWitness: true, Release: true,
	Capability: true, Certificate: true, Conflict: true, ProjectionSpec: true,
}

// knownEdgeTypes enumerates every edge type in the ontology.
var knownEdgeTypes = map[EdgeType]bool{
	DependsOn: true, DerivedFrom: true, Executes: true, AuthoredBy: true,
	ApprovedBy: true, EvaluatedBy: true, Constrains: true, Supersedes: true,
	Observes: true, Proves: true, Refutes: true, Calls: true,
	Materializes: true, Targets: true, BelongsTo: true, ForksFrom: true,
	MergesInto: true, Selects: true, Binds: true, TracesTo: true,
	Justifies: true, Violates: true, Satisfies: true, Certifies: true,
}

// admissibleEdges is the minimal conservative admissibility table.
// The system is closed under extension: entries can be added but not safely removed.
var admissibleEdges = map[edgeTriple]bool{
	// Artifact lineage
	{Artifact, DerivedFrom, Artifact}: true,
	{Revision, DerivedFrom, Artifact}: true,
	{Artifact, DerivedFrom, Revision}: true,

	// Execution
	{Execution, Executes, Model}:       true,
	{Execution, Executes, Tool}:        true,
	{Execution, DerivedFrom, Prompt}:   true,
	{Execution, Materializes, Artifact}: true,

	// Evaluation
	{Evaluation, EvaluatedBy, Execution}: true,
	{Evaluation, Targets, Artifact}:      true,
	{Evaluation, Targets, Revision}:      true,

	// Governance
	{Certificate, Certifies, Artifact}: true,
	{Certificate, Certifies, Revision}: true,
	{Policy, Constrains, Artifact}:     true,
	{Policy, Constrains, Revision}:     true,

	// Provenance links
	{Claim, Proves, Artifact}: true,
	{Proof, Proves, Claim}:    true,

	// Agents (non-causal metadata)
	{Agent, AuthoredBy, Artifact}: true,
	{Human, ApprovedBy, Artifact}: true,
}

// DefaultSchema implements Schema with the minimal conservative admissibility table.
type DefaultSchema struct{}

// NewDefaultSchema returns a schema encoding the minimal admissibility relation.
func NewDefaultSchema() *DefaultSchema {
	return &DefaultSchema{}
}

func (s *DefaultSchema) KnownVertexType(vt VertexType) bool {
	return knownVertexTypes[vt]
}

func (s *DefaultSchema) KnownEdgeType(et EdgeType) bool {
	return knownEdgeTypes[et]
}

func (s *DefaultSchema) EdgeAllowed(src VertexType, et EdgeType, dst VertexType) bool {
	return admissibleEdges[edgeTriple{src, et, dst}]
}

// HyperedgeAllowed checks the canonical executes hyperedge signature.
//
// The one canonical hyperedge:
//
//	Type: executes
//	Inputs:  Prompt, Model, context (Artifact | Revision | Dataset), Policy
//	Outputs: Revision, Observation
func (s *DefaultSchema) HyperedgeAllowed(inputs []VertexType, et EdgeType, outputs []VertexType) bool {
	if et == Executes {
		return checkExecutesHyperedge(inputs, outputs)
	}
	return false
}

func checkExecutesHyperedge(inputs, outputs []VertexType) bool {
	inCount := make(map[VertexType]int)
	for _, vt := range inputs {
		inCount[vt]++
	}

	// Required inputs: at least one Prompt, Model, and Policy.
	if inCount[Prompt] < 1 || inCount[Model] < 1 || inCount[Policy] < 1 {
		return false
	}

	// At least one context input from {Artifact, Revision, Dataset}.
	if inCount[Artifact]+inCount[Revision]+inCount[Dataset] < 1 {
		return false
	}

	// All input types must be from the allowed set.
	allowedIn := map[VertexType]bool{
		Prompt: true, Model: true, Policy: true,
		Artifact: true, Revision: true, Dataset: true,
	}
	for _, vt := range inputs {
		if !allowedIn[vt] {
			return false
		}
	}

	// Required outputs: at least one Revision and one Observation.
	outCount := make(map[VertexType]int)
	for _, vt := range outputs {
		outCount[vt]++
	}
	if outCount[Revision] < 1 || outCount[Observation] < 1 {
		return false
	}

	// All output types must be from the allowed set.
	allowedOut := map[VertexType]bool{
		Revision: true, Observation: true,
	}
	for _, vt := range outputs {
		if !allowedOut[vt] {
			return false
		}
	}

	return true
}

// DefaultRegistry provides access to the default schema.
type DefaultRegistry struct{}

// NewDefaultRegistry creates a registry that returns the default schema.
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{}
}

func (r *DefaultRegistry) DefaultSchema() Schema {
	return NewDefaultSchema()
}
