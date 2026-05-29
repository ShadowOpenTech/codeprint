// Package progress is the internal progress sink. The public WithProgress
// option was deferred to v2; this sink lets the CLI show a spinner without a
// public API. The library default is a no-op, so Scan stays silent and pure for
// in-process consumers.
//
// The sink is a package-level value, which means concurrent Scan calls in one
// process share it. That is acceptable: only the CLI (single scan per process)
// sets it; library consumers never do.
package progress

import "sync/atomic"

// Reporter receives a notification as each file is processed. Implementations
// must be safe for concurrent use (workers call Report from many goroutines).
type Reporter interface {
	FileScanned()
}

type noop struct{}

func (noop) FileScanned() {}

var active Reporter = noop{}

// Set installs the active reporter (CLI use). Pass nil to reset to no-op.
func Set(r Reporter) {
	if r == nil {
		active = noop{}
		return
	}
	active = r
}

// Report notifies the active reporter that one file was processed.
func Report() { active.FileScanned() }

// Counter is a minimal thread-safe Reporter that just counts files. Useful for
// tests and as a building block.
type Counter struct{ n int64 }

// FileScanned increments the count.
func (c *Counter) FileScanned() { atomic.AddInt64(&c.n, 1) }

// Count returns the number of files reported so far.
func (c *Counter) Count() int64 { return atomic.LoadInt64(&c.n) }
