package release_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/release"
)

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
