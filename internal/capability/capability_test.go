package capability_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/capability"
)

func TestErrNoEmergenceSentinel(t *testing.T) {
	wrapped := errors.Join(capability.ErrNoEmergence, errors.New("detail"))
	if !errors.Is(wrapped, capability.ErrNoEmergence) {
		t.Fatal("wrapped error must match the sentinel via errors.Is")
	}
}

func TestWitnessStruct(t *testing.T) {
	w := capability.Witness{Name: "merge-fidelity"}
	if w.Name != "merge-fidelity" {
		t.Fatalf("Witness.Name = %q, want merge-fidelity", w.Name)
	}
}
