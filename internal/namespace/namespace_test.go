package namespace_test

import (
	"crypto/sha256"
	"testing"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

// Axiom: resolveRef(bindRef(N, r, v), r) = some(v).
func TestBindResolveRef(t *testing.T) {
	s := namespace.NewStore()
	v := vid("target")

	if err := s.BindRef("main", v); err != nil {
		t.Fatal(err)
	}
	got, ok := s.ResolveRef("main")
	if !ok {
		t.Fatal("ResolveRef returned false after BindRef")
	}
	if got != v {
		t.Fatalf("ResolveRef = %v, want %v", got, v)
	}
}

// Axiom: resolveAlias(bindAlias(N, a, v), a) = some(v).
func TestBindResolveAlias(t *testing.T) {
	s := namespace.NewStore()
	v := vid("release-1")

	if err := s.BindAlias("v1.0", v); err != nil {
		t.Fatal(err)
	}
	got, ok := s.ResolveAlias("v1.0")
	if !ok {
		t.Fatal("ResolveAlias returned false after BindAlias")
	}
	if got != v {
		t.Fatalf("ResolveAlias = %v, want %v", got, v)
	}
}

// Unbound names return false.
func TestResolveUnbound(t *testing.T) {
	s := namespace.NewStore()
	if _, ok := s.ResolveRef("nonexistent"); ok {
		t.Fatal("unbound ref should return false")
	}
	if _, ok := s.ResolveAlias("nonexistent"); ok {
		t.Fatal("unbound alias should return false")
	}
	if _, ok := s.ResolveProjection("nonexistent"); ok {
		t.Fatal("unbound projection should return false")
	}
}

// Rebinding overwrites the previous value.
func TestRebind(t *testing.T) {
	s := namespace.NewStore()
	v1 := vid("first")
	v2 := vid("second")

	s.BindRef("main", v1)
	s.BindRef("main", v2)

	got, ok := s.ResolveRef("main")
	if !ok || got != v2 {
		t.Fatalf("rebind should overwrite: got %v, want %v", got, v2)
	}
}

// Projection handle binding.
func TestBindResolveProjection(t *testing.T) {
	s := namespace.NewStore()
	v := vid("proj-target")

	if err := s.BindProjection("default-view", v); err != nil {
		t.Fatal(err)
	}
	got, ok := s.ResolveProjection("default-view")
	if !ok {
		t.Fatal("ResolveProjection returned false after BindProjection")
	}
	if got != v {
		t.Fatalf("ResolveProjection = %v, want %v", got, v)
	}
}
