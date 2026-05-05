package replay_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/replay"
)

func TestOutcomeStruct(t *testing.T) {
	o := replay.Outcome{Deterministic: true}
	if !o.Deterministic {
		t.Fatal("Outcome.Deterministic round-trip failed")
	}
	var zero replay.Outcome
	if zero.Deterministic {
		t.Fatal("zero-value Outcome must not be deterministic")
	}
}

func TestErrNonDeterministicSentinel(t *testing.T) {
	if !errors.Is(replay.ErrNonDeterministic, replay.ErrNonDeterministic) {
		t.Fatal("sentinel must match itself")
	}
}
