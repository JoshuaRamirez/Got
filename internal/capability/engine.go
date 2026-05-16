package capability

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// Predicate decides whether a (frontier, policies, certificates) triple
// triggers an emergence. If it returns true, the supplied Witness names
// the emerged capability. Predicates are registered with DefaultEngine
// and evaluated in order.
type Predicate interface {
	Emerges(g graph.Graph, f projection.Frontier, ps []governance.Policy, cs []verification.Certificate) (bool, Witness)
}

// PredicateFunc adapts a function into a Predicate.
type PredicateFunc func(graph.Graph, projection.Frontier, []governance.Policy, []verification.Certificate) (bool, Witness)

// Emerges implements Predicate.
func (f PredicateFunc) Emerges(g graph.Graph, fr projection.Frontier, ps []governance.Policy, cs []verification.Certificate) (bool, Witness) {
	return f(g, fr, ps, cs)
}

// DefaultEngine evaluates a sequence of registered Predicates and returns
// the first match.
type DefaultEngine struct {
	predicates []Predicate
}

// NewEngine returns a default capability Engine with the supplied
// predicates evaluated in order.
func NewEngine(predicates ...Predicate) *DefaultEngine {
	return &DefaultEngine{predicates: append([]Predicate(nil), predicates...)}
}

// Register appends a predicate. Predicates fire in registration order.
func (e *DefaultEngine) Register(p Predicate) {
	e.predicates = append(e.predicates, p)
}

func (e *DefaultEngine) Emerges(ctx context.Context, g graph.Graph, f projection.Frontier, ps []governance.Policy, cs []verification.Certificate) (bool, Witness, error) {
	if err := ctx.Err(); err != nil {
		return false, Witness{}, err
	}
	for _, pred := range e.predicates {
		if err := ctx.Err(); err != nil {
			return false, Witness{}, err
		}
		ok, w := pred.Emerges(g, f, ps, cs)
		if ok {
			return true, w, nil
		}
	}
	return false, Witness{}, fmt.Errorf("%w: no predicate fired", ErrNoEmergence)
}

// CertifiedNonEmpty is a useful out-of-the-box predicate: emerges when
// the frontier has at least one vertex AND at least one certificate is
// present. The witness is named by the supplied label.
func CertifiedNonEmpty(label string) Predicate {
	return PredicateFunc(func(_ graph.Graph, f projection.Frontier, _ []governance.Policy, cs []verification.Certificate) (bool, Witness) {
		if len(f.VertexIDs()) > 0 && len(cs) > 0 {
			return true, Witness{Name: label}
		}
		return false, Witness{}
	})
}
