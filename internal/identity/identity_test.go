package identity_test

import (
	"testing"

	"github.com/joshuaramirez/got/internal/identity"
)

// testCanonical is a Canonical value backed by a raw byte slice.
type testCanonical struct{ data []byte }

func (c testCanonical) CanonicalBytes() ([]byte, error) { return c.data, nil }

func factory() identity.Factory {
	return identity.NewFactory(identity.NewSHA256Hasher())
}

// Axiom: encode(x) = encode(y) => vid(x) = vid(y).
func TestEqualEncodingYieldsEqualVID(t *testing.T) {
	f := factory()
	a := testCanonical{data: []byte("hello")}
	b := testCanonical{data: []byte("hello")}

	va, err := f.VertexID(a)
	if err != nil {
		t.Fatal(err)
	}
	vb, err := f.VertexID(b)
	if err != nil {
		t.Fatal(err)
	}
	if va != vb {
		t.Fatalf("equal encodings produced different VIDs: %x != %x", va, vb)
	}
}

// Distinct encodings should (with overwhelming probability) produce distinct IDs.
func TestDistinctEncodingYieldsDistinctVID(t *testing.T) {
	f := factory()
	a := testCanonical{data: []byte("alpha")}
	b := testCanonical{data: []byte("beta")}

	va, _ := f.VertexID(a)
	vb, _ := f.VertexID(b)
	if va == vb {
		t.Fatal("distinct encodings produced identical VIDs")
	}
}

// vid, eid, hid all derive from the same hash of the same encoding.
func TestAllIDTypesConsistent(t *testing.T) {
	f := factory()
	c := testCanonical{data: []byte("consistent")}

	vid, _ := f.VertexID(c)
	eid, _ := f.EdgeID(c)
	hid, _ := f.HyperedgeID(c)

	// All should have the same underlying bytes.
	if identity.Hash(vid) != identity.Hash(eid) {
		t.Fatal("VertexID and EdgeID differ for same canonical input")
	}
	if identity.Hash(vid) != identity.Hash(hid) {
		t.Fatal("VertexID and HyperedgeID differ for same canonical input")
	}
}
