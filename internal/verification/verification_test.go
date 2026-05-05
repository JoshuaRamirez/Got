package verification_test

import (
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/verification"
)

func TestEnvironmentBindingStruct(t *testing.T) {
	id := identity.VertexID(sha256.Sum256([]byte("env")))
	eb := verification.EnvironmentBinding{ID: id, Version: "1.2.3"}
	if eb.ID != id || eb.Version != "1.2.3" {
		t.Fatal("EnvironmentBinding round-trip failed")
	}
}

func TestSentinels(t *testing.T) {
	for _, e := range []error{verification.ErrCertificationFailed, verification.ErrEnvironmentMismatch} {
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}
