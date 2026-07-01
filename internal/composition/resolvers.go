package composition

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/projection"
)

// Payloaded is an optional interface a Conflict may satisfy to expose a
// typed payload describing the disagreement. Resolvers type-assert to
// Payloaded and switch on the concrete payload type to act per-medium.
//
// auditConflict satisfies Payloaded. The free-text Detail() is still
// available for human-readable rendering.
type Payloaded interface {
	Conflict
	Payload() any
}

// --- typed payloads, one per ConflictKind the audits emit ---

// SchemaPayload carries the disagreeing types for a Schema conflict.
type SchemaPayload struct {
	Vertex    identity.VertexID
	LeftType  ontology.VertexType
	RightType ontology.VertexType
}

// TextualPayload carries the per-key Attrs disagreement for a Textual
// conflict.
type TextualPayload struct {
	Vertex identity.VertexID
	Key    string
	Left   any
	Right  any
}

// TrustPayload carries the disagreeing TrustAnnotations for a Trust
// conflict.
type TrustPayload struct {
	Vertex identity.VertexID
	Left   graph.TrustAnnotation
	Right  graph.TrustAnnotation
}

// TemporalPayload carries the disagreeing TimeTriples for a Temporal
// conflict. For in-graph audits Left is the offending value and Right
// is the zero TimeTriple.
type TemporalPayload struct {
	Vertex identity.VertexID
	Left   graph.TimeTriple
	Right  graph.TimeTriple
}

// StructuralPayload carries the disagreeing edge types on the same
// endpoint pair.
type StructuralPayload struct {
	From      identity.VertexID
	To        identity.VertexID
	LeftType  ontology.EdgeType
	RightType ontology.EdgeType
}

// CapabilityPayload carries a per-key Attrs disagreement on a
// Capability-typed vertex — the two branches define the capability
// differently. Structurally identical to TextualPayload but tagged with
// the Capability kind so resolvers can route on semantics.
type CapabilityPayload struct {
	Vertex identity.VertexID
	Key    string
	Left   any
	Right  any
}

// EvaluationPayload carries a per-key Attrs disagreement on an
// Evaluation-typed vertex — the two branches disagree on an evaluation
// result. Tagged with the Evaluation kind.
type EvaluationPayload struct {
	Vertex identity.VertexID
	Key    string
	Left   any
	Right  any
}

// --- Resolver framework ---

// Resolver is a typed conflict handler. AppliesTo names the
// ConflictKind this resolver handles; Apply receives the host graph,
// the offending Conflict, and (when supplied by the caller) the
// original per-side EditedFrontiers that produced the conflict.
//
// Per-side audit conflicts (Schema/Textual/Trust/Temporal/Structural
// from frontier edits) are resolved by mutating one side's edit map so
// the two sides agree on the disputed field. In-graph audit conflicts
// (Structural from edge collisions, Temporal from malformed TimeTriple)
// are resolved by mutating the graph itself. Resolvers may do either or
// both; they return the (possibly) mutated graph.
//
// If left or right is nil, per-side resolvers should be a no-op and
// return the graph unchanged. ResolveTyped does not require both
// frontiers to be EditedFrontiers — the field is supplied separately
// from the MergeResult.
type Resolver interface {
	AppliesTo() ConflictKind
	Apply(ctx context.Context, g graph.Graph, c Conflict, left, right *projection.EditedFrontier) (graph.Graph, error)
}

