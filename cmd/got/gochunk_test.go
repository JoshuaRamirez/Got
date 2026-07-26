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

// Byte-fidelity is the sharp risk for import decomposition: Split→Join must be
// verbatim for every import shape, or the tiling silently drops/dupes bytes.
func TestGoChunkerImportRoundTrip(t *testing.T) {
	ch := newGoChunker()
	cases := []string{
		"package p\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc F() {}\n",
		"package p\n\nimport (\n\t\"fmt\"\n\n\t\"github.com/x/y\" // c\n)\n",
		"package p\n\nimport (\n\tf \"fmt\"\n\t_ \"embed\"\n\t. \"errors\"\n)\n",
		"package p\n\nimport ()\n\nfunc F() {}\n",
		"package p\n\nimport \"fmt\"\n\nfunc F() { _ = fmt.Sprint }\n",
		"package p\n\n// leading doc\nimport (\n\t// grouped\n\t\"a/b\"\n)\n",
	}
	for _, in := range cases {
		if got := ch.Join(ch.Split(in)); got != in {
			t.Fatalf("import round-trip mismatch:\n in=%q\nout=%q", in, got)
		}
	}
}

func TestGoValidityGateImports(t *testing.T) {
	// import name collides with a package-scope var (the fixed P1).
	if goValidityOK("package p\n\nimport \"fmt\"\n\nvar fmt = 1\n") {
		t.Fatal("import name vs var must collide")
	}
	// duplicate import.
	if goValidityOK("package p\n\nimport (\n\t\"fmt\"\n\t\"fmt\"\n)\n") {
		t.Fatal("duplicate import must fail the gate")
	}
	// an alias avoids the collision.
	if !goValidityOK("package p\n\nimport f \"fmt\"\n\nvar fmt = 1\n") {
		t.Fatal("aliased import should not collide with var fmt")
	}
	// blank and dot imports bind no checkable name.
	if !goValidityOK("package p\n\nimport _ \"embed\"\n\nvar embed = 1\n") {
		t.Fatal("blank import binds no name; should not collide")
	}
}
