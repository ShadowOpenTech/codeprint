// Command codeprint is the reference producer for codeprint fingerprints.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Cancel the scan on SIGINT/SIGTERM; the CLI then exits 128+signum.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := execute(ctx)
	stop()
	os.Exit(code)
}
