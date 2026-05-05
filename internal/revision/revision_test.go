package revision_test

import (
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/revision"
)

func TestTransformKindStringForm(t *testing.T) {
	var k revision.TransformKind = "merge-pushout"
	if string(k) != "merge-pushout" {
		t.Fatal("TransformKind string conversion broken")
	}
}

func TestChangeCapsuleZeroValue(t *testing.T) {
	var c revision.ChangeCapsule
	if c.Consumed != nil || c.Produced != nil {
		t.Fatal("zero-value ChangeCapsule should have nil slices")
	}
}

func TestChangeCapsuleRoundTrip(t *testing.T) {
	v := identity.VertexID(sha256.Sum256([]byte("v")))
	c := revision.ChangeCapsule{
		Consumed: []identity.VertexID{v},
		Kind:     "test",
	}
	if len(c.Consumed) != 1 || c.Consumed[0] != v || c.Kind != "test" {
		t.Fatal("ChangeCapsule round-trip failed")
	}
}

func TestSentinels(t *testing.T) {
	for _, e := range []error{revision.ErrSideConditionFailed, revision.ErrNoMatch} {
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}
