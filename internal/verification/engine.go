package verification

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
)

// Evaluator produces a ResultValue for a (frontier, environment) pair.
// Callers register one via NewEngine so the default engine can dispatch
// evaluation without knowing domain specifics.
type Evaluator interface {
	Evaluate(g graph.Graph, f projection.Frontier, env EnvironmentBinding) (ResultValue, error)
}

// EvaluatorFunc adapts a function into an Evaluator.
type EvaluatorFunc func(graph.Graph, projection.Frontier, EnvironmentBinding) (ResultValue, error)

// Evaluate implements Evaluator.
func (f EvaluatorFunc) Evaluate(g graph.Graph, fr projection.Frontier, env EnvironmentBinding) (ResultValue, error) {
	return f(g, fr, env)
}

// DefaultEngine evaluates frontiers via a registered Evaluator and issues
// certificates by delegating to a governance.Engine for the gate decision.
type DefaultEngine struct {
	evaluator  Evaluator
	governance governance.Engine
}

// NewEngine returns a default verification Engine. The evaluator is the
// domain-specific scoring function; the governance engine decides whether
// a frontier's evaluations meet policy obligations.
func NewEngine(gov governance.Engine, evaluator Evaluator) *DefaultEngine {
	return &DefaultEngine{
		evaluator:  evaluator,
		governance: gov,
	}
}

func (e *DefaultEngine) Evaluate(ctx context.Context, g graph.Graph, f projection.Frontier, env EnvironmentBinding) (Evaluation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e.evaluator == nil {
		return nil, fmt.Errorf("verification: no evaluator configured")
	}
	rv, err := e.evaluator.Evaluate(g, f, env)
	if err != nil {
		return nil, fmt.Errorf("verification: evaluator returned error: %w", err)
	}
	return &evaluation{target: f, env: env, result: rv}, nil
}

// Prove returns true iff there is a Proves edge from the proof vertex to
// the claim vertex in g, and no Refutes edge from the same proof. This is
// the minimum interpretation: a proof "proves" a claim iff the graph
// records that relationship.
func (e *DefaultEngine) Prove(ctx context.Context, g graph.Graph, c Claim, p Proof) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	claimID := c.ID()
	proofID := p.ID()
	if _, ok := g.Vertex(claimID); !ok {
		return false, fmt.Errorf("verification: claim vertex %v not in graph: %w", claimID, graph.ErrVertexNotFound)
	}
	if _, ok := g.Vertex(proofID); !ok {
		return false, fmt.Errorf("verification: proof vertex %v not in graph: %w", proofID, graph.ErrVertexNotFound)
	}
	proves, refutes := false, false
	for _, edge := range g.Edges() {
		if edge.From != proofID || edge.To != claimID {
			continue
		}
		switch edge.Type {
		case ontology.Proves:
			proves = true
		case ontology.Refutes:
			refutes = true
		}
	}
	return proves && !refutes, nil
}

// Certify issues a certificate iff governance.GateRelease(g, f, ps)
// returns true. Otherwise returns ErrCertificationFailed wrapped with the
// outstanding obligations.
func (e *DefaultEngine) Certify(ctx context.Context, g graph.Graph, f projection.Frontier, evals []Evaluation, ps []governance.Policy) (Certificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ok, obligations, err := e.governance.GateRelease(ctx, g, f, ps)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %d obligation(s) unmet", ErrCertificationFailed, len(obligations))
	}
	policyNames := make([]string, 0, len(ps))
	for _, p := range ps {
		policyNames = append(policyNames, p.Name())
	}
	return &certificate{
		target:   f,
		evidence: append([]Evaluation(nil), evals...),
		policies: policyNames,
	}, nil
}

// --- concrete data types ---

// evaluation is the default Evaluation: a wrapper over target, env, result.
type evaluation struct {
	target projection.Frontier
	env    EnvironmentBinding
	result ResultValue
}

func (e *evaluation) Target() projection.Frontier     { return e.target }
func (e *evaluation) Environment() EnvironmentBinding { return e.env }
func (e *evaluation) Result() ResultValue             { return e.result }

// certificate is the default Certificate.
type certificate struct {
	target   projection.Frontier
	evidence []Evaluation
	policies []string
}

func (c *certificate) Target() projection.Frontier { return c.target }
func (c *certificate) Evidence() []Evaluation      { return c.evidence }
func (c *certificate) Policies() []string          { return c.policies }

// --- ResultValue helpers ---

// WeightedAverageEvaluator composes child evaluators and returns the
// weighted average of their ScalarResult outputs. Non-scalar child
// results are skipped. Zero total weight returns ScalarResult(0).
type WeightedAverageEvaluator struct {
	Children []WeightedChild
}

// WeightedChild pairs a child Evaluator with its weight.
type WeightedChild struct {
	Weight    float64
	Evaluator Evaluator
}

// Evaluate runs each child, sums (weight * scalarResult), divides by the
// sum of weights for the children that returned ScalarResults. Errors
// from any child abort the call.
func (w WeightedAverageEvaluator) Evaluate(g graph.Graph, f projection.Frontier, env EnvironmentBinding) (ResultValue, error) {
	var sum, totalWeight float64
	for _, c := range w.Children {
		rv, err := c.Evaluator.Evaluate(g, f, env)
		if err != nil {
			return nil, err
		}
		s, ok := rv.(ScalarResult)
		if !ok {
			continue
		}
		sum += float64(s) * c.Weight
		totalWeight += c.Weight
	}
	if totalWeight == 0 {
		return ScalarResult(0), nil
	}
	return ScalarResult(sum / totalWeight), nil
}

// ScalarResult is a numeric ResultValue ordered by ordinary comparison
// when compared to another ScalarResult. Comparison against any other
// ResultValue type returns 0 (incomparable).
type ScalarResult float64

// Compare implements ResultValue.
func (s ScalarResult) Compare(other ResultValue) int {
	o, ok := other.(ScalarResult)
	if !ok {
		return 0
	}
	switch {
	case s < o:
		return -1
	case s > o:
		return 1
	default:
		return 0
	}
}
