package codeprint

import "runtime"

// Option configures a Scan. Options follow the WithX naming convention; new
// options can be added without changing the Scan signature.
type Option func(*config)

// config holds resolved scan settings. Unexported: only WithX constructors and
// Scan touch it.
type config struct {
	concurrency   int
	ignoreFile    string
	includeHidden bool
	maxFileSize   int64
	emitFiles     bool
}

// defaultMaxFileSize is the default per-file ceiling (10 MB). Larger files are
// counted but not scanned for LOC and flagged skipped_large.
const defaultMaxFileSize int64 = 10 << 20

// newConfig returns the default configuration before options are applied.
func newConfig() *config {
	return &config{
		concurrency:   runtime.NumCPU(),
		includeHidden: true, // hidden files are NOT ignored by default
		maxFileSize:   defaultMaxFileSize,
		emitFiles:     true, // per-file records included by default
	}
}

// WithConcurrency sets the worker-pool size. Values < 1 are ignored (default
// runtime.NumCPU() is kept).
func WithConcurrency(n int) Option {
	return func(c *config) {
		if n >= 1 {
			c.concurrency = n
		}
	}
}

// WithIgnoreFile adds an extra ignore file, layered on top of .gitignore and
// .codeprintignore.
func WithIgnoreFile(path string) Option {
	return func(c *config) { c.ignoreFile = path }
}

// WithIncludeHidden controls whether dotfiles are scanned. Default true.
func WithIncludeHidden(b bool) Option {
	return func(c *config) { c.includeHidden = b }
}

// WithMaxFileSize overrides the per-file size ceiling in bytes. Values < 1 are
// ignored.
func WithMaxFileSize(bytes int64) Option {
	return func(c *config) {
		if bytes >= 1 {
			c.maxFileSize = bytes
		}
	}
}

// WithoutFileRecords drops per-file records, emitting language/build-system
// rollups only. Useful for size-sensitive consumers at fleet scale.
func WithoutFileRecords() Option {
	return func(c *config) { c.emitFiles = false }
}
