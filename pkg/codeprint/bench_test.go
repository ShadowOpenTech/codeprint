package codeprint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// buildSyntheticTree writes n small source files across a few languages and
// returns the root. Used for a throughput benchmark when the big-repo perf
// corpus (submodule, deferred) is unavailable.
func buildSyntheticTree(tb testing.TB, n int) string {
	tb.Helper()
	root := tb.TempDir()
	langs := []struct{ ext, body string }{
		{"go", "package x\n\n// doc\nfunc F%d() int { return %d }\n"},
		{"py", "# mod\ndef f%d():\n    return %d\n"},
		{"js", "// file\nfunction f%d(){ return %d; }\n"},
		{"java", "class C%d { int v(){ return %d; } }\n"},
	}
	for i := 0; i < n; i++ {
		l := langs[i%len(langs)]
		dir := filepath.Join(root, fmt.Sprintf("pkg%02d", i%50))
		_ = os.MkdirAll(dir, 0o755)
		p := filepath.Join(dir, fmt.Sprintf("f%d.%s", i, l.ext))
		if err := os.WriteFile(p, []byte(fmt.Sprintf(l.body, i, i)), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	return root
}

// BenchmarkScan measures scan throughput on a synthetic tree. Run:
//
//	go test -bench=BenchmarkScan -benchmem ./pkg/codeprint
//
// The files/sec custom metric is the NFR-1 signal. The >10% regression gate
// (docs/planning/code-quality.md) compares this across commits; the full
// big-repo perf corpus is a deferred submodule (BLOCKERS.md #1).
func BenchmarkScan(b *testing.B) {
	const files = 2000
	root := buildSyntheticTree(b, files)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fp, err := Scan(ctx, root)
		if err != nil {
			b.Fatal(err)
		}
		if fp.Totals.Files != files {
			b.Fatalf("expected %d files, got %d", files, fp.Totals.Files)
		}
	}
	b.StopTimer()
	secs := b.Elapsed().Seconds()
	if secs > 0 {
		b.ReportMetric(float64(files*b.N)/secs, "files/sec")
	}
}
