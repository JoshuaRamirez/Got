package projection_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/projection"
)

func TestErrInvalidSelectorSentinel(t *testing.T) {
	if !errors.Is(projection.ErrInvalidSelector, projection.ErrInvalidSelector) {
		t.Fatal("sentinel must match itself")
	}
}
