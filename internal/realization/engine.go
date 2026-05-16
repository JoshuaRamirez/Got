package realization

import (
	"context"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/projection"
)

// Materializer produces a Bundle for one Target from a projected view's
// subgraph. Users register a Materializer per target via
// DefaultEngine.Register.
type Materializer interface {
	Materialize(graph.Subgraph) (Bundle, error)
}

// MaterializerFunc adapts a function into a Materializer.
type MaterializerFunc func(graph.Subgraph) (Bundle, error)

// Materialize implements Materializer.
func (f MaterializerFunc) Materialize(s graph.Subgraph) (Bundle, error) { return f(s) }

// ManifestTarget is the standard "list every vertex as a path with itself
// as provenance" target. NewEngine registers a materializer for it.
const ManifestTarget Target = "manifest"

// DefaultEngine is the default Engine implementation: a Target →
// Materializer registry. Targets without a registered materializer fail
// with ErrTargetUnsupported. Use NewEngine to construct one preloaded
// with the manifest target; use Register to add more.
type DefaultEngine struct {
	registry map[Target]Materializer
}

// NewEngine returns a DefaultEngine preloaded with ManifestTarget.
func NewEngine() *DefaultEngine {
	e := &DefaultEngine{registry: make(map[Target]Materializer)}
	e.Register(ManifestTarget, MaterializerFunc(manifestMaterialize))
	return e
}

// Register associates a Materializer with a Target. Registering the same
// Target twice overwrites the prior registration.
func (e *DefaultEngine) Register(t Target, m Materializer) {
	e.registry[t] = m
}

// Materialize satisfies the Engine interface. It looks up the registered
// Materializer for target and delegates.
func (e *DefaultEngine) Materialize(ctx context.Context, v projection.View, t Target) (Bundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, ok := e.registry[t]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrTargetUnsupported, t)
	}
	return m.Materialize(v.Subgraph())
}

// manifestMaterialize is the Materializer registered for ManifestTarget.
// It emits one path per vertex (formatted as the hex of the VertexID)
// with that vertex as its sole provenance witness.
func manifestMaterialize(sub graph.Subgraph) (Bundle, error) {
	paths := make([]string, 0, len(sub.VertexIDs()))
	prov := make(map[string][]identity.VertexID, len(sub.VertexIDs()))
	for _, id := range sub.VertexIDs() {
		path := fmt.Sprintf("vertex/%x", [32]byte(id))
		paths = append(paths, path)
		prov[path] = []identity.VertexID{id}
	}
	return &mapBundle{
		target:   ManifestTarget,
		paths:    paths,
		prov:     prov,
		fidelity: FidelityContract{Name: "lossless-manifest"},
	}, nil
}

// mapBundle is a default Bundle backed by a map of path → provenance.
type mapBundle struct {
	target   Target
	paths    []string
	prov     map[string][]identity.VertexID
	fidelity FidelityContract
}

func (b *mapBundle) Target() Target                             { return b.target }
func (b *mapBundle) Paths() []string                            { return b.paths }
func (b *mapBundle) Provenance(path string) []identity.VertexID { return b.prov[path] }
func (b *mapBundle) Fidelity() FidelityContract                 { return b.fidelity }
