package progress

import (
	"sync"
	"testing"
)

func TestCounterConcurrent(t *testing.T) {
	c := &Counter{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.FileScanned() }()
	}
	wg.Wait()
	if c.Count() != 100 {
		t.Errorf("count = %d, want 100", c.Count())
	}
}

func TestSetAndReport(t *testing.T) {
	defer Set(nil) // restore no-op
	c := &Counter{}
	Set(c)
	Report()
	Report()
	if c.Count() != 2 {
		t.Errorf("count = %d, want 2", c.Count())
	}
	// nil resets to no-op (must not panic).
	Set(nil)
	Report()
	if c.Count() != 2 {
		t.Errorf("count changed after reset: %d", c.Count())
	}
}
