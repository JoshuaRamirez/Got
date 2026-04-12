// Package namespace implements the NamespaceControl specification.
//
// It is the only mutable component in the architecture. It manages bindings
// from human-readable names (refs, aliases, projection handles) to immutable
// vertex identifiers in the graph.
//
// Imports: internal/identity only.
// Must not import: projection, graph, or repo.
package namespace

import "github.com/joshuaramirez/got/internal/identity"

// RefName is a mutable reference name (analogous to a branch pointer).
type RefName string

// Alias is a mutable alias (analogous to a tag or release name).
type Alias string

// ProjectionHandle is a named handle for a stored projection specification.
type ProjectionHandle string

// Store manages the mutable namespace state. All bindings map a name to an
// immutable vertex identifier.
//
// Axiom: ResolveRef(r) == (v, true) after BindRef(r, v) succeeds.
// Axiom: ResolveAlias(a) == (v, true) after BindAlias(a, v) succeeds.
type Store interface {
	BindRef(RefName, identity.VertexID) error
	ResolveRef(RefName) (identity.VertexID, bool)

	BindAlias(Alias, identity.VertexID) error
	ResolveAlias(Alias) (identity.VertexID, bool)

	BindProjection(ProjectionHandle, identity.VertexID) error
	ResolveProjection(ProjectionHandle) (identity.VertexID, bool)
}