// ResolveTyped applies resolvers to the conflicts in mr and re-merges
// against the resulting graph and the (possibly mutated) per-side
// frontiers. Each Conflict in mr.Conflicts is matched against the
// resolvers by ConflictKind; the first matching resolver runs.
// Conflicts with no matching resolver are left in place and re-surface
// on the re-merge (intended behavior).
//
// The original left and right EditedFrontiers must be passed in so the
// re-merge has access to the per-side data the initial Merge used. The
// frontiers may be nil if the original Merge did not use per-side data;
// per-side resolvers become no-ops in that case.
//
// **Mutation note**: per-side resolvers (PreferLeftAttr,
// PreferHigherTrust, etc.) mutate the supplied left/right frontiers in
// place. Callers that want to retry ResolveTyped against the original
// per-side data with a different resolver set should
// projection.EditedFrontier.Clone() the inputs first.
//
// Unlike Resolve, which applies generic graph-mutating Resolutions
// blindly, ResolveTyped routes each conflict to the resolver that
// claims its kind. This makes typed resolution composable: callers
// declare a small library of per-medium policies and reuse them.
func (e *DefaultEngine) ResolveTyped(ctx context.Context, g graph.Graph, left, right *projection.EditedFrontier, mr MergeResult, resolvers []Resolver) (MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return MergeResult{}, err
	}

	byKind := make(map[ConflictKind]Resolver, len(resolvers))
	for _, r := range resolvers {
		byKind[r.AppliesTo()] = r
	}

	current := g
	for _, c := range mr.Conflicts {
		r, ok := byKind[c.Kind()]
		if !ok {
			continue
		}
		next, err := r.Apply(ctx, current, c, left, right)
		if err != nil {
			return MergeResult{}, fmt.Errorf("%w: %s resolver failed: %v", ErrConflictUnresolvable, c.Kind(), err)
		}
		current = next
	}

	// Re-merge with the (possibly mutated) frontiers. Nil frontiers
	// fall back to the historical boundary-only behavior.
	var lf, rf projection.Frontier
	if left != nil {
		lf = left
	} else {
		var boundary []identity.VertexID
		for _, c := range mr.Conflicts {
			boundary = append(boundary, c.Boundary()...)
		}
		lf = &mergedFrontier{ids: dedupe(boundary)}
	}
	if right != nil {
		rf = right
	} else {
		rf = lf
	}
	return e.Merge(ctx, current, lf, rf, nil)
}

// --- stock resolvers ---

// PreferLeftAttr resolves a Textual conflict on the given Attrs key by
// copying the left side's value onto the right side's edit map. The
// graph is also updated to carry the chosen value. After this resolver
// runs, the per-side audit for that key sees agreement.
//
// If either left or right is nil, or the conflict's payload key does
// not match, the resolver is a no-op.
func PreferLeftAttr(key string) Resolver {
	return preferLeftAttrResolver{key: key}
}

type preferLeftAttrResolver struct{ key string }

func (preferLeftAttrResolver) AppliesTo() ConflictKind { return Textual }
func (r preferLeftAttrResolver) Apply(_ context.Context, g graph.Graph, c Conflict, left, right *projection.EditedFrontier) (graph.Graph, error) {
	p, ok := payloadOf[TextualPayload](c)
	if !ok || p.Key != r.key {
		return g, nil
	}
	if right != nil {
		rv, ok := right.Vertices[p.Vertex]
		if ok {
			if rv.Attrs == nil {
				rv.Attrs = make(graph.AttrMap, 1)
			}
			rv.Attrs[r.key] = p.Left
			right.Vertices[p.Vertex] = rv
		}
	}
	if v, vok := g.Vertex(p.Vertex); vok {
		if v.Attrs == nil {
			v.Attrs = make(graph.AttrMap, 1)
		}
		v.Attrs[r.key] = p.Left
		return g.WithVertex(v)
	}
	return g, nil
}

// PreferHigherTrust resolves a Trust conflict by keeping the
// higher-Score TrustAnnotation on both sides. Ties go to Left.
func PreferHigherTrust() Resolver {
	return preferTrustResolver{higher: true}
}

// PreferLowerTrust is the mirror of PreferHigherTrust — keeps the
// lower-Score TrustAnnotation. Useful when "least privilege" is the
// merge policy. Ties go to Left.
func PreferLowerTrust() Resolver {
	return preferTrustResolver{higher: false}
}

type preferTrustResolver struct{ higher bool }

func (preferTrustResolver) AppliesTo() ConflictKind { return Trust }
func (r preferTrustResolver) Apply(_ context.Context, g graph.Graph, c Conflict, left, right *projection.EditedFrontier) (graph.Graph, error) {
	p, ok := payloadOf[TrustPayload](c)
	if !ok {
		return g, nil
	}
	chosen := p.Left
	if r.higher {
		if p.Right.Score > p.Left.Score {
			chosen = p.Right
		}
	} else {
		if p.Right.Score < p.Left.Score {
			chosen = p.Right
		}
	}
	if left != nil {
		if v, ok := left.Vertices[p.Vertex]; ok {
			v.Trust = chosen
			left.Vertices[p.Vertex] = v
		}
	}
	if right != nil {
		if v, ok := right.Vertices[p.Vertex]; ok {
			v.Trust = chosen
			right.Vertices[p.Vertex] = v
		}
	}
	if v, vok := g.Vertex(p.Vertex); vok {
		v.Trust = chosen
		return g.WithVertex(v)
	}
	return g, nil
}

