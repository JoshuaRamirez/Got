package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/revision"
)

// multiFlag collects a repeatable string flag (e.g. --attr k=v --attr a=b).
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// parse turns the collected "k=v" entries into a map, erroring on any entry
// that is not of the form key=value with a non-empty key.
func (m multiFlag) parse() (map[string]string, error) {
	if len(m) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for _, kv := range m {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("bad attribute %q: want key=value", kv)
		}
		out[k] = v
	}
	return out, nil
}

// splitName takes the leading positional <name> from args and returns the
// remaining flag arguments. ok is false when args is empty or the first
// argument looks like a flag (the name is mandatory and must come first).
func splitName(args []string) (name string, rest []string, ok bool) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", nil, false
	}
	return args[0], args[1:], true
}

func refName(s string) namespace.RefName { return namespace.RefName(s) }

// shortID renders the first 6 bytes of an ID as hex for compact display.
func shortID(b []byte) string {
	if len(b) > 6 {
		b = b[:6]
	}
	return hex.EncodeToString(b)
}

// joinArrow renders a vertex-name path as "a -> b -> c".
func joinArrow(names []string) string {
	return strings.Join(names, " -> ")
}

// joinComma renders a name list as "a, b, c".
func joinComma(names []string) string {
	return strings.Join(names, ", ")
}

// splitCSV splits a comma-separated flag value into trimmed, non-empty
// fields. An empty string yields nil.
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// resolveNames maps a list of vertex names to their content-addressed IDs,
// erroring on the first name that is not a known vertex in the snapshot.
func resolveNames(snap *snapshot, names []string) ([]identity.VertexID, error) {
	ids := make([]identity.VertexID, 0, len(names))
	for _, n := range names {
		if _, ok := snap.vertexByName(n); !ok {
			return nil, fmt.Errorf("unknown vertex %q", n)
		}
		ids = append(ids, vid(n))
	}
	return ids, nil
}

// subgraph is a literal in-memory graph.Subgraph used to build DPO rewrite
// rules in the CLI. Unlike Graph.Induce it can carry produced (R-side)
// vertices and edges that do not yet exist in the host graph.
type subgraph struct {
	ids   []identity.VertexID
	verts []graph.Vertex
	edges []graph.Edge
}

func (s subgraph) VertexIDs() []identity.VertexID { return s.ids }
func (s subgraph) Vertices() []graph.Vertex       { return s.verts }
func (s subgraph) Edges() []graph.Edge            { return s.edges }
func (s subgraph) Hyperedges() []graph.Hyperedge  { return nil }

// rule is a literal DPO rewrite rule with no side conditions.
type rule struct {
	left, ctx, right graph.Subgraph
}

func (r rule) Left() graph.Subgraph                 { return r.left }
func (r rule) Context() graph.Subgraph              { return r.ctx }
func (r rule) Right() graph.Subgraph                { return r.right }
func (r rule) SideConditions() []revision.Predicate { return nil }

// match is a literal injective L → G vertex map.
type match struct {
	m map[identity.VertexID]identity.VertexID
}

func (m match) Mapping() map[identity.VertexID]identity.VertexID { return m.m }
