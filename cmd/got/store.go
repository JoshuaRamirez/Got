package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
	"github.com/joshuaramirez/got/internal/repo"
)

// nameAttr is the reserved vertex/edge attribute under which the CLI stores a
// human-readable name. A content-addressed ID is sha256(name), which is
// one-way, so the name is carried inside the graph itself (preserved by the
// UC-S23 snapshot codec) to be recoverable for display after a reload.
const nameAttr = "got.name"

func schema() ontology.Schema { return ontology.NewDefaultSchema() }

// stateDir returns the repository directory: $GOT_DIR or ".got". The directory
// holds graph.json + namespace.json, managed by repo.SaveState / LoadState.
func stateDir() string {
	if d := os.Getenv("GOT_DIR"); d != "" {
		return d
	}
	return ".got"
}

// graphFilePath is the file repo.SaveState writes; its presence marks an
// initialized repository.
func graphFilePath() string { return filepath.Join(stateDir(), "graph.json") }

func repoInitialized() bool {
	_, err := os.Stat(graphFilePath())
	return err == nil
}

// vid maps a vertex name to its content-addressed VertexID.
func vid(name string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(name)))
}

// eid maps an edge name to its content-addressed EdgeID.
func eid(name string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte(name)))
}

// loadState loads the repository State (graph.json + a durable namespace
// FileStore over namespace.json). It errors if the repository is not
// initialized so callers surface the "run 'got init'" hint.
func loadState() (repo.State, error) {
	if !repoInitialized() {
		return nil, fmt.Errorf("no repository; run 'got init'")
	}
	return repo.LoadState(stateDir(), schema())
}

// saveState persists the state's graph value.
func saveState(state repo.State) error {
	return repo.SaveState(stateDir(), state)
}

// withName builds a graph.AttrMap carrying the human name plus any
// user-supplied attributes.
func withName(name string, user map[string]string) graph.AttrMap {
	m := make(graph.AttrMap, len(user)+1)
	for k, v := range user {
		m[k] = v
	}
	m[nameAttr] = name
	return m
}

// nameOf returns the human name stored on a vertex, falling back to a short
// hex id if none was recorded. Branch vertices carry their name under
// repo's "branch.name" attribute instead of got.name, so it is checked too.
func nameOf(v graph.Vertex) string {
	if n, ok := v.Attrs[nameAttr].(string); ok {
		return n
	}
	if n, ok := v.Attrs["branch.name"].(string); ok {
		return n
	}
	return shortID(v.ID[:])
}

// edgeNameOf returns the human name stored on an edge, or a short hex id.
func edgeNameOf(e graph.Edge) string {
	if n, ok := e.Attrs[nameAttr].(string); ok {
		return n
	}
	return shortID(e.ID[:])
}

// nameIndex maps each vertex ID to its human name for display.
func nameIndex(g graph.Graph) map[identity.VertexID]string {
	idx := make(map[identity.VertexID]string)
	for _, v := range g.Vertices() {
		idx[v.ID] = nameOf(v)
	}
	return idx
}

// vertexNamed returns the vertex with the given human name, if present.
func vertexNamed(g graph.Graph, name string) (graph.Vertex, bool) {
	return g.Vertex(vid(name))
}
