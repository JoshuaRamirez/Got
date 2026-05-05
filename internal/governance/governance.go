// Package governance implements the GovernanceSystem specification.
//
// Governance restricts the ambient repository category to an admissible
// subcategory defined by a set of policies. It checks policy satisfaction
// against a frontier and computes outstanding obligations.
//
// Categorically, for a policy set Pi, governance defines the full subcategory:
//
//	Repo_Pi subset Repo_Sigma
//
// consisting of all objects and morphisms that satisfy Pi.
//
// Imports: internal/graph, internal/projection, internal/identity.
// Must not import: verification, composition, realization, repo.
package governance

import (
	"context"
	"errors"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/projection"
)

// ErrPolicyViolation is returned when a policy aggregate decision is Unsat
// at a release gate or other hard policy boundary.
var ErrPolicyViolation = errors.New("governance: policy violation")

// Decision is the three-valued outcome of a policy check.
type Decision uint8

const (
	// Unsat means the policy is not satisfied.
	Unsat Decision = iota
	// Unknown means the policy cannot be decided with available information.
	Unknown
	// Sat means the policy is satisfied.
	Sat
)

// Obligation records a specific requirement that must be discharged
// before a frontier can pass governance.
type Obligation struct {
	Code   string
	Detail string
}

// Policy is a named governance rule that can be checked against a frontier.
//
// Per docs/design-rules.md, Policy is a value type (a rule), not an Engine,
// so its Check method does not take context.Context. The Engine that runs
// policies is responsible for honoring cancellation.
//
// Axiom: admissible(G, F, Ps) = true <=> check(G, F, Ps) = Sat.
type Policy interface {
	Name() string
	Check(g graph.Graph, f projection.Frontier) (Decision, []Obligation, error)
}

// Engine evaluates policies against frontiers.
//
// Axiom: gateRelease(G, F, Ps) = true => check(G, F, Ps) = Sat.
type Engine interface {
	// Check evaluates all policies against the frontier and returns the
	// aggregate decision along with any outstanding obligations.
	Check(ctx context.Context, g graph.Graph, f projection.Frontier, ps []Policy) (Decision, []Obligation, error)

	// GateRelease checks whether the frontier is eligible for release
	// under the given policies. A true result implies Check returns Sat.
	GateRelease(ctx context.Context, g graph.Graph, f projection.Frontier, ps []Policy) (bool, []Obligation, error)
}
