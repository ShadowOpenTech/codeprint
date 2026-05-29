package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/ShadowOpenTech/codeprint/internal/emit"
	"github.com/ShadowOpenTech/codeprint/internal/progress"
	"github.com/ShadowOpenTech/codeprint/pkg/codeprint"
)

// version is overridden at release time via -ldflags.
var version = "0.0.0-dev"

// sysexits-aligned exit codes (NFR-6) plus signal convention.
const (
	exitOK       = 0
	exitUsage    = 64
	exitNoInput  = 66
	exitSoftware = 70
	exitIOErr    = 74
)

type options struct {
	format      string
	prettyJSON  bool
	out         string
	noFiles     bool
	noHidden    bool
	ignoreFile  string
	maxFileSize int64
	concurrency int
	strictIO    bool
	verbose     bool
	debug       bool
	noColor     bool
}

// execute builds and runs the CLI, returning a process exit code.
func execute(ctx context.Context) int {
	var o options
	code := exitOK

	root := &cobra.Command{
		Use:           "codeprint [path]",
		Short:         "Produce a stable JSON fingerprint of a codebase",
		Args:          cobra.MaximumNArgs(1),
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			code = runScan(ctx, path, o)
			return nil
		},
	}

	f := root.Flags()
	f.StringVar(&o.format, "format", "", "output: json|pretty|summary (default: auto by TTY)")
	f.BoolVar(&o.prettyJSON, "pretty-json", false, "indent JSON output")
	f.StringVar(&o.out, "out", "", "write JSON to file instead of stdout")
	f.BoolVar(&o.noFiles, "no-files", false, "omit per-file records (rollups only)")
	f.BoolVar(&o.noHidden, "no-hidden", false, "ignore dotfiles")
	f.StringVar(&o.ignoreFile, "ignore-file", "", "extra ignore file")
	f.Int64Var(&o.maxFileSize, "max-file-size", 0, "max file size in bytes (0 = 10MB default)")
	f.IntVar(&o.concurrency, "concurrency", 0, "worker count (0 = NumCPU)")
	f.BoolVar(&o.strictIO, "strict-io", false, "exit 74 if any file could not be read")
	f.BoolVar(&o.verbose, "verbose", false, "progress + summary on stderr")
	f.BoolVar(&o.debug, "debug", false, "per-file diagnostics on stderr")
	f.BoolVar(&o.noColor, "no-color", false, "disable color")

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "codeprint:", err)
		return exitUsage
	}
	return code
}

func runScan(ctx context.Context, path string, o options) int {
	format, err := resolveFormat(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "codeprint:", err)
		return exitUsage
	}
	useColor := wantColor(o)

	// Progress spinner: stderr TTY only, suppressed in CI/pipes (NFR-7 / api-contracts).
	var sp *spinner
	if isatty.IsTerminal(os.Stderr.Fd()) && !o.noColor && os.Getenv("NO_COLOR") == "" {
		sp = startSpinner()
		progress.Set(sp)
	}

	opts := buildOptions(o)
	start := time.Now()
	fp, scanErr := codeprint.Scan(ctx, path, opts...)
	dur := time.Since(start)

	if sp != nil {
		sp.stop()
		progress.Set(nil)
	}

	if scanErr != nil {
		if errors.Is(scanErr, context.Canceled) || errors.Is(scanErr, context.DeadlineExceeded) {
			fmt.Fprintln(os.Stderr, "codeprint: cancelled")
			return 130 // 128 + SIGINT
		}
		if errors.Is(scanErr, codeprint.ErrUnreadableRoot) {
			fmt.Fprintln(os.Stderr, "codeprint:", scanErr)
			return exitNoInput
		}
		fmt.Fprintln(os.Stderr, "codeprint:", scanErr)
		return exitSoftware
	}

	if err := write(fp, path, dur, format, useColor, o); err != nil {
		fmt.Fprintln(os.Stderr, "codeprint:", err)
		return exitIOErr
	}

	if o.verbose {
		fmt.Fprintf(os.Stderr, "scanned %d files in %s\n", fp.Totals.Files, dur.Round(time.Millisecond))
	}
	if o.debug {
		for _, f := range fp.Files {
			fmt.Fprintf(os.Stderr, "  %s\t%s\t%s\n", emit.Sanitize(f.Path), f.Language, f.Kind)
		}
	}
	if o.strictIO && len(fp.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "codeprint: %d file(s) unreadable (--strict-io)\n", len(fp.Errors))
		return exitIOErr
	}
	return exitOK
}

func write(fp *codeprint.Fingerprint, path string, dur time.Duration, format string, useColor bool, o options) error {
	out := codeprint.NewOutput(fp, "", version, time.Now().UTC().Format(time.RFC3339))

	// --out always writes JSON to the file, regardless of format/TTY.
	if o.out != "" {
		f, err := os.Create(o.out)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		return emitJSON(f, out, o.prettyJSON)
	}

	switch format {
	case "pretty":
		return emit.Pretty(os.Stdout, out, path, dur, useColor)
	case "summary":
		return emit.Summary(os.Stdout, out)
	default: // json
		return emitJSON(os.Stdout, out, o.prettyJSON)
	}
}

func emitJSON(w io.Writer, out codeprint.Output, indent bool) error {
	if indent {
		return emit.JSONIndent(w, out)
	}
	return emit.JSON(w, out)
}

func buildOptions(o options) []codeprint.Option {
	var opts []codeprint.Option
	if o.concurrency > 0 {
		opts = append(opts, codeprint.WithConcurrency(o.concurrency))
	}
	if o.maxFileSize > 0 {
		opts = append(opts, codeprint.WithMaxFileSize(o.maxFileSize))
	}
	if o.noHidden {
		opts = append(opts, codeprint.WithIncludeHidden(false))
	}
	if o.ignoreFile != "" {
		opts = append(opts, codeprint.WithIgnoreFile(o.ignoreFile))
	}
	if o.noFiles {
		opts = append(opts, codeprint.WithoutFileRecords())
	}
	return opts
}

// resolveFormat applies the explicit --format or auto-detects by stdout TTY.
func resolveFormat(o options) (string, error) {
	switch o.format {
	case "json", "pretty", "summary":
		return o.format, nil
	case "":
		if isatty.IsTerminal(os.Stdout.Fd()) {
			return "pretty", nil
		}
		return "json", nil
	default:
		return "", fmt.Errorf("invalid --format %q (want json|pretty|summary)", o.format)
	}
}

// wantColor reports whether ANSI color should be used.
func wantColor(o options) bool {
	if o.noColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd())
}
