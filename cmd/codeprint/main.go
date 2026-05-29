// Command codeprint is the reference producer for codeprint fingerprints.
//
// M1 provides a minimal entry point: scan a path and emit canonical JSON. The
// full CLI (cobra, pretty/summary output, progress, exit-code polish) lands in M4.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ShadowOpenTech/codeprint/internal/emit"
	"github.com/ShadowOpenTech/codeprint/pkg/codeprint"
)

// version is overridden at release time via -ldflags.
var version = "0.0.0-dev"

// sysexits-aligned codes (subset used in M1; full mapping in M4).
const (
	exitUsage   = 64
	exitNoInput = 66
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		format      = flag.String("format", "json", "output format: json")
		concurrency = flag.Int("concurrency", 0, "worker count (0 = NumCPU)")
		out         = flag.String("out", "", "write JSON to file instead of stdout")
		showVer     = flag.Bool("version", false, "print version and exit")
	)

	// M1 uses the stdlib flag package, which stops at the first positional.
	// Partition args so flags work in any position; flags must use --name=value
	// form. The full cobra CLI (M4) supports all flag styles. The first bare
	// argument is the path.
	root := "."
	var flagArgs []string
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
		} else if root == "." {
			root = a
		}
	}
	if err := flag.CommandLine.Parse(flagArgs); err != nil {
		return exitUsage
	}

	if *showVer {
		fmt.Println("codeprint", version)
		return 0
	}
	if *format != "json" {
		fmt.Fprintln(os.Stderr, "codeprint: only --format=json is implemented in M1 (pretty/summary land in M4)")
		return exitUsage
	}

	var opts []codeprint.Option
	if *concurrency > 0 {
		opts = append(opts, codeprint.WithConcurrency(*concurrency))
	}

	fp, err := codeprint.Scan(context.Background(), root, opts...)
	if err != nil {
		if errors.Is(err, codeprint.ErrUnreadableRoot) {
			fmt.Fprintln(os.Stderr, "codeprint:", err)
			return exitNoInput
		}
		fmt.Fprintln(os.Stderr, "codeprint:", err)
		return exitNoInput
	}

	output := codeprint.NewOutput(fp, "", version, time.Now().UTC().Format(time.RFC3339))

	w := os.Stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "codeprint:", err)
			return exitNoInput
		}
		defer func() { _ = f.Close() }()
		w = f
	}
	if err := emit.JSON(w, output); err != nil {
		fmt.Fprintln(os.Stderr, "codeprint:", err)
		return exitNoInput
	}
	return 0
}
