package codeprint

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestScanFilesBasic(t *testing.T) {
	files := map[string][]byte{
		"main.go":          []byte("package main\n\n// doc\nfunc main() {}\n"),
		"app.py":           []byte("import flask\n\n\ndef f():\n    return 1\n"),
		"package.json":     []byte(`{"dependencies":{"react":"^18"}}`),
		"requirements.txt": []byte("Flask==3.0\n"),
	}
	fp, err := ScanFiles(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if fp.Totals.Files != 4 {
		t.Errorf("files = %d, want 4", fp.Totals.Files)
	}
	langs := map[string]bool{}
	for _, l := range fp.Languages {
		langs[l.Language] = true
	}
	if !langs["Go"] || !langs["Python"] {
		t.Errorf("expected Go + Python, got %v", langs)
	}
	bs := map[string]bool{}
	for _, b := range fp.BuildSystems {
		bs[b.Ecosystem] = true
	}
	if !bs["npm"] || !bs["pip"] {
		t.Errorf("expected npm + pip build systems, got %v", bs)
	}
	fw := map[string]bool{}
	for _, h := range fp.FrameworkHints {
		fw[h.Name] = true
	}
	if !fw["react"] || !fw["flask"] {
		t.Errorf("expected react + flask hints, got %v", fw)
	}
}

func TestScanFilesDeterministic(t *testing.T) {
	files := map[string][]byte{"a.go": []byte("package a\n"), "b.go": []byte("package b\n")}
	a, _ := ScanFiles(context.Background(), files)
	b, _ := ScanFiles(context.Background(), files)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Error("ScanFiles not deterministic")
	}
}

func TestScanFilesEmpty(t *testing.T) {
	fp, err := ScanFiles(context.Background(), map[string][]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if fp.Totals.Files != 0 || fp.Files == nil || fp.Languages == nil {
		t.Errorf("empty input must yield zero totals and non-nil lists: %+v", fp.Totals)
	}
}

// TestScanFilesParityWithScan pins the core invariant the PR exists for: the
// disk path Scan and the in-memory path ScanFiles must produce byte-identical
// fingerprints for the same file set, except for the Symlinks report (the disk
// path may populate it; the in-memory path is always empty).
func TestScanFilesParityWithScan(t *testing.T) {
	files := map[string][]byte{
		"main.go":          []byte("package main\n\n// doc\nfunc main() {}\n"),
		"app.py":           []byte("import flask\n\n\ndef f():\n    return 1\n"),
		"lib.ts":           []byte("export const x: number = 1;\n"),
		"package.json":     []byte(`{"dependencies":{"react":"^18"}}`),
		"requirements.txt": []byte("Flask==3.0\n"),
	}

	dir := t.TempDir()
	for p, c := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, c, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	disk, err := Scan(context.Background(), dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	mem, err := ScanFiles(context.Background(), files)
	if err != nil {
		t.Fatalf("ScanFiles: %v", err)
	}

	// Ignore the deliberately-empty Symlinks report on the in-memory path by
	// blanking it on both before comparing.
	disk.Symlinks = SymlinkReport{}
	mem.Symlinks = SymlinkReport{}

	jd, err := json.Marshal(disk)
	if err != nil {
		t.Fatal(err)
	}
	jm, err := json.Marshal(mem)
	if err != nil {
		t.Fatal(err)
	}
	if string(jd) != string(jm) {
		t.Errorf("Scan and ScanFiles fingerprints differ (excluding symlinks):\n disk = %s\n mem  = %s", jd, jm)
	}
}

// TestScanFilesOversized covers the max-file-size path via ScanFiles: a file
// past the ceiling is counted but flagged skipped_large with 0 LOC.
func TestScanFilesOversized(t *testing.T) {
	big := []byte("package big\n" + string(make([]byte, 4096)))
	files := map[string][]byte{
		"small.go": []byte("package small\n\nfunc f() {}\n"),
		"big.go":   big,
	}
	fp, err := ScanFiles(context.Background(), files, WithMaxFileSize(64))
	if err != nil {
		t.Fatal(err)
	}
	var rec *FileRecord
	for i := range fp.Files {
		if fp.Files[i].Path == "big.go" {
			rec = &fp.Files[i]
		}
	}
	if rec == nil {
		t.Fatal("big.go not present in file records")
	}
	flagged := false
	for _, f := range rec.Flags {
		if f == flagSkippedLarge {
			flagged = true
		}
	}
	if !flagged {
		t.Errorf("big.go missing %q flag, flags = %v", flagSkippedLarge, rec.Flags)
	}
	if rec.Code != 0 || rec.Comment != 0 || rec.Blank != 0 {
		t.Errorf("oversized file must have 0 LOC, got code=%d comment=%d blank=%d", rec.Code, rec.Comment, rec.Blank)
	}
}

// TestScanFilesNoFileRecords covers WithoutFileRecords via ScanFiles: the
// per-file list is empty while rollups and totals remain populated.
func TestScanFilesNoFileRecords(t *testing.T) {
	files := map[string][]byte{
		"main.go": []byte("package main\n\nfunc main() {}\n"),
		"app.py":  []byte("def f():\n    return 1\n"),
	}
	fp, err := ScanFiles(context.Background(), files, WithoutFileRecords())
	if err != nil {
		t.Fatal(err)
	}
	if len(fp.Files) != 0 {
		t.Errorf("WithoutFileRecords must yield empty fp.Files, got %d records", len(fp.Files))
	}
	if fp.Totals.Files != 2 {
		t.Errorf("totals must remain, files = %d, want 2", fp.Totals.Files)
	}
	if len(fp.Languages) == 0 {
		t.Error("language rollups must remain populated")
	}
}
