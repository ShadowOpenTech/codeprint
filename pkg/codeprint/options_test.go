package codeprint

import (
	"runtime"
	"testing"
)

func TestNewConfigDefaults(t *testing.T) {
	c := newConfig()
	if c.concurrency != runtime.NumCPU() {
		t.Errorf("concurrency = %d, want %d", c.concurrency, runtime.NumCPU())
	}
	if !c.includeHidden {
		t.Error("includeHidden should default true")
	}
	if c.maxFileSize != defaultMaxFileSize {
		t.Errorf("maxFileSize = %d, want %d", c.maxFileSize, defaultMaxFileSize)
	}
	if !c.emitFiles {
		t.Error("emitFiles should default true")
	}
	if c.ignoreFile != "" {
		t.Errorf("ignoreFile = %q, want empty", c.ignoreFile)
	}
}

func TestOptions(t *testing.T) {
	c := newConfig()
	for _, opt := range []Option{
		WithConcurrency(4),
		WithIgnoreFile(".myignore"),
		WithIncludeHidden(false),
		WithMaxFileSize(2048),
		WithoutFileRecords(),
	} {
		opt(c)
	}
	if c.concurrency != 4 {
		t.Errorf("concurrency = %d, want 4", c.concurrency)
	}
	if c.ignoreFile != ".myignore" {
		t.Errorf("ignoreFile = %q", c.ignoreFile)
	}
	if c.includeHidden {
		t.Error("includeHidden should be false")
	}
	if c.maxFileSize != 2048 {
		t.Errorf("maxFileSize = %d, want 2048", c.maxFileSize)
	}
	if c.emitFiles {
		t.Error("emitFiles should be false")
	}
}

func TestOptionsRejectInvalid(t *testing.T) {
	c := newConfig()
	WithConcurrency(0)(c)
	WithConcurrency(-3)(c)
	if c.concurrency != runtime.NumCPU() {
		t.Errorf("invalid concurrency should be ignored, got %d", c.concurrency)
	}
	WithMaxFileSize(0)(c)
	WithMaxFileSize(-1)(c)
	if c.maxFileSize != defaultMaxFileSize {
		t.Errorf("invalid maxFileSize should be ignored, got %d", c.maxFileSize)
	}
}
