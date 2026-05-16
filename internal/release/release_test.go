package release_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/joshuaramirez/got/internal/graph"
	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
	"github.com/joshuaramirez/got/internal/projection"
	"github.com/joshuaramirez/got/internal/release"
	"github.com/joshuaramirez/got/internal/verification"
)

func vid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

func TestSentinels(t *testing.T) {
	for _, e := range []error{release.ErrPolicyGate, release.ErrUnknownVersion} {
		if e == nil {
			t.Fatal("sentinel should not be nil")
		}
		if !errors.Is(e, e) {
			t.Fatal("sentinel must match itself")
		}
	}
}

// --- helpers ---

type stubFrontier struct{ ids []identity.VertexID }

func (s stubFrontier) VertexIDs() []identity.VertexID { return s.ids }

type stubCertificate struct{ target projection.Frontier }

func (c stubCertificate) Target() projection.Frontier         { return c.target }
func (c stubCertificate) Evidence() []verification.Evaluation { return nil }
func (c stubCertificate) Policies() []string                  { return nil }

// versionFor reproduces the service's internal version-string derivation
// so the test can issue Rollback against a known version.
func versionFor(target identity.VertexID) string {
	return fmt.Sprintf("v-%x", [32]byte(target))[:10]
}

// --- behavioral tests ---

// Main path: Promote binds the alias and lets ResolveAlias return the target.
func TestPromoteHappyPath(t *testing.T) {
	ctx := context.Background()
	target := vid("rel-1")
	frontier := stubFrontier{ids: []identity.VertexID{target}}
	cert := stubCertificate{target: frontier}

	store := namespace.NewStore()
	svc := release.NewService(store)

	if err := svc.Promote(ctx, "v1.0", frontier, cert, nil); err != nil {
		t.Fatal(err)
	}
	got, ok := store.ResolveAlias(ctx, "v1.0")
	if !ok || got != target {
		t.Fatalf("ResolveAlias = (%v, %v), want (%v, true)", got, ok, target)
	}
}

// Failure: empty frontier → ErrPolicyGate.
func TestPromoteEmptyFrontier(t *testing.T) {
	ctx := context.Background()
	store := namespace.NewStore()
	svc := release.NewService(store)

	err := svc.Promote(ctx, "v1.0", stubFrontier{}, stubCertificate{target: stubFrontier{}}, nil)
	if !errors.Is(err, release.ErrPolicyGate) {
		t.Fatalf("expected ErrPolicyGate for empty frontier, got %v", err)
	}
}

// Failure: nil certificate → ErrPolicyGate.
func TestPromoteNilCertificate(t *testing.T) {
	ctx := context.Background()
	target := vid("rel-1")
	frontier := stubFrontier{ids: []identity.VertexID{target}}

	store := namespace.NewStore()
	svc := release.NewService(store)

	err := svc.Promote(ctx, "v1.0", frontier, nil, nil)
	if !errors.Is(err, release.ErrPolicyGate) {
		t.Fatalf("expected ErrPolicyGate for nil certificate, got %v", err)
	}
}

// Failure: certificate.Target() does not match frontier → ErrPolicyGate.
func TestPromoteCertificateTargetMismatch(t *testing.T) {
	ctx := context.Background()
	target := vid("rel-1")
	frontier := stubFrontier{ids: []identity.VertexID{target}}
	otherFrontier := stubFrontier{ids: []identity.VertexID{vid("other")}}
	cert := stubCertificate{target: otherFrontier}

	store := namespace.NewStore()
	svc := release.NewService(store)

	err := svc.Promote(ctx, "v1.0", frontier, cert, nil)
	if !errors.Is(err, release.ErrPolicyGate) {
		t.Fatalf("expected ErrPolicyGate for target mismatch, got %v", err)
	}
}

// Main path: Rollback rebinds to a previously-promoted version.
func TestRollbackHappyPath(t *testing.T) {
	ctx := context.Background()
	target := vid("rel-1")
	frontier := stubFrontier{ids: []identity.VertexID{target}}
	cert := stubCertificate{target: frontier}

	store := namespace.NewStore()
	svc := release.NewService(store)

	if err := svc.Promote(ctx, "v1.0", frontier, cert, nil); err != nil {
		t.Fatal(err)
	}

	newTarget := vid("rel-2")
	newFrontier := stubFrontier{ids: []identity.VertexID{newTarget}}
	if err := svc.Promote(ctx, "v1.0", newFrontier, stubCertificate{target: newFrontier}, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.ResolveAlias(ctx, "v1.0"); got != newTarget {
		t.Fatalf("alias bound to %v, want %v", got, newTarget)
	}

	// Roll back to the original.
	if err := svc.Rollback(ctx, "v1.0", versionFor(target)); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.ResolveAlias(ctx, "v1.0"); got != target {
		t.Fatalf("after rollback alias = %v, want %v", got, target)
	}
}

// Failure: unknown version → ErrUnknownVersion.
func TestRollbackUnknownVersion(t *testing.T) {
	ctx := context.Background()
	store := namespace.NewStore()
	svc := release.NewService(store)

	err := svc.Rollback(ctx, "v1.0", "ghost-version")
	if !errors.Is(err, release.ErrUnknownVersion) {
		t.Fatalf("expected ErrUnknownVersion, got %v", err)
	}
}

// Failure: ctx cancelled.
func TestPromoteContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	target := vid("rel-1")
	frontier := stubFrontier{ids: []identity.VertexID{target}}
	cert := stubCertificate{target: frontier}

	store := namespace.NewStore()
	svc := release.NewService(store)

	err := svc.Promote(ctx, "v1.0", frontier, cert, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// Smoke: makeGraph kept for any future cross-package fixture use.
var _ = graph.NewGraph
