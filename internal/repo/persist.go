package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/ontology"
)

// A repository is persisted as a directory holding two files:
//
//	graph.json      — the immutable graph value (UC-S23 snapshot codec)
//	namespace.json  — the mutable namespace bindings (UC-S22 FileStore)
//
// The two components have different persistence rhythms. The graph is an
// immutable value that changes only when an operation returns a new State, so
// it is written explicitly by SaveState. The namespace is the sole mutable
// component; LoadState backs it with a namespace.FileStore, which flushes
// every bind to disk on its own — so namespace changes are durable the moment
// they happen, without a SaveState call.
const (
	graphFileName     = "graph.json"
	namespaceFileName = "namespace.json"
)

// LoadState opens the repository directory dir and returns its State. The
// graph is read from graph.json (an absent file yields an empty graph); the
// namespace is backed by a durable namespace.FileStore over namespace.json,
// so subsequent binds persist automatically.
//
// The schema validates the decoded graph (UC-S23 validates on load), so a
// corrupt or hand-edited graph file is rejected here rather than surfacing as
// an ill-formed graph later.
func LoadState(dir string, schema ontology.Schema) (State, error) {
	var g graph.Graph
	data, err := os.ReadFile(filepath.Join(dir, graphFileName))
	switch {
	case err == nil:
		g, err = graph.Unmarshal(schema, data)
		if err != nil {
			return nil, fmt.Errorf("repo: load graph: %w", err)
		}
	case os.IsNotExist(err):
		g = graph.NewGraph(schema)
	default:
		return nil, fmt.Errorf("repo: load graph: %w", err)
	}

	ns, err := namespace.NewFileStore(filepath.Join(dir, namespaceFileName))
	if err != nil {
		return nil, fmt.Errorf("repo: load namespace: %w", err)
	}
	return NewState(g, ns), nil
}

// SaveState writes the state's graph to dir/graph.json with an atomic
// write-then-rename, creating dir if needed. It does not write the namespace:
// a State returned by LoadState is backed by a self-persisting
// namespace.FileStore, so its bindings are already durable. (If a caller
// constructs a State with a non-FileStore namespace, SaveState still persists
// the graph, but namespace durability is then the caller's responsibility.)
//
// Call SaveState after any operation that produces a new graph value (Ingest,
// Revise, Merge-with-witness, ...) to make the on-disk repository match the
// in-memory State.
func SaveState(dir string, state State) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := graph.Marshal(state.Graph())
	if err != nil {
		return fmt.Errorf("repo: save graph: %w", err)
	}
	tmp := filepath.Join(dir, graphFileName+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, graphFileName))
}
