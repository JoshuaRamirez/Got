package graph

import (
	"reflect"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/ontology"
)

// This file provides a small, composable query language over the graph. A
// Query selects a set of vertices; Graph.Query induces the subgraph on that
// set (every edge and hyperedge whose endpoints all lie in the set). Queries
// compose with And/Or, so callers build predicates without the engine needing
// to know every combination.

// ByType selects vertices of a given ontology type.
type ByType struct {
	Type ontology.VertexType
}

// QueryKind identifies ByType.
func (ByType) QueryKind() string { return "by-type" }

// ByAttr selects vertices whose attribute Key equals Value (deep equality).
// A vertex without Key does not match.
type ByAttr struct {
	Key   string
	Value any
}

// QueryKind identifies ByAttr.
func (ByAttr) QueryKind() string { return "by-attr" }

// And selects vertices that match every sub-query (set intersection). An
// empty And matches no vertices.
type And struct {
	Queries []Query
}

// QueryKind identifies And.
func (And) QueryKind() string { return "and" }

// Or selects vertices that match any sub-query (set union). An empty Or
// matches no vertices.
type Or struct {
	Queries []Query
}

// QueryKind identifies Or.
func (Or) QueryKind() string { return "or" }

// matchVertices resolves a Query to the set of matching vertex IDs in g.
// Unknown query types yield ErrQueryUnsupported.
func matchVertices(g Graph, q Query) (map[identity.VertexID]bool, error) {
	switch qq := q.(type) {
	case ByType:
		out := make(map[identity.VertexID]bool)
		for _, v := range g.Vertices() {
			if v.Type == qq.Type {
				out[v.ID] = true
			}
		}
		return out, nil

	case ByAttr:
		out := make(map[identity.VertexID]bool)
		for _, v := range g.Vertices() {
			if val, ok := v.Attrs[qq.Key]; ok && reflect.DeepEqual(val, qq.Value) {
				out[v.ID] = true
			}
		}
		return out, nil

	case And:
		if len(qq.Queries) == 0 {
			return map[identity.VertexID]bool{}, nil
		}
		var acc map[identity.VertexID]bool
		for i, sub := range qq.Queries {
			m, err := matchVertices(g, sub)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				acc = m
				continue
			}
			for id := range acc {
				if !m[id] {
					delete(acc, id)
				}
			}
		}
		return acc, nil

	case Or:
		out := make(map[identity.VertexID]bool)
		for _, sub := range qq.Queries {
			m, err := matchVertices(g, sub)
			if err != nil {
				return nil, err
			}
			for id := range m {
				out[id] = true
			}
		}
		return out, nil

	default:
		return nil, ErrQueryUnsupported
	}
}
