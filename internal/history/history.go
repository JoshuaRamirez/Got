// Package history implements an operation-first commit DAG.
//
// Where git records a snapshot (a commit points at a tree) and reconstructs
// intent heuristically, a Commit here records the operation that produced it —
// the Consumed/Produced vertex delta — alongside the resulting graph snapshot,
// the parents, the actor, and a message. History is therefore non-lossy about
// how each state was reached, and forms a DAG (a merge commit has two parents)
// you can walk like `git log`.
//
// Identity is content-addressed: a CommitID is the SHA-256 of the parents,
// message, actor, and the resulting state's element IDs — so equal commits
// share an ID, and any tampering changes it.
//
// Imports: internal/identity, internal/graph.
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
)

// CommitID is the content-addressed identifier of a Commit.
type CommitID [32]byte

// ErrUnknownCommit indicates a CommitID is not present in the Log.
var ErrUnknownCommit = errors.New("history: unknown commit")

// ErrUnknownParent indicates a commit references a parent not in the Log.
var ErrUnknownParent = errors.New("history: unknown parent")

// Commit is a single node in the history DAG.
type Commit struct {
	ID      CommitID
	Parents []CommitID
	Message string
	Actor   string
	// Operation delta (the non-lossy part): what this commit removed and added.
	Consumed []identity.VertexID
	Produced []identity.VertexID
	// Resulting graph state (UC-S23 snapshot).
	Snapshot graph.Snapshot
}

// NewCommit builds a Commit and computes its content-addressed ID from the
// parents, message, actor, and resulting-state element IDs.
func NewCommit(parents []CommitID, message, actor string, consumed, produced []identity.VertexID, snap graph.Snapshot) Commit {
	c := Commit{
		Parents:  append([]CommitID(nil), parents...),
		Message:  message,
		Actor:    actor,
		Consumed: append([]identity.VertexID(nil), consumed...),
		Produced: append([]identity.VertexID(nil), produced...),
		Snapshot: snap,
	}
	c.ID = computeID(parents, message, actor, snap)
	return c
}

// computeID hashes the parents, message, actor, and a per-element content
// digest of the resulting snapshot. Consumed/Produced are annotation, not
// identity — two commits reaching the same state from the same parents are the
// same commit (mirrors git: id = hash(tree, parents, author, message)).
//
// The tree digest folds in each element's full content (id, type, attributes,
// and structural fields), not just its id. This matters because a vertex id is
// content-addressed on its *name* (e.g. a file path), so two trees that differ
// only in a vertex's attributes — a file edited in place at the same path —
// share vertex ids. Hashing bare ids would collide those trees to one commit
// id, and Log.Add would silently drop the second. Digesting content keeps such
// commits distinct.
func computeID(parents []CommitID, message, actor string, snap graph.Snapshot) CommitID {
	h := sha256.New()
	h.Write([]byte("history.commit\x00"))

	ps := make([]string, len(parents))
	for i, p := range parents {
		ps[i] = hex.EncodeToString(p[:])
	}
	sort.Strings(ps)
	for _, p := range ps {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	h.Write([]byte("msg\x00"))
	h.Write([]byte(message))
	h.Write([]byte("\x00actor\x00"))
	h.Write([]byte(actor))

	writeSorted := func(tag string, digests []string) {
		h.Write([]byte(tag))
		s := append([]string(nil), digests...)
		sort.Strings(s)
		for _, d := range s {
			h.Write([]byte(d))
			h.Write([]byte{0})
		}
	}
	// A canonical JSON encoding of each element is its content digest: Go sorts
	// map keys, so AttrMap serializes deterministically, and struct fields have
	// a fixed order.
	digest := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v) // unreachable for snapshot types
		}
		return string(b)
	}
	vds := make([]string, len(snap.Vertices))
	for i, v := range snap.Vertices {
		vds[i] = digest(v)
	}
	eds := make([]string, len(snap.Edges))
	for i, e := range snap.Edges {
		eds[i] = digest(e)
	}
	hds := make([]string, len(snap.Hyperedges))
	for i, he := range snap.Hyperedges {
		hds[i] = digest(he)
	}
	writeSorted("\x00v\x00", vds)
	writeSorted("\x00e\x00", eds)
	writeSorted("\x00h\x00", hds)

	var id CommitID
	copy(id[:], h.Sum(nil))
	return id
}

// Log is an append-only DAG of commits.
type Log struct {
	commits map[CommitID]Commit
	order   []CommitID
}

// NewLog returns an empty Log.
func NewLog() *Log {
	return &Log{commits: make(map[CommitID]Commit)}
}

// Add appends a commit. Its parents must already be present. Re-adding an
// identical commit (same ID) is a no-op.
func (l *Log) Add(c Commit) error {
	for _, p := range c.Parents {
		if _, ok := l.commits[p]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownParent, hex.EncodeToString(p[:]))
		}
	}
	if _, ok := l.commits[c.ID]; ok {
		return nil
	}
	l.commits[c.ID] = c
	l.order = append(l.order, c.ID)
	return nil
}

