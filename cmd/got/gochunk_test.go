package main

import "testing"

func TestGoChunkerRoundTrip(t *testing.T) {
	ch := newGoChunker()
	cases := []string{
		"package main\n\nfunc F() int {\n\treturn 1\n}\n",
		"package main\n\nimport \"fmt\"\n\n// Doc.\nfunc F() { fmt.Println(1) }\n\ntype T struct{ X int }\n",
		"package p\n\nvar A = 1\n\nconst B = 2\n\nfunc (t T) M() {}\n",
		"not valid go {{{", // parse error → block-chunker fallback, still lossless
	}
	for _, in := range cases {
		if got := ch.Join(ch.Split(in)); got != in {
			t.Fatalf("round-trip mismatch:\n in=%q\nout=%q", in, got)
		}
	}
}

// A declaration is keyed by its symbol, and that key is stable when the body is
// edited or reindented — so a merge aligns "same func, edited".
func TestGoChunkerSymbolKeysStable(t *testing.T) {
	ch := newGoChunker()
	before := ch.Split("package p\n\nfunc Foo() int {\n\treturn 1\n}\n")
	after := ch.Split("package p\n\nfunc Foo() int {\n\t\treturn 2 // reindented + edited\n}\n")

	keyOf := func(chunks []chunk, want string) string {
		for _, c := range chunks {
			if len(c.Key) >= len(want) && c.Key[:len(want)] == want {
				return c.Key
			}
		}
		return ""
	}
	kb := keyOf(before, "func:Foo")
	ka := keyOf(after, "func:Foo")
	if kb == "" || ka == "" {
		t.Fatalf("expected func:Foo chunk in both, got %v / %v", before, after)
	}
	if kb != ka {
		t.Fatalf("func key not stable across body edit: %q vs %q", kb, ka)
	}
}

func TestGoValidityGate(t *testing.T) {
	valid := "package p\n\nfunc A() {}\n\nfunc B() {}\n"
	if !goValidityOK(valid) {
		t.Fatal("well-formed file should pass")
	}
	dupFunc := "package p\n\nfunc A() {}\n\nfunc A() {}\n"
	if goValidityOK(dupFunc) {
		t.Fatal("duplicate func must fail the gate")
	}
	funcVarCollision := "package p\n\nfunc Size() int { return 0 }\n\nvar Size = 1\n"
	if goValidityOK(funcVarCollision) {
		t.Fatal("func/var name collision must fail the gate")
	}
	// Methods on different types may share a name without colliding.
	methods := "package p\n\ntype T struct{}\ntype U struct{}\n\nfunc (T) M() {}\nfunc (U) M() {}\n"
	if !goValidityOK(methods) {
		t.Fatal("same method name on different types should pass")
	}
	if goValidityOK("package p\n\nfunc broken( {") {
		t.Fatal("unparseable file must fail the gate")
	}
}
