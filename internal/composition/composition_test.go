package composition_test

import (
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/composition"
	"github.com/joshuaramirez/got/internal/identity"
)

func TestConflictKindValues(t *testing.T) {
	cases := map[composition.ConflictKind]string{
		composition.Textual:    "textual",
		composition.Structural: "structural",
		composition.Schema:     "schema",
		composition.Policy:     "policy",
		composition.Trust:      "trust",
		composition.Capability: "capability",
		composition.Evaluation: "evaluation",
		composition.Temporal:   "temporal",
	}
	for k, want := range cases {
		if string(k) != want {
			t.Errorf("ConflictKind %q has wrong string form %q", want, string(k))
		}
	}
}

func TestMergeWitnessStruct(t *testing.T) {
	id := identity.VertexID(sha256.Sum256([]byte("merge")))
	w := composition.MergeWitness{ID: id}
	if w.ID != id {
		t.Fatal("MergeWitness.ID round-trip failed")
	}
}

func TestMergeResultZeroValue(t *testing.T) {
	var mr composition.MergeResult
	if mr.Conflicts != nil {
		t.Fatal("zero-value MergeResult.Conflicts should be nil")
	}
}

func TestSentinels(t *testing.T) {
	for _, e := range []error{composition.ErrConflictUnresolvable, composition.ErrNoPushout} {
		if e == nil {
			t.Fatal("sentinel should not be nil")
		}
		if !errors.Is(e, e) {
			t.Fatal("sentinel must be equal to itself under errors.Is")
		}
	}
}