// Get returns the commit with the given ID.
func (l *Log) Get(id CommitID) (Commit, bool) {
	c, ok := l.commits[id]
	return c, ok
}

// Commits returns every commit in insertion order.
func (l *Log) Commits() []Commit {
	out := make([]Commit, 0, len(l.order))
	for _, id := range l.order {
		out = append(out, l.commits[id])
	}
	return out
}

// Ancestors returns the commit id and all its ancestors (transitively via
// parents), in breadth-first discovery order — the commit first, then its
// history, like `git log`.
func (l *Log) Ancestors(id CommitID) ([]Commit, error) {
	if _, ok := l.commits[id]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommit, hex.EncodeToString(id[:]))
	}
	var out []Commit
	seen := make(map[CommitID]bool)
	queue := []CommitID{id}
	seen[id] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		c := l.commits[cur]
		out = append(out, c)
		for _, p := range c.Parents {
			if !seen[p] {
				seen[p] = true
				queue = append(queue, p)
			}
		}
	}
	return out, nil
}

// MergeBase returns the nearest common ancestor of commits a and b — the
// closest ancestor of b that is also an ancestor of a (both inclusive). It is
// the three-way merge base. Returns ok == false when the two commits share no
// history.
func (l *Log) MergeBase(a, b CommitID) (CommitID, bool) {
	ancestorsA := make(map[CommitID]bool)
	queue := []CommitID{a}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if ancestorsA[c] {
			continue
		}
		ancestorsA[c] = true
		if cm, ok := l.commits[c]; ok {
			queue = append(queue, cm.Parents...)
		}
	}

	seen := make(map[CommitID]bool)
	queue = []CommitID{b}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if seen[c] {
			continue
		}
		seen[c] = true
		if ancestorsA[c] {
			return c, true
		}
		if cm, ok := l.commits[c]; ok {
			queue = append(queue, cm.Parents...)
		}
	}
	return CommitID{}, false
}

// --- JSON persistence ---

type commitJSON struct {
	ID       string         `json:"id"`
	Parents  []string       `json:"parents,omitempty"`
	Message  string         `json:"message"`
	Actor    string         `json:"actor,omitempty"`
	Consumed []string       `json:"consumed,omitempty"`
	Produced []string       `json:"produced,omitempty"`
	Snapshot graph.Snapshot `json:"snapshot"`
}

// Marshal serializes the log to JSON in insertion order.
func Marshal(l *Log) ([]byte, error) {
	arr := make([]commitJSON, 0, len(l.order))
	for _, c := range l.Commits() {
		cj := commitJSON{
			ID:       hex.EncodeToString(c.ID[:]),
			Message:  c.Message,
			Actor:    c.Actor,
			Snapshot: c.Snapshot,
		}
		for _, p := range c.Parents {
			cj.Parents = append(cj.Parents, hex.EncodeToString(p[:]))
		}
		for _, v := range c.Consumed {
			cj.Consumed = append(cj.Consumed, hex.EncodeToString(v[:]))
		}
		for _, v := range c.Produced {
			cj.Produced = append(cj.Produced, hex.EncodeToString(v[:]))
		}
		arr = append(arr, cj)
	}
	return json.MarshalIndent(arr, "", "  ")
}

// Unmarshal deserializes a log from JSON, validating parent references.
func Unmarshal(data []byte) (*Log, error) {
	var arr []commitJSON
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("history: corrupt log: %w", err)
	}
	l := NewLog()
	for _, cj := range arr {
		id, err := decodeCommitID(cj.ID)
		if err != nil {
			return nil, err
		}
		c := Commit{ID: id, Message: cj.Message, Actor: cj.Actor, Snapshot: cj.Snapshot}
		for _, p := range cj.Parents {
			pid, err := decodeCommitID(p)
			if err != nil {
				return nil, err
			}
			c.Parents = append(c.Parents, pid)
		}
		for _, v := range cj.Consumed {
			vid, err := decodeVertexID(v)
			if err != nil {
				return nil, err
			}
			c.Consumed = append(c.Consumed, vid)
		}
		for _, v := range cj.Produced {
			vid, err := decodeVertexID(v)
			if err != nil {
				return nil, err
			}
			c.Produced = append(c.Produced, vid)
		}
		if err := l.Add(c); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func decodeCommitID(s string) (CommitID, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return CommitID{}, fmt.Errorf("history: bad commit id %q", s)
	}
	var id CommitID
	copy(id[:], b)
	return id, nil
}

func decodeVertexID(s string) (identity.VertexID, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return identity.VertexID{}, fmt.Errorf("history: bad vertex id %q", s)
	}
	var id identity.VertexID
	copy(id[:], b)
	return id, nil
}
