package temporal_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/temporal"
)

func TestIntervalStruct(t *testing.T) {
	iv := temporal.Interval{From: 100, To: 200}
	if iv.From != 100 || iv.To != 200 {
		t.Fatal("Interval round-trip failed")
	}
}

func TestErrUnknownVertexSentinel(t *testing.T) {
	if !errors.Is(temporal.ErrUnknownVertex, temporal.ErrUnknownVertex) {
		t.Fatal("sentinel must match itself")
	}
}
