package composition

import (
	"context"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/projection"
)

// Auditor is the optional capability a composition Engine may satisfy to
// run the in-graph structural/temporal well-formedness audit over a
// frontier independently of a merge. It is kept off the core Engine
// interface so the two-way contract stays minimal; callers type-assert
// (or use *DefaultEngine directly).
//
// The audit is the same in-graph check Strict Merge runs before the
// governance gate: it detects Structural conflicts (distinct edges sharing
// an endpoint pair with incompatible types) and Temporal conflicts
// (malformed TimeTriple) reachable from the frontier. It does NOT run the
// per-side edit audit (Textual/Trust/Schema) — that needs two frontiers.
type Auditor interface {
	Audit(ctx context.Context, g graph.Graph, f projection.Frontier) ([]Conflict, error)
}

var _ Auditor = (*DefaultEngine)(nil)

// Audit runs the in-graph structural/temporal audit over the frontier and
// returns any conflicts found. Unlike Merge, Audit ignores the engine's
// strictness setting — it is an explicit, always-on check. An empty result
// means the frontier is structurally and temporally well-formed in g.
func (e *DefaultEngine) Audit(ctx context.Context, g graph.Graph, f projection.Frontier) ([]Conflict, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return strictAudit(g, f.VertexIDs()), nil
}
