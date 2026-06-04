package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// writeResults writes the collected resources to stdout as a JSON array.
// Choosing the destination (file or pipe) is left to shell redirection.
// It warns on stderr only when writing directly to a terminal, to guard
// against accidental runs.
func writeResults(results []resource) error {
	if fi, err := os.Stdout.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr,
			"warning: writing JSON to terminal; redirect with '> dump.json'")
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(results)
}
