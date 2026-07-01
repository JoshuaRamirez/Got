package namespace

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/joshuaramirez/got/internal/identity"
)

// FileStore is a durable, concurrency-safe implementation of Store backed by
// a single JSON file. It is the persistent counterpart to the in-memory
// memStore: bindings survive process restarts (each mutation is flushed to
// disk with an atomic write-then-rename) and every method holds a mutex, so
// unlike memStore it is safe for concurrent writers.
//
// The namespace is the sole mutable component of the architecture (the graph
// is content-addressed and reconstructable), so persisting it is the
// meaningful durability boundary for a repository.
type FileStore struct {
	path    string
	mu      sync.Mutex
	refs    map[RefName]identity.VertexID
	aliases map[Alias]identity.VertexID
	projs   map[ProjectionHandle]identity.VertexID
}

// fileState is the on-disk shape: names mapped to hex-encoded vertex IDs.
type fileState struct {
	Refs        map[string]string `json:"refs"`
	Aliases     map[string]string `json:"aliases"`
	Projections map[string]string `json:"projections"`
}

// NewFileStore opens (or creates) a FileStore backed by the given path. An
// existing file is loaded; a missing file starts empty and is created on the
// first bind.
func NewFileStore(path string) (*FileStore, error) {
	s := &FileStore{
		path:    path,
		refs:    make(map[RefName]identity.VertexID),
		aliases: make(map[Alias]identity.VertexID),
		projs:   make(map[ProjectionHandle]identity.VertexID),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileStore) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var st fileState
	if err := json.Unmarshal(b, &st); err != nil {
		return fmt.Errorf("namespace: corrupt store %q: %w", s.path, err)
	}
	for k, v := range st.Refs {
		id, err := decodeID(v)
		if err != nil {
			return err
		}
		s.refs[RefName(k)] = id
	}
	for k, v := range st.Aliases {
		id, err := decodeID(v)
		if err != nil {
			return err
		}
		s.aliases[Alias(k)] = id
	}
	for k, v := range st.Projections {
		id, err := decodeID(v)
		if err != nil {
			return err
		}
		s.projs[ProjectionHandle(k)] = id
	}
	return nil
}

// save writes the current state atomically. The caller must hold s.mu.
func (s *FileStore) save() error {
	st := fileState{
		Refs:        make(map[string]string, len(s.refs)),
		Aliases:     make(map[string]string, len(s.aliases)),
		Projections: make(map[string]string, len(s.projs)),
	}
	for k, v := range s.refs {
		st.Refs[string(k)] = encodeID(v)
	}
	for k, v := range s.aliases {
		st.Aliases[string(k)] = encodeID(v)
	}
	for k, v := range s.projs {
		st.Projections[string(k)] = encodeID(v)
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *FileStore) BindRef(_ context.Context, name RefName, id identity.VertexID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refs[name] = id
	return s.save()
}

func (s *FileStore) ResolveRef(_ context.Context, name RefName) (identity.VertexID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.refs[name]
	return id, ok
}

func (s *FileStore) BindAlias(_ context.Context, name Alias, id identity.VertexID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.aliases[name] = id
	return s.save()
}

func (s *FileStore) ResolveAlias(_ context.Context, name Alias) (identity.VertexID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.aliases[name]
	return id, ok
}

func (s *FileStore) BindProjection(_ context.Context, name ProjectionHandle, id identity.VertexID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projs[name] = id
	return s.save()
}

func (s *FileStore) ResolveProjection(_ context.Context, name ProjectionHandle) (identity.VertexID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.projs[name]
	return id, ok
}

func encodeID(id identity.VertexID) string {
	return hex.EncodeToString(id[:])
}

func decodeID(s string) (identity.VertexID, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return identity.VertexID{}, fmt.Errorf("namespace: bad hex id %q: %w", s, err)
	}
	if len(b) != len(identity.VertexID{}) {
		return identity.VertexID{}, fmt.Errorf("namespace: id %q has wrong length %d", s, len(b))
	}
	var id identity.VertexID
	copy(id[:], b)
	return id, nil
}
