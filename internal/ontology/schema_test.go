package ontology_test

import (
	"testing"

	"github.com/joshuaramirez/got/internal/ontology"
)

func schema() ontology.Schema {
	return ontology.NewDefaultSchema()
}

func TestKnownVertexTypes(t *testing.T) {
	s := schema()
	known := []ontology.VertexType{
		ontology.Artifact, ontology.Revision, ontology.Agent, ontology.Human,
		ontology.Prompt, ontology.Model, ontology.Tool, ontology.Execution,
		ontology.Observation, ontology.Evaluation, ontology.Policy, ontology.Dataset,
		ontology.Certificate, ontology.Claim, ontology.Proof,
	}
	for _, vt := range known {
		if !s.KnownVertexType(vt) {
			t.Errorf("expected %s to be a known vertex type", vt)
		}
	}
	if s.KnownVertexType("Nonexistent") {
		t.Error("Nonexistent should not be a known vertex type")
	}
}

func TestAdmissibleEdges(t *testing.T) {
	s := schema()
	cases := []struct {
		src ontology.VertexType
		et  ontology.EdgeType
		dst ontology.VertexType
		ok  bool
	}{
		// Artifact lineage
		{ontology.Artifact, ontology.DerivedFrom, ontology.Artifact, true},
		{ontology.Revision, ontology.DerivedFrom, ontology.Artifact, true},
		{ontology.Artifact, ontology.DerivedFrom, ontology.Revision, true},
		// Execution
		{ontology.Execution, ontology.Executes, ontology.Model, true},
		{ontology.Execution, ontology.Executes, ontology.Tool, true},
		{ontology.Execution, ontology.DerivedFrom, ontology.Prompt, true},
		{ontology.Execution, ontology.Materializes, ontology.Artifact, true},
		// Evaluation
		{ontology.Evaluation, ontology.EvaluatedBy, ontology.Execution, true},
		{ontology.Evaluation, ontology.Targets, ontology.Artifact, true},
		// Governance
		{ontology.Certificate, ontology.Certifies, ontology.Artifact, true},
		{ontology.Policy, ontology.Constrains, ontology.Revision, true},
		// Provenance
		{ontology.Claim, ontology.Proves, ontology.Artifact, true},
		{ontology.Proof, ontology.Proves, ontology.Claim, true},
		// Agents (non-causal)
		{ontology.Agent, ontology.AuthoredBy, ontology.Artifact, true},
		{ontology.Human, ontology.ApprovedBy, ontology.Artifact, true},
		// Inadmissible
		{ontology.Artifact, ontology.Executes, ontology.Model, false},
		{ontology.Human, ontology.Materializes, ontology.Artifact, false},
		{ontology.Agent, ontology.Certifies, ontology.Revision, false},
	}
	for _, tc := range cases {
		got := s.EdgeAllowed(tc.src, tc.et, tc.dst)
		if got != tc.ok {
			t.Errorf("EdgeAllowed(%s, %s, %s) = %v, want %v",
				tc.src, tc.et, tc.dst, got, tc.ok)
		}
	}
}

func TestCanonicalHyperedge(t *testing.T) {
	s := schema()

	// Valid: Prompt + Model + Artifact (context) + Policy -> Revision + Observation
	inputs := []ontology.VertexType{ontology.Prompt, ontology.Model, ontology.Artifact, ontology.Policy}
	outputs := []ontology.VertexType{ontology.Revision, ontology.Observation}
	if !s.HyperedgeAllowed(inputs, ontology.Executes, outputs) {
		t.Error("canonical executes hyperedge should be allowed")
	}

	// Valid with Dataset context
	inputs2 := []ontology.VertexType{ontology.Prompt, ontology.Model, ontology.Dataset, ontology.Policy}
	if !s.HyperedgeAllowed(inputs2, ontology.Executes, outputs) {
		t.Error("executes hyperedge with Dataset context should be allowed")
	}

	// Invalid: missing Policy
	bad := []ontology.VertexType{ontology.Prompt, ontology.Model, ontology.Artifact}
	if s.HyperedgeAllowed(bad, ontology.Executes, outputs) {
		t.Error("executes hyperedge without Policy should not be allowed")
	}

	// Invalid: wrong edge type
	if s.HyperedgeAllowed(inputs, ontology.DerivedFrom, outputs) {
		t.Error("non-executes hyperedge should not be allowed")
	}
}

func TestCausalEdgeSet(t *testing.T) {
	causal := []ontology.EdgeType{
		ontology.DerivedFrom, ontology.Executes, ontology.Calls,
		ontology.Materializes, ontology.TracesTo, ontology.Proves,
	}
	for _, et := range causal {
		if !ontology.IsCausal(et) {
			t.Errorf("expected %s to be causal", et)
		}
	}

	nonCausal := []ontology.EdgeType{
		ontology.AuthoredBy, ontology.ApprovedBy, ontology.BelongsTo,
	}
	for _, et := range nonCausal {
		if ontology.IsCausal(et) {
			t.Errorf("expected %s to be non-causal", et)
		}
	}
}
