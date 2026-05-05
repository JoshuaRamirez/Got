package governance_test

import (
	"errors"
	"testing"

	"github.com/joshuaramirez/got/internal/governance"
)

// Decision values are ordered Unsat < Unknown < Sat so a callers can rely on
// the three-valued comparison.
func TestDecisionOrdering(t *testing.T) {
	if !(governance.Unsat < governance.Unknown && governance.Unknown < governance.Sat) {
		t.Fatalf("decision ordering broken: Unsat=%d Unknown=%d Sat=%d",
			governance.Unsat, governance.Unknown, governance.Sat)
	}
}

func TestObligationStruct(t *testing.T) {
	o := governance.Obligation{Code: "P-001", Detail: "missing reviewer"}
	if o.Code != "P-001" || o.Detail != "missing reviewer" {
		t.Fatal("Obligation field round-trip failed")
	}
}

func TestErrPolicyViolationSentinel(t *testing.T) {
	if !errors.Is(governance.ErrPolicyViolation, governance.ErrPolicyViolation) {
		t.Fatal("sentinel must match itself")
	}
}
