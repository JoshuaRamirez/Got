// Package namespace implements the NamespaceControl specification.
//
// It is the only mutable component in the architecture. It manages bindings
// from human-readable names (refs, aliases, projection handles) to immutable
// vertex identifiers in the graph.
//
// Imports: internal/identity only.
// Must not import: projection, graph, or repo.
package namespace

import (
	"context"

	"github.com/joshuaramirez/got/internal/identity"
)

// RefName is a mutable reference name (analogous to a branch pointer).
type RefName string

// Alias is a mutable alias (analogous to a tag or release name).
type Alias string

// ProjectionHandle is a named handle for a stored projection specification.
type ProjectionHandle string

// Store manages the mutable namespace state. All bindings map a name to an
// immutable vertex identifier.
//
// Per docs/design-rules.md, Store is the named exception that takes
// context.Context on every method: it is the only stateful interface in the
// architecture and may be backed by a remote or persistent store.
//
// Axiom: ResolveRef(r) == (v, true) after BindRef(r, v) succeeds.
// Axiom: ResolveAlias(a) == (v, true) after BindAlias(a, v) succeeds.
type Store interface {
	BindRef(ctx context.Context, name RefName, id identity.VertexID) error
	ResolveRef(ctx context.Context, name RefName) (identity.VertexID, bool)

	// DeleteRef removes a ref binding. Deleting an absent ref is a no-op
	// (idempotent). Used by branch delete/rename.
	DeleteRef(ctx context.Context, name RefName) error

	BindAlias(ctx context.Context, alias Alias, id identity.VertexID) error
	ResolveAlias(ctx context.Context, alias Alias) (identity.VertexID, bool)

	BindProjection(ctx context.Context, handle ProjectionHandle, id identity.VertexID) error
	ResolveProjection(ctx context.Context, handle ProjectionHandle) (identity.VertexID, bool)
}
