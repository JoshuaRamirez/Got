package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/repo"
)

// snapshot is the on-disk repository state. Vertices and edges reference each
// other by human-readable name; a vertex's content-addressed VertexID is
// sha256(name), the same convention the library's tests use. Refs map a
// branch name to the vertex name it points at.
type snapshot struct {
	Vertices []vertexRec       `json:"vertices"`
	Edges    []edgeRec         `json:"edges"`
	Refs     map[string]string `json:"refs"`
}

type vertexRec struct {
	Name  string            `json:"name"`
	Type  string            `json:"type"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

type edgeRec struct {
	Name string `json:"name"`
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
}

// stateDir returns the repository directory: $GOT_DIR or ".got".
func stateDir() string {
	if d := os.Getenv("GOT_DIR"); d != "" {
		return d
	}
	return ".got"
}

func statePath() string { return filepath.Join(stateDir(), "state.json") }

// vid maps a vertex name to its content-addressed VertexID.
func vid(name string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(name)))
}

// eid maps an edge name to its content-addressed EdgeID.
func eid(name string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(name)))
}

// loadSnapshot reads the state file. It returns a clear error when the
// repository has not been initialized.
func loadSnapshot() (*snapshot, error) {
	b, err := os.ReadFile(statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no repository; run 'got init'")
		}
		return nil, err
	}
	var s snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("corrupt state file: %w", err)
	}
	if s.Refs == nil {
		s.Refs = make(map[string]string)
	}
	return &s, nil
}

// saveSnapshot writes the state file, creating the state directory if needed.
func saveSnapshot(s *snapshot) error {
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), b, 0o644)
}

// repoExists reports whether a state file is already present.
func repoExists() bool {
	_, err := os.Stat(statePath())
	return err == nil
}

// nameIndex maps each known VertexID back to its name for display.
func (s *snapshot) nameIndex() map[identity.VertexID]string {
	idx := make(map[identity.VertexID]string, len(s.Vertices))
	for _, v := range s.Vertices {
		idx[vid(v.Name)] = v.Name
	}
	return idx
}

// vertexByName returns the record for a vertex name, if present.
func (s *snapshot) vertexByName(name string) (vertexRec, bool) {
	for _, v := range s.Vertices {
		if v.Name == name {
			return v, true
		}
	}
	return vertexRec{}, false
}

// buildState reconstructs a repo.State (graph + namespace) from the snapshot.
// The graph is built with the bulk Builder; refs are rebound into a fresh
// namespace store.
func (s *snapshot) buildState() (repo.State, error) {
	schema := ontology.NewDefaultSchema()
	b := graph.NewBuilder(schema)

	for _, v := range s.Vertices {
		b.AddVertex(graph.Vertex{
			ID:    vid(v.Name),
			Type:  ontology.VertexType(v.Type),
			Attrs: attrMap(v.Attrs),
		})
	}
	for _, e := range s.Edges {
		if err := b.AddEdge(graph.Edge{
			ID:   eid(e.Name),
			Type: ontology.EdgeType(e.Type),
			From: vid(e.From),
			To:   vid(e.To),
		}); err != nil {
			return nil, fmt.Errorf("edge %q: %w", e.Name, err)
		}
	}
	g := b.Build()

	ns := namespace.NewStore()
	for ref, target := range s.Refs {
		if err := ns.BindRef(context.Background(), namespace.RefName(ref), vid(target)); err != nil {
			return nil, err
		}
	}
	return repo.NewState(g, ns), nil
}

// attrMap converts the string-valued snapshot attrs into a graph.AttrMap.
// Returns nil for an empty map so vertices with no attrs compare cleanly.
func attrMap(m map[string]string) graph.AttrMap {
	if len(m) == 0 {
		return nil
	}
	out := make(graph.AttrMap, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
