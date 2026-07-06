package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/history"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
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

// --- commit history ---

func loadHistory() (*history.Log, error) { return repo.LoadHistory(stateDir()) }

func saveHistory(log *history.Log) error { return repo.SaveHistory(stateDir(), log) }

// commitRefName is the namespace ref that tracks a branch's current commit,
// kept separate from the branch's vertex tip.
func commitRefName(branch string) namespace.RefName { return namespace.RefName("commit:" + branch) }

// A CommitID and a VertexID are both 32-byte content hashes; the namespace
// stores VertexIDs, so branch commit pointers are round-tripped through these.
func vidFromCommit(c history.CommitID) identity.VertexID { return identity.VertexID(c) }
func commitFromVID(v identity.VertexID) history.CommitID { return history.CommitID(v) }

// --- HEAD (current branch) ---

// headPath is the git-style file naming the current branch.
func headPath() string { return filepath.Join(stateDir(), "HEAD") }

// currentBranch returns the branch HEAD points at, defaulting to "main".
func currentBranch() string {
	b, err := os.ReadFile(headPath())
	if err != nil {
		return "main"
	}
	name := strings.TrimSpace(string(b))
	if name == "" {
		return "main"
	}
	return name
}

// setHEAD points HEAD at the given branch name.
func setHEAD(branch string) error {
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(headPath(), []byte(branch+"\n"), 0o644)
}

// --- stash (a stack of saved working states) ---

type stashEntry struct {
	Branch   string         `json:"branch"`
	Snapshot graph.Snapshot `json:"snapshot"`
}

func stashPath() string { return filepath.Join(stateDir(), "stash.json") }

func loadStashes() ([]stashEntry, error) {
	b, err := os.ReadFile(stashPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s []stashEntry
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("corrupt stash file: %w", err)
	}
	return s, nil
}

func saveStashes(s []stashEntry) error {
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := stashPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, stashPath())
}

// --- tags (lightweight commit names) ---

func tagsPath() string { return filepath.Join(stateDir(), "tags.json") }

func loadTags() (map[string]string, error) {
	b, err := os.ReadFile(tagsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("corrupt tags file: %w", err)
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

func saveTags(m map[string]string) error {
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := tagsPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, tagsPath())
}

// resolveCommit resolves a commit-ish — a branch name (its tip), a tag, or a
// commit-id hex prefix — to a CommitID.
func resolveCommit(state repo.State, log *history.Log, ref string) (history.CommitID, bool) {
	if id, ok := state.Namespace().ResolveRef(context.Background(), commitRefName(ref)); ok {
		if _, ok := log.Get(commitFromVID(id)); ok {
			return commitFromVID(id), true
		}
	}
	if tags, err := loadTags(); err == nil {
		if h, ok := tags[ref]; ok {
			if cid, err := decodeCommitHex(h); err == nil {
				if _, ok := log.Get(cid); ok {
					return cid, true
				}
			}
		}
	}
	if len(ref) >= 4 {
		for _, c := range log.Commits() {
			if strings.HasPrefix(hex.EncodeToString(c.ID[:]), strings.ToLower(ref)) {
				return c.ID, true
			}
		}
	}
	return history.CommitID{}, false
}

func decodeCommitHex(s string) (history.CommitID, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return history.CommitID{}, fmt.Errorf("bad commit hex %q", s)
	}
	var id history.CommitID
	copy(id[:], b)
	return id, nil
}

// applySnapDelta applies a graph.Delta to a snapshot, returning the resulting
// snapshot. Used to revert a commit: the reverse delta (c -> parent) applied to
// the current state undoes the commit. Edges whose endpoints do not survive are
// dropped.
func applySnapDelta(cur graph.Snapshot, d graph.Delta) graph.Snapshot {
	verts := make(map[string]graph.VertexSnapshot, len(cur.Vertices))
	for _, v := range cur.Vertices {
		verts[v.ID] = v
	}
	for _, v := range d.RemovedVertices {
		delete(verts, v.ID)
	}
	for _, c := range d.ChangedVertices {
		verts[c.New.ID] = c.New
	}
	for _, v := range d.AddedVertices {
		verts[v.ID] = v
	}

	edges := make(map[string]graph.EdgeSnapshot, len(cur.Edges))
	for _, e := range cur.Edges {
		edges[e.ID] = e
	}
	for _, e := range d.RemovedEdges {
		delete(edges, e.ID)
	}
	for _, c := range d.ChangedEdges {
		edges[c.New.ID] = c.New
	}
	for _, e := range d.AddedEdges {
		edges[e.ID] = e
	}

	var out graph.Snapshot
	for _, v := range verts {
		out.Vertices = append(out.Vertices, v)
	}
	for _, e := range edges {
		if _, ok := verts[e.From]; !ok {
			continue
		}
		if _, ok := verts[e.To]; !ok {
			continue
		}
		out.Edges = append(out.Edges, e)
	}
	out.Hyperedges = cur.Hyperedges
	return out
}

// headSnapshot returns the committed snapshot at a branch's tip, and whether
// the branch has any commit.
func headSnapshot(state repo.State, log *history.Log, branch string) (graph.Snapshot, bool) {
	id, ok := state.Namespace().ResolveRef(context.Background(), commitRefName(branch))
	if !ok {
		return graph.Snapshot{}, false
	}
	c, ok := log.Get(commitFromVID(id))
	if !ok {
		return graph.Snapshot{}, false
	}
	return c.Snapshot, true
}
