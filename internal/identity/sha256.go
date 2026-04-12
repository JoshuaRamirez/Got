package identity

import "crypto/sha256"

// SHA256Hasher computes identity hashes using SHA-256.
type SHA256Hasher struct{}

// NewSHA256Hasher returns a Hasher backed by SHA-256.
// The [32]byte Hash type is sized to match this algorithm.
func NewSHA256Hasher() Hasher {
	return SHA256Hasher{}
}

func (SHA256Hasher) Sum(data []byte) Hash {
	return sha256.Sum256(data)
}

// DefaultFactory composes a Hasher with Canonical.CanonicalBytes to derive
// typed identifiers.
type DefaultFactory struct {
	hasher Hasher
}

// NewFactory creates a Factory that derives IDs using the given Hasher.
func NewFactory(h Hasher) Factory {
	return &DefaultFactory{hasher: h}
}

func (f *DefaultFactory) VertexID(c Canonical) (VertexID, error) {
	b, err := c.CanonicalBytes()
	if err != nil {
		return VertexID{}, err
	}
	return VertexID(f.hasher.Sum(b)), nil
}

func (f *DefaultFactory) EdgeID(c Canonical) (EdgeID, error) {
	b, err := c.CanonicalBytes()
	if err != nil {
		return EdgeID{}, err
	}
	return EdgeID(f.hasher.Sum(b)), nil
}

func (f *DefaultFactory) HyperedgeID(c Canonical) (HyperedgeID, error) {
	b, err := c.CanonicalBytes()
	if err != nil {
		return HyperedgeID{}, err
	}
	return HyperedgeID(f.hasher.Sum(b)), nil
}
