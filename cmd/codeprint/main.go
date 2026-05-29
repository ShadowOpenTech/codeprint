// Command codeprint is the reference producer for codeprint fingerprints.
//
// The full CLI (flags, output modes, progress) is built in milestone M4; this
// entry point is intentionally minimal until the Scan pipeline lands (M1).
package main

import (
	"fmt"
	"os"
)

// version is overridden at release time via -ldflags.
var version = "0.0.0-dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("codeprint", version)
		return
	}
	fmt.Fprintln(os.Stderr, "codeprint", version, "- scan pipeline lands in M1; CLI in M4")
}
