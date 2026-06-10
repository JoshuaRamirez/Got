// Command got is a thin command-line shell over the repository library. It
// persists a single JSON state file under $GOT_DIR (default .got) and drives
// the library engines for each subcommand. See
// docs/requirements/use-cases/user/UC-U19-operate-from-cli.md.
//
// Usage:
//
//	got init
//	got add-vertex <name> --type <VertexType> [--attr k=v ...]
//	got add-edge <name> --type <EdgeType> --from <v> --to <v>
//	got bind <ref> <vertex>
//	got resolve <ref>
//	got list vertices|edges
//	got trace <from> <to>
//	got cone <name>
package main

import (
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
