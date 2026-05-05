package namespace

import (
	"context"

	"github.com/joshuaramirez/got/internal/identity"
)

// memStore is an in-memory implementation of Store.
// It is the sole mutable component in the architecture.
type memStore struct {
	refs    map[RefName]identity.VertexID
	aliases map[Alias]identity.VertexID
	projs   map[ProjectionHandle]identity.VertexID
}

// NewStore creates an empty in-memory namespace store.
func NewStore() Store {
	return &memStore{
		refs:    make(map[RefName]identity.VertexID),
		aliases: make(map[Alias]identity.VertexID),
		projs:   make(map[ProjectionHandle]identity.VertexID),
	}
}

func (s *memStore) BindRef(_ context.Context, name RefName, id identity.VertexID) error {
	s.refs[name] = id
	return nil
}

func (s *memStore) ResolveRef(_ context.Context, name RefName) (identity.VertexID, bool) {
	id, ok := s.refs[name]
	return id, ok
}

func (s *memStore) BindAlias(_ context.Context, name Alias, id identity.VertexID) error {
	s.aliases[name] = id
	return nil
}

func (s *memStore) ResolveAlias(_ context.Context, name Alias) (identity.VertexID, bool) {
	id, ok := s.aliases[name]
	return id, ok
}

func (s *memStore) BindProjection(_ context.Context, name ProjectionHandle, id identity.VertexID) error {
	s.projs[name] = id
	return nil
}

func (s *memStore) ResolveProjection(_ context.Context, name ProjectionHandle) (identity.VertexID, bool) {
	id, ok := s.projs[name]
	return id, ok
}
