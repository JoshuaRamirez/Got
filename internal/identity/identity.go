// Package identity implements the IdentityKernel specification.
//
// It provides content-addressable identity for vertices, edges, and hyperedges.
// All identifiers are derived deterministically from canonical byte encodings,
// guaranteeing that structurally equal objects produce equal identifiers.
//
// This is the leaf of the dependency graph: it imports no other internal package.
package identity

// Hash is a fixed-size content hash. All identity in the system derives from it.
type Hash [32]byte

// VertexID uniquely identifies a vertex via the hash of its canonical encoding.
type VertexID Hash

// EdgeID uniquely identifies an edge via the hash of its canonical encoding.
type EdgeID Hash

// HyperedgeID uniquely identifies a hyperedge via the hash of its canonical encoding.
type HyperedgeID Hash

// Canonical is implemented by any value that can produce a deterministic byte
// encoding. The encoding must be injective: distinct logical values must yield
// distinct byte sequences.
type Canonical interface {
	CanonicalBytes() ([]byte, error)
}

// Hasher computes the fixed-size hash of an arbitrary byte slice.
type Hasher interface {
	Sum(data []byte) Hash
}

// Factory derives typed identifiers from Canonical values. Implementations
// compose a Hasher with Canonical.CanonicalBytes to produce each ID.
//
// Axiom: for any Canonical x, VertexID(x) == Hasher.Sum(x.CanonicalBytes()).
// Axiom: encode(x) == encode(y) => VertexID(x) == VertexID(y).
type Factory interface {
	VertexID(Canonical) (VertexID, error)
	EdgeID(Canonical) (EdgeID, error)
	HyperedgeID(Canonical) (HyperedgeID, error)
}
