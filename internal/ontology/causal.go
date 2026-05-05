package ontology

// CausalEdges defines the subset of edge types that constitute causal
// relationships for provenance computation.
//
// Included (causal):   derived_from, executes, calls, materializes, traces_to, proves
// Excluded (metadata): authored_by, approved_by, belongs_to
// Deferred (may become causal in future extensions): depends_on, evaluated_by
var CausalEdges = map[EdgeType]bool{
	DerivedFrom:  true,
	Executes:     true,
	Calls:        true,
	Materializes: true,
	TracesTo:     true,
	Proves:       true,
}

// IsCausal returns true if the given edge type is in the causal set.
func IsCausal(et EdgeType) bool {
	return CausalEdges[et]
}
