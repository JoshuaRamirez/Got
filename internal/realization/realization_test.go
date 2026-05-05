package realization_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/realization"
)

func TestTargetType(t *testing.T) {
	var tgt realization.Target = "html-bundle-v1"
	if string(tgt) != "html-bundle-v1" {
		t.Fatal("Target string conversion broken")
	}
}

func TestFidelityContractStruct(t *testing.T) {
	fc := realization.FidelityContract{Name: "lossless"}
	if fc.Name != "lossless" {
		t.Fatal("FidelityContract.Name round-trip failed")
	}
}

func TestErrTargetUnsupportedSentinel(t *testing.T) {
	if !errors.Is(realization.ErrTargetUnsupported, realization.ErrTargetUnsupported) {
		t.Fatal("sentinel must match itself")
	}
}
