package namespace_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/joshuaramirez/got/internal/namespace"
)

// newRemote wires an HTTPStore client to an in-process server backed by a
// fresh memStore, returning the client and the backing store.
func newRemote(t *testing.T) (*namespace.HTTPStore, namespace.Store) {
	t.Helper()
	backing := namespace.NewStore()
	srv := httptest.NewServer(namespace.NewHTTPHandler(backing))
	t.Cleanup(srv.Close)
	return namespace.NewHTTPStore(srv.URL, srv.Client()), backing
}

func TestHTTPStoreImplementsStore(t *testing.T) {
	var _ namespace.Store = namespace.NewHTTPStore("http://example", nil)
}

// Bind/resolve round-trips over HTTP for all three name kinds, and the binding
// actually lands in the backing store.
func TestHTTPStoreRoundTripAllKinds(t *testing.T) {
	ctx := context.Background()
	client, backing := newRemote(t)

	ref, alias, proj := fvid("r"), fvid("a"), fvid("p")
	if err := client.BindRef(ctx, "main", ref); err != nil {
		t.Fatal(err)
	}
	if err := client.BindAlias(ctx, "v1", alias); err != nil {
		t.Fatal(err)
	}
	if err := client.BindProjection(ctx, "view", proj); err != nil {
		t.Fatal(err)
	}

	if got, ok := client.ResolveRef(ctx, "main"); !ok || got != ref {
		t.Fatalf("ref over HTTP: got %v ok=%v", got, ok)
	}
	if got, ok := client.ResolveAlias(ctx, "v1"); !ok || got != alias {
		t.Fatalf("alias over HTTP: got %v ok=%v", got, ok)
	}
	if got, ok := client.ResolveProjection(ctx, "view"); !ok || got != proj {
		t.Fatalf("projection over HTTP: got %v ok=%v", got, ok)
	}

	// The bind landed server-side.
	if got, ok := backing.ResolveRef(ctx, "main"); !ok || got != ref {
		t.Fatal("client bind did not reach the backing store")
	}
}

// Resolving an unbound name over HTTP returns not-found (not an error).
func TestHTTPStoreUnbound(t *testing.T) {
	ctx := context.Background()
	client, _ := newRemote(t)
	if _, ok := client.ResolveRef(ctx, "nope"); ok {
		t.Fatal("unbound ref should resolve to ok=false over HTTP")
	}
}

// A name bound with special characters round-trips (URL-escaped).
func TestHTTPStoreEscaping(t *testing.T) {
	ctx := context.Background()
	client, _ := newRemote(t)
	name := namespace.RefName("feature/x y&z")
	if err := client.BindRef(ctx, name, fvid("special")); err != nil {
		t.Fatal(err)
	}
	if got, ok := client.ResolveRef(ctx, name); !ok || got != fvid("special") {
		t.Fatalf("special-char name did not round-trip: got %v ok=%v", got, ok)
	}
}

// A cancelled context makes a bind fail (the ctx is threaded onto the request).
func TestHTTPStoreContextCancelled(t *testing.T) {
	client, _ := newRemote(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.BindRef(ctx, "main", fvid("x")); err == nil {
		t.Fatal("expected bind to fail with a cancelled context")
	}
	// Resolve has no error return; a cancelled ctx surfaces as not-found.
	if _, ok := client.ResolveRef(ctx, "main"); ok {
		t.Fatal("cancelled resolve should surface as not-found")
	}
}

func TestHTTPStoreDelete(t *testing.T) {
	ctx := context.Background()
	client, backing := newRemote(t)
	if err := client.BindRef(ctx, "main", fvid("t")); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteRef(ctx, "main"); err != nil {
		t.Fatal(err)
	}
	if _, ok := client.ResolveRef(ctx, "main"); ok {
		t.Fatal("deleted ref should not resolve over HTTP")
	}
	if _, ok := backing.ResolveRef(ctx, "main"); ok {
		t.Fatal("delete should reach the backing store")
	}
}
