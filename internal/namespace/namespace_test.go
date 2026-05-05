package namespace_test

import (
	"context"
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
	ctx := context.Background()
	s := namespace.NewStore()
	v := vid("target")

	if err := s.BindRef(ctx, "main", v); err != nil {
		t.Fatal(err)
	}
	got, ok := s.ResolveRef(ctx, "main")
	if !ok {
		t.Fatal("ResolveRef returned false after BindRef")
	}
	if got != v {
		t.Fatalf("ResolveRef = %v, want %v", got, v)
	}
}

// Axiom: resolveAlias(bindAlias(N, a, v), a) = some(v).
func TestBindResolveAlias(t *testing.T) {
	ctx := context.Background()
	s := namespace.NewStore()
	v := vid("release-1")

	if err := s.BindAlias(ctx, "v1.0", v); err != nil {
		t.Fatal(err)
	}
	got, ok := s.ResolveAlias(ctx, "v1.0")
	if !ok {
		t.Fatal("ResolveAlias returned false after BindAlias")
	}
	if got != v {
		t.Fatalf("ResolveAlias = %v, want %v", got, v)
	}
}

// Unbound names return false.
func TestResolveUnbound(t *testing.T) {
	ctx := context.Background()
	s := namespace.NewStore()
	if _, ok := s.ResolveRef(ctx, "nonexistent"); ok {
		t.Fatal("unbound ref should return false")
	}
	if _, ok := s.ResolveAlias(ctx, "nonexistent"); ok {
		t.Fatal("unbound alias should return false")
	}
	if _, ok := s.ResolveProjection(ctx, "nonexistent"); ok {
		t.Fatal("unbound projection should return false")
	}
}

// Rebinding overwrites the previous value.
func TestRebind(t *testing.T) {
	ctx := context.Background()
	s := namespace.NewStore()
	v1 := vid("first")
	v2 := vid("second")

	s.BindRef(ctx, "main", v1)
	s.BindRef(ctx, "main", v2)

	got, ok := s.ResolveRef(ctx, "main")
	if !ok || got != v2 {
		t.Fatalf("rebind should overwrite: got %v, want %v", got, v2)
	}
}

// Projection handle binding.
func TestBindResolveProjection(t *testing.T) {
	ctx := context.Background()
	s := namespace.NewStore()
	v := vid("proj-target")

	if err := s.BindProjection(ctx, "default-view", v); err != nil {
		t.Fatal(err)
	}
	got, ok := s.ResolveProjection(ctx, "default-view")
	if !ok {
		t.Fatal("ResolveProjection returned false after BindProjection")
	}
	if got != v {
		t.Fatalf("ResolveProjection = %v, want %v", got, v)
	}
}
