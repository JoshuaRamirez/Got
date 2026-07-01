package namespace_test

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/joshuaramirez/got/internal/identity"
	"github.com/joshuaramirez/got/internal/namespace"
)

func fvid(s string) identity.VertexID {
	return identity.VertexID(sha256.Sum256([]byte(s)))
}

// FileStore satisfies the Store interface.
func TestFileStoreImplementsStore(t *testing.T) {
	var _ namespace.Store = mustFileStore(t, filepath.Join(t.TempDir(), "ns.json"))
}

func mustFileStore(t *testing.T, path string) *namespace.FileStore {
	t.Helper()
	s, err := namespace.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	return s
}

func TestFileStoreBindResolveAllKinds(t *testing.T) {
	ctx := context.Background()
	s := mustFileStore(t, filepath.Join(t.TempDir(), "ns.json"))

	ref, alias, proj := fvid("r"), fvid("a"), fvid("p")
	if err := s.BindRef(ctx, "main", ref); err != nil {
		t.Fatal(err)
	}
	if err := s.BindAlias(ctx, "v1", alias); err != nil {
		t.Fatal(err)
	}
	if err := s.BindProjection(ctx, "view", proj); err != nil {
		t.Fatal(err)
	}

	if got, ok := s.ResolveRef(ctx, "main"); !ok || got != ref {
		t.Fatalf("ref: got %v ok=%v", got, ok)
	}
	if got, ok := s.ResolveAlias(ctx, "v1"); !ok || got != alias {
		t.Fatalf("alias: got %v ok=%v", got, ok)
	}
	if got, ok := s.ResolveProjection(ctx, "view"); !ok || got != proj {
		t.Fatalf("proj: got %v ok=%v", got, ok)
	}
}

func TestFileStoreUnbound(t *testing.T) {
	ctx := context.Background()
	s := mustFileStore(t, filepath.Join(t.TempDir(), "ns.json"))
	if _, ok := s.ResolveRef(ctx, "nope"); ok {
		t.Fatal("unbound ref should resolve to ok=false")
	}
}

func TestFileStoreRebindOverwrites(t *testing.T) {
	ctx := context.Background()
	s := mustFileStore(t, filepath.Join(t.TempDir(), "ns.json"))
	if err := s.BindRef(ctx, "main", fvid("first")); err != nil {
		t.Fatal(err)
	}
	if err := s.BindRef(ctx, "main", fvid("second")); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ResolveRef(ctx, "main"); got != fvid("second") {
		t.Fatal("rebind should overwrite")
	}
}

// Durability: a second FileStore opened on the same path sees the bindings a
// first one wrote — surviving a simulated process restart.
func TestFileStoreDurableAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ns.json")

	s1 := mustFileStore(t, path)
	if err := s1.BindRef(ctx, "main", fvid("target")); err != nil {
		t.Fatal(err)
	}
	if err := s1.BindAlias(ctx, "v1", fvid("rel")); err != nil {
		t.Fatal(err)
	}

	s2 := mustFileStore(t, path) // "restart"
	if got, ok := s2.ResolveRef(ctx, "main"); !ok || got != fvid("target") {
		t.Fatalf("ref did not survive reopen: got %v ok=%v", got, ok)
	}
	if got, ok := s2.ResolveAlias(ctx, "v1"); !ok || got != fvid("rel") {
		t.Fatalf("alias did not survive reopen: got %v ok=%v", got, ok)
	}
}

// Corrupt file surfaces an error at open rather than silently starting empty.
func TestFileStoreCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ns.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := namespace.NewFileStore(path); err == nil {
		t.Fatal("expected error opening corrupt store")
	}
}

// Concurrent writers are safe (run under -race).
func TestFileStoreConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	s := mustFileStore(t, filepath.Join(t.TempDir(), "ns.json"))

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := namespace.RefName(string(rune('a' + n)))
			if err := s.BindRef(ctx, name, fvid(string(rune('a'+n)))); err != nil {
				t.Errorf("BindRef: %v", err)
			}
			_, _ = s.ResolveRef(ctx, name)
		}(i)
	}
	wg.Wait()

	for i := 0; i < 16; i++ {
		name := namespace.RefName(string(rune('a' + i)))
		if got, ok := s.ResolveRef(ctx, name); !ok || got != fvid(string(rune('a'+i))) {
			t.Fatalf("binding %q lost under concurrency", name)
		}
	}
}