// PreferRightAttr is the mirror of PreferLeftAttr — copies the right
// value onto the left side and the graph for the named attr key.
func PreferRightAttr(key string) Resolver {
	return preferRightAttrResolver{key: key}
}

type preferRightAttrResolver struct{ key string }

func (preferRightAttrResolver) AppliesTo() ConflictKind { return Textual }
func (r preferRightAttrResolver) Apply(_ context.Context, g graph.Graph, c Conflict, left, right *projection.EditedFrontier) (graph.Graph, error) {
	p, ok := payloadOf[TextualPayload](c)
	if !ok || p.Key != r.key {
		return g, nil
	}
	if left != nil {
		lv, ok := left.Vertices[p.Vertex]
		if ok {
			if lv.Attrs == nil {
				lv.Attrs = make(graph.AttrMap, 1)
			}
			lv.Attrs[r.key] = p.Right
			left.Vertices[p.Vertex] = lv
		}
	}
	if v, vok := g.Vertex(p.Vertex); vok {
		if v.Attrs == nil {
			v.Attrs = make(graph.AttrMap, 1)
		}
		v.Attrs[r.key] = p.Right
		return g.WithVertex(v)
	}
	return g, nil
}

// PreferEarlierTime resolves a Temporal conflict by keeping the earlier
// ValidFrom on both sides. Ties go to Left.
func PreferEarlierTime() Resolver {
	return preferTimeResolver{earlier: true}
}

// PreferLaterTime is the mirror of PreferEarlierTime — keeps the later
// ValidFrom on both sides. Ties go to Left.
func PreferLaterTime() Resolver {
	return preferTimeResolver{earlier: false}
}

type preferTimeResolver struct{ earlier bool }

func (preferTimeResolver) AppliesTo() ConflictKind { return Temporal }
func (r preferTimeResolver) Apply(_ context.Context, g graph.Graph, c Conflict, left, right *projection.EditedFrontier) (graph.Graph, error) {
	p, ok := payloadOf[TemporalPayload](c)
	if !ok {
		return g, nil
	}
	chosen := p.Left
	if r.earlier {
		if p.Right.ValidFrom < p.Left.ValidFrom {
			chosen = p.Right
		}
	} else {
		if p.Right.ValidFrom > p.Left.ValidFrom {
			chosen = p.Right
		}
	}
	if left != nil {
		if v, ok := left.Vertices[p.Vertex]; ok {
			v.Time = chosen
			left.Vertices[p.Vertex] = v
		}
	}
	if right != nil {
		if v, ok := right.Vertices[p.Vertex]; ok {
			v.Time = chosen
			right.Vertices[p.Vertex] = v
		}
	}
	if v, vok := g.Vertex(p.Vertex); vok {
		v.Time = chosen
		return g.WithVertex(v)
	}
	return g, nil
}

// RejectSchemaConflict is a resolver that does NOT auto-resolve Schema
// conflicts — Type changes are semantically dangerous and usually
// require human review. Instead, it returns an error so ResolveTyped
// reports ErrConflictUnresolvable up the stack. Use this when callers
// want Schema conflicts to fail loudly rather than persist as
// unresolved noise in the re-merge.
func RejectSchemaConflict() Resolver {
	return rejectSchemaResolver{}
}

type rejectSchemaResolver struct{}

func (rejectSchemaResolver) AppliesTo() ConflictKind { return Schema }
func (rejectSchemaResolver) Apply(_ context.Context, g graph.Graph, c Conflict, _, _ *projection.EditedFrontier) (graph.Graph, error) {
	if p, ok := payloadOf[SchemaPayload](c); ok {
		return g, fmt.Errorf("schema disagreement on %v: %q vs %q (no auto-resolution)",
			p.Vertex, p.LeftType, p.RightType)
	}
	return g, fmt.Errorf("schema conflict (no payload available)")
}

// payloadOf is a small generic helper for type-asserting a Payloaded
// conflict to a specific payload type. Returns the zero value plus
// ok=false if the conflict is not Payloaded or its payload is not P.
func payloadOf[P any](c Conflict) (P, bool) {
	var zero P
	pl, ok := c.(Payloaded)
	if !ok {
		return zero, false
	}
	p, ok := pl.Payload().(P)
	if !ok {
		return zero, false
	}
	return p, true
}
