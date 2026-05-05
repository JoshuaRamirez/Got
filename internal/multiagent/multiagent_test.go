package multiagent_test

import (
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/multiagent"
)

func TestResponsibilityStruct(t *testing.T) {
	a := identity.VertexID(sha256.Sum256([]byte("agent")))
	r := multiagent.Responsibility{Path: []identity.VertexID{a}}
	if len(r.Path) != 1 || r.Path[0] != a {
		t.Fatal("Responsibility.Path round-trip failed")
	}
}

func TestErrNoAuthorshipSentinel(t *testing.T) {
	if !errors.Is(multiagent.ErrNoAuthorship, multiagent.ErrNoAuthorship) {
		t.Fatal("sentinel must match itself")
	}
}
