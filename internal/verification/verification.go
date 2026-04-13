// Package verification implements the VerificationSystem specification.
//
// Verification evaluates frontiers in a given environment, proves claims, and
// issues certificates. Certification is separate from evaluation: a certificate
// is only issued when all policy obligations are discharged.
//
// Categorically, for a fixed environment E, evaluation is a functor:
//
//	Eval_E : Front_Pi -> Res
//
// Certificates are objects in a fiber over a frontier, ordered by evidential
// strength.
//
// Imports: internal/graph, internal/projection, internal/governance, internal/identity.
// Must not import: composition, realization, repo.
package verification

import (
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/projection"
)

// EnvironmentBinding identifies a specific execution environment.
type EnvironmentBinding struct {
	ID      identity.VertexID
	Version string
}

// ResultValue is the outcome of an evaluation, with a total order for
// comparison (pass/fail/score).
type ResultValue interface {
	Compare(other ResultValue) int
}

// Evaluation is the result of evaluating a frontier in a given environment.
//
// Axiom: target(evaluate(G, F, E)) = F and environment(evaluate(G, F, E)) = E.
type Evaluation interface {
	Target() projection.Frontier
	Environment() EnvironmentBinding
	Result() ResultValue
}

// Claim is a proposition about the graph that can be proved or refuted.
type Claim interface {
	ID() identity.VertexID
}

// Proof is evidence for or against a Claim.
type Proof interface {
	ID() identity.VertexID
}

// Certificate attests that a frontier satisfies a set of policies, backed by
// evaluation evidence.
//
// Axiom: certify(G, F, Es, Ps) = some(C) => certTarget(C) = F and
//
//	check(G, F, Ps) = Sat.
type Certificate interface {
	Target() projection.Frontier
	Evidence() []Evaluation
	Policies() []string
}

// Engine performs evaluations, proves claims, and issues certificates.
type Engine interface {
	// Evaluate runs an evaluation of the frontier in the given environment.
	Evaluate(g graph.Graph, f projection.Frontier, env EnvironmentBinding) (Evaluation, error)

	// Prove checks whether the given proof validates the given claim.
	Prove(g graph.Graph, c Claim, p Proof) (bool, error)

	// Certify attempts to issue a certificate for the frontier, given a set
	// of evaluations and policies. Returns nil if obligations are not met.
	Certify(g graph.Graph, f projection.Frontier, evals []Evaluation, ps []governance.Policy) (Certificate, error)
}
