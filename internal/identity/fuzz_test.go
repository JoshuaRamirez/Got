package identity_test

import (
	"testing"

	"github.com/joshuaramirez/got/internal/identity"
)

// rawCanonical wraps a byte slice as a Canonical without any encoding
// logic. Useful for tests and fuzz inputs.
type rawCanonical []byte

func (r rawCanonical) CanonicalBytes() ([]byte, error) { return []byte(r), nil }

// FuzzVertexIDDeterminism asserts that the same canonical bytes always
// produce the same VertexID. If this ever fails the identity contract
// is broken and every higher-layer guarantee collapses.
func FuzzVertexIDDeterminism(f *testing.F) {
	for _, seed := range [][]byte{
		{},
		{0x00},
		{0xff, 0xff, 0xff, 0xff},
		[]byte("hello"),
		[]byte("the quick brown fox jumps over the lazy dog"),
	} {
		f.Add(seed)
	}

	factory := identity.NewFactory(identity.NewSHA256Hasher())

	f.Fuzz(func(t *testing.T, data []byte) {
		c := rawCanonical(data)
		id1, err := factory.VertexID(c)
		if err != nil {
			t.Fatalf("VertexID returned error for valid Canonical: %v", err)
		}
		id2, err := factory.VertexID(c)
		if err != nil {
			t.Fatal(err)
		}
		if id1 != id2 {
			t.Fatalf("non-deterministic VertexID: %x vs %x", id1, id2)
		}
	})
}

// FuzzIDDistinctnessAcrossKinds asserts that VertexID, EdgeID, and
// HyperedgeID for the same canonical bytes produce the same 32 bytes
// (they are type aliases over the same Hash) — i.e. the kind is a
// type-level discriminator, not derived from extra bytes.
func FuzzIDDistinctnessAcrossKinds(f *testing.F) {
	f.Add([]byte("seed"))
	f.Add([]byte{})
	factory := identity.NewFactory(identity.NewSHA256Hasher())

	f.Fuzz(func(t *testing.T, data []byte) {
		c := rawCanonical(data)
		vid, _ := factory.VertexID(c)
		eid, _ := factory.EdgeID(c)
		hid, _ := factory.HyperedgeID(c)
		if [32]byte(vid) != [32]byte(eid) || [32]byte(eid) != [32]byte(hid) {
			t.Fatalf("ID kinds disagree for same bytes: v=%x e=%x h=%x", vid, eid, hid)
		}
	})
}
