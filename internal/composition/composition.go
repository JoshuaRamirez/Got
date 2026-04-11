// Package composition implements the CompositionSystem specification.
//
// Composition merges two frontiers under governance constraints. When a
// conflict-free merge exists it is computed as a guarded pushout in the policy
// subcategory. When no admissible pushout exists, typed conflicts are returned.
//
// Categorically, merge is a Kleisli morphism over the conflict monad:
//   T(X) = X + Conf_Pi
//   Merge_Pi : Front_Pi x Front_Pi -> Kl(T)
//
// The XOR invariant — either merged or conflicted, never both — follows
// structurally from the monad.
//
// Imports: internal/graph, internal/projection, internal/governance,
//
//	internal/verification, internal/revision, internal/identity.
//
// Must not import: realization or repo.
package composition

import (
	"github.com/joshuaramirez/got/internal/governance"
	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/verification"
)

// ConflictKind classifies the nature of a merge conflict.
type ConflictKind string

const (
	Textual    ConflictKind = "textual"
	Structural ConflictKind = "structural"
	Schema     ConflictKind = "schema"
	Policy     ConflictKind = "policy"
	Trust      ConflictKind = "trust"
	Capability ConflictKind = "capability"
	Evaluation ConflictKind = "evaluation"
	Temporal   ConflictKind = "temporal"
)

// Conflict describes a single merge conflict, including its kind and the
// boundary vertices involved.
type Conflict interface {
	Kind() ConflictKind
	Boundary() []identity.VertexID
}

// Resolution is a strategy for resolving a conflict by transforming the graph.
type Resolution interface {
	Apply(g graph.Graph) (graph.Graph, error)
}

// MergeWitness is a vertex that attests to a completed merge.
type MergeWitness interface {
	ID() identity.VertexID
}

// MergeResult holds the outcome of a merge operation.
//
// Axiom: (merged(MR) != none) xor (conflicts(MR) != {}).
// Axiom: merged(MR) = some(FM) => check(G, FM, Ps) = Sat.
type MergeResult struct {
	Frontier    projection.Frontier
	Witness     MergeWitness
	Certificate verification.Certificate
	Conflicts   []Conflict
}

// Engine performs merge and conflict resolution.
type Engine interface {
	// Merge computes the guarded pushout of two frontiers under the given
	// policies. Returns either a merged frontier (with witness and certificate)
	// or a set of typed conflicts — never both.
	Merge(g graph.Graph, left, right projection.Frontier, ps []governance.Policy) (MergeResult, error)

	// Resolve attempts to apply a set of resolutions to an existing conflict
	// result, producing a new MergeResult.
	Resolve(g graph.Graph, mr MergeResult, rs []Resolution) (MergeResult, error)
}
