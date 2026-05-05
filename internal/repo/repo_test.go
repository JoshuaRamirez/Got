package repo_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/repo"
)

// stubPayload exists solely to verify that Payload's PayloadKind discriminator
// is sufficient for callers to implement custom payload types.
type stubPayload struct{ kind string }

func (s stubPayload) PayloadKind() string { return s.kind }

func TestPayloadInterface(t *testing.T) {
	var p repo.Payload = stubPayload{kind: "vertex-batch"}
	if p.PayloadKind() != "vertex-batch" {
		t.Fatalf("PayloadKind = %q, want vertex-batch", p.PayloadKind())
	}
}

func TestErrIngestRejectedSentinel(t *testing.T) {
	if !errors.Is(repo.ErrIngestRejected, repo.ErrIngestRejected) {
		t.Fatal("sentinel must match itself")
	}
}
