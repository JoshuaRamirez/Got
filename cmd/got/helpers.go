package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/joshuaramirez/got/internal/namespace"
)

// multiFlag collects a repeatable string flag (e.g. --attr k=v --attr a=b).
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// parse turns the collected "k=v" entries into a map, erroring on any entry
// that is not of the form key=value with a non-empty key.
func (m multiFlag) parse() (map[string]string, error) {
	if len(m) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for _, kv := range m {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("bad attribute %q: want key=value", kv)
		}
		out[k] = v
	}
	return out, nil
}

// splitName takes the leading positional <name> from args and returns the
// remaining flag arguments. ok is false when args is empty or the first
// argument looks like a flag (the name is mandatory and must come first).
func splitName(args []string) (name string, rest []string, ok bool) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", nil, false
	}
	return args[0], args[1:], true
}

func refName(s string) namespace.RefName { return namespace.RefName(s) }

// shortID renders the first 6 bytes of an ID as hex for compact display.
func shortID(b []byte) string {
	if len(b) > 6 {
		b = b[:6]
	}
	return hex.EncodeToString(b)
}

// joinArrow renders a vertex-name path as "a -> b -> c".
func joinArrow(names []string) string {
	return strings.Join(names, " -> ")
}
