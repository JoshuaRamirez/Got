package repo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/ontology"
)

// First-class branches. Unlike git — where a branch is a bare mutable pointer
// (a ref file holding one hash, with no identity, metadata, or history) — a
// branch here is a BranchSelector vertex in the append-only graph: it has a
// content-addressed identity, carries metadata, and records its fork parent as
// a forks_from edge. It therefore persists, is queryable, and has traceable
// ancestry. The one mutable part — the branch's tip, which advances as work
// lands — stays a namespace binding (name -> tip vertex); mutability
// concentrates there rather than disappearing.

// ErrBranchExists indicates CreateBranch was asked to create a branch that
// already exists.
var ErrBranchExists = errors.New("repo: branch already exists")

// ErrUnknownBranch indicates a named branch is not present in the graph.
var ErrUnknownBranch = errors.New("repo: unknown branch")

// Reserved BranchSelector attribute keys.
const (
	branchNameAttr   = "branch.name"
	branchParentAttr = "branch.parent"
)

// Branch is a first-class branch: the identity and metadata of a
// BranchSelector vertex. The tip (what the branch currently points at) is
// resolved separately through the namespace by the branch name.
type Branch struct {
	ID     identity.VertexID
	Name   string
	Parent string            // parent branch name; "" for a root branch
	Attrs  map[string]string // metadata (e.g. description)
}

// BranchVID derives the content-addressed id of a branch vertex from its name.
func BranchVID(name string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte("branch:" + name)))
}

func branchForkEdgeID(child, parent string) identity.EdgeID {
	return identity.EdgeID(sha256.Sum256([]byte("forks_from:" + child + "->" + parent)))
}

// CreateBranch records a first-class branch. It ingests a BranchSelector
// vertex (carrying the branch name, optional parent, and any metadata), links
// it to its parent branch with a forks_from edge when a parent is given, and
// binds the namespace tip name -> tip (when tip is non-zero). Returns the new
// State and the created Branch.
//
// Errors: ErrBranchExists if the branch already exists; ErrUnknownBranch if a
// named parent branch is not present; graph.ErrVertexNotFound if a non-zero
// tip is not in the graph.
func (s *DefaultService) CreateBranch(ctx context.Context, state State, name, parent string, tip identity.VertexID, meta map[string]string) (State, Branch, error) {
	if err := ctx.Err(); err != nil {
		return nil, Branch{}, err
	}
	if name == "" {
		return nil, Branch{}, fmt.Errorf("%w: empty branch name", ErrIngestRejected)
	}
	if _, ok := state.Graph().Vertex(BranchVID(name)); ok {
		return nil, Branch{}, fmt.Errorf("%w: %q", ErrBranchExists, name)
	}
	if parent != "" {
		if _, ok := state.Graph().Vertex(BranchVID(parent)); !ok {
			return nil, Branch{}, fmt.Errorf("%w: parent %q", ErrUnknownBranch, parent)
		}
	}
	if tip != (identity.VertexID{}) {
		if _, ok := state.Graph().Vertex(tip); !ok {
			return nil, Branch{}, fmt.Errorf("%w: tip %v", graph.ErrVertexNotFound, tip)
		}
	}

	attrs := graph.AttrMap{branchNameAttr: name}
	if parent != "" {
		attrs[branchParentAttr] = parent
	}
	for k, v := range meta {
		attrs[k] = v
	}

	newState, err := s.Ingest(ctx, state, VertexPayload{
		Vertices: []graph.Vertex{{ID: BranchVID(name), Type: ontology.BranchSelector, Attrs: attrs}},
	})
	if err != nil {
		return nil, Branch{}, err
	}

	if parent != "" {
		newState, err = s.Ingest(ctx, newState, EdgePayload{
			Edges: []graph.Edge{{
				ID:   branchForkEdgeID(name, parent),
				Type: ontology.ForksFrom,
				From: BranchVID(name),
				To:   BranchVID(parent),
			}},
		})
		if err != nil {
			return nil, Branch{}, err
		}
	}

	if tip != (identity.VertexID{}) {
		if err := newState.Namespace().BindRef(ctx, namespace.RefName(name), tip); err != nil {
			return nil, Branch{}, err
		}
	}

	return newState, branchFromVertex(mustVertex(newState.Graph(), BranchVID(name))), nil
}

// Branches returns all first-class branches recorded in the graph.
func (s *DefaultService) Branches(ctx context.Context, state State) ([]Branch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []Branch
	for _, v := range state.Graph().Vertices() {
		if v.Type == ontology.BranchSelector {
			out = append(out, branchFromVertex(v))
		}
	}
	return out, nil
}

// BranchLineage returns the fork ancestry of a branch, starting with the
// branch itself and walking forks_from edges up to a root branch. A cycle
// (which the admissibility rules do not create in normal use) is broken
// defensively.
func (s *DefaultService) BranchLineage(ctx context.Context, state State, name string) ([]Branch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g := state.Graph()
	if _, ok := g.Vertex(BranchVID(name)); !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownBranch, name)
	}
	var lineage []Branch
	seen := make(map[string]bool)
	cur := name
	for cur != "" && !seen[cur] {
		seen[cur] = true
		v, ok := g.Vertex(BranchVID(cur))
		if !ok {
			break
		}
		b := branchFromVertex(v)
		lineage = append(lineage, b)
		cur = b.Parent
	}
	return lineage, nil
}

func branchFromVertex(v graph.Vertex) Branch {
	b := Branch{ID: v.ID}
	attrs := make(map[string]string)
	for k, val := range v.Attrs {
		s, ok := val.(string)
		if !ok {
			continue
		}
		switch k {
		case branchNameAttr:
			b.Name = s
		case branchParentAttr:
			b.Parent = s
		default:
			attrs[k] = s
		}
	}
	if len(attrs) > 0 {
		b.Attrs = attrs
	}
	return b
}

func mustVertex(g graph.Graph, id identity.VertexID) graph.Vertex {
	v, _ := g.Vertex(id)
	return v
}
