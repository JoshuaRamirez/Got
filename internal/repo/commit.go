package repo

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/history"
	"github.com/joshuaramirez/got/internal/identity"
)

// history.json holds the operation-first commit DAG (UC-S26) for a repository
// directory, alongside graph.json (current graph) and namespace.json (refs,
// including branch commit pointers).
const historyFileName = "history.json"

// LoadHistory reads the commit log from dir/history.json. An absent file
// yields an empty log.
func LoadHistory(dir string) (*history.Log, error) {
	data, err := os.ReadFile(filepath.Join(dir, historyFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return history.NewLog(), nil
		}
		return nil, fmt.Errorf("repo: load history: %w", err)
	}
	return history.Unmarshal(data)
}

// SaveHistory writes the commit log to dir/history.json with an atomic
// write-then-rename.
func SaveHistory(dir string, log *history.Log) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := history.Marshal(log)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, historyFileName+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, historyFileName))
}

// Commit records the current graph state as a new commit in log and returns
// it. The commit's operation delta (Consumed/Produced) is computed as the
// vertex-set difference against the first parent's state — so history captures
// what changed, not only the resulting snapshot. The graph itself is not
// modified; a commit is a recording.
func (s *DefaultService) Commit(ctx context.Context, state State, log *history.Log, message, actor string, parents []history.CommitID) (history.Commit, error) {
	if err := ctx.Err(); err != nil {
		return history.Commit{}, err
	}
	snap := graph.EncodeSnapshot(state.Graph())

	cur := make(map[identity.VertexID]bool)
	for _, id := range state.Graph().VertexIDs() {
		cur[id] = true
	}

	var consumed, produced []identity.VertexID
	if len(parents) > 0 {
		parent, ok := log.Get(parents[0])
		if !ok {
			return history.Commit{}, fmt.Errorf("%w: %s", history.ErrUnknownCommit, hex.EncodeToString(parents[0][:]))
		}
		prev := snapshotVertexIDSet(parent.Snapshot)
		for id := range prev {
			if !cur[id] {
				consumed = append(consumed, id)
			}
		}
		for id := range cur {
			if !prev[id] {
				produced = append(produced, id)
			}
		}
	} else {
		for id := range cur {
			produced = append(produced, id)
		}
	}

	c := history.NewCommit(parents, message, actor, consumed, produced, snap)
	if err := log.Add(c); err != nil {
		return history.Commit{}, err
	}
	return c, nil
}

// snapshotVertexIDSet decodes the vertex IDs recorded in a snapshot.
func snapshotVertexIDSet(snap graph.Snapshot) map[identity.VertexID]bool {
	out := make(map[identity.VertexID]bool, len(snap.Vertices))
	for _, v := range snap.Vertices {
		b, err := hex.DecodeString(v.ID)
		if err != nil || len(b) != 32 {
			continue
		}
		var id identity.VertexID
		copy(id[:], b)
		out[id] = true
	}
	return out
}
