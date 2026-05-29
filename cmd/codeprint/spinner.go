package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// spinner is an indeterminate progress indicator: a spinner glyph plus a
// running file counter, drawn to stderr and throttled. It implements
// progress.Reporter. Suppressed entirely unless stderr is a TTY (set up by the
// caller). The single-pass streaming scan has no total, so there is no bar.
type spinner struct {
	count int64
	done  chan struct{}
}

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

func startSpinner() *spinner {
	s := &spinner{done: make(chan struct{})}
	go s.loop()
	return s
}

// FileScanned implements progress.Reporter (called concurrently by workers).
func (s *spinner) FileScanned() { atomic.AddInt64(&s.count, 1) }

func (s *spinner) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			n := atomic.LoadInt64(&s.count)
			fmt.Fprintf(os.Stderr, "\r%c scanning… %d files", spinnerFrames[i%len(spinnerFrames)], n)
			i++
		}
	}
}

// stop halts the animation and clears the line.
func (s *spinner) stop() {
	close(s.done)
	fmt.Fprint(os.Stderr, "\r\033[K") // carriage return + clear to EOL
}
