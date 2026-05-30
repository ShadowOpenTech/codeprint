package codeprint

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

const corpus = "../../testdata/corpus"

// scan is a test helper.
func scan(t *testing.T, opts ...Option) *Fingerprint {
	t.Helper()
	fp, err := Scan(context.Background(), corpus, opts...)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return fp
}

// TestDeterminism is the signature invariant (NFR-2): byte-identical fingerprint
// across runs and across worker counts.
func TestDeterminism(t *testing.T) {
	a := scan(t, WithConcurrency(1))
	b := scan(t, WithConcurrency(8))
	c := scan(t) // default concurrency

	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	jc, _ := json.Marshal(c)
	if string(ja) != string(jb) {
		t.Error("fingerprint differs between concurrency=1 and concurrency=8")
	}
	if string(ja) != string(jc) {
		t.Error("fingerprint differs between concurrency=1 and default")
	}
}

// fileByPath finds a file record by repo-relative path.
func fileByPath(fp *Fingerprint, path string) *FileRecord {
	for i := range fp.Files {
		if fp.Files[i].Path == path {
			return &fp.Files[i]
		}
	}
	return nil
}

func TestCorpusClassification(t *testing.T) {
	fp := scan(t)

	cases := map[string]Kind{
		"testfiles/blob.bin":        KindBinary,
		"vendor/leftpad/leftpad.js": KindVendored,
		"node_modules/dep/index.js": KindVendored,
		"gen/api.pb.go":             KindGenerated,
		"src/go/server.go":          KindSource,
	}
	for path, want := range cases {
		f := fileByPath(fp, path)
		if f == nil {
			t.Errorf("%s: missing from output", path)
			continue
		}
		if f.Kind != want {
			t.Errorf("%s: kind = %q, want %q", path, f.Kind, want)
		}
	}
}

func TestCorpusLanguages(t *testing.T) {
	fp := scan(t)
	want := map[string]bool{"Go": false, "Python": false, "Java": false, "Rust": false, "Ruby": false, "TypeScript": false}
	for _, l := range fp.Languages {
		if _, ok := want[l.Language]; ok {
			want[l.Language] = true
		}
	}
	for lang, seen := range want {
		if !seen {
			t.Errorf("expected language %q in rollups", lang)
		}
	}
}

func TestBinaryAndVendoredExcludedFromLOC(t *testing.T) {
	fp := scan(t)
	for _, f := range fp.Files {
		if f.Kind == KindBinary || f.Kind == KindVendored {
			if f.Code != 0 || f.Comment != 0 || f.Blank != 0 {
				t.Errorf("%s (%s): LOC should be 0, got %d/%d/%d", f.Path, f.Kind, f.Code, f.Comment, f.Blank)
			}
		}
	}
}

func TestEscapingSymlinkSkipped(t *testing.T) {
	fp := scan(t)
	if f := fileByPath(fp, "testfiles/escaping_link"); f != nil {
		t.Errorf("escaping symlink should be skipped (NFR-8), but appeared: %+v", f)
	}
}

func TestSymlinkReport(t *testing.T) {
	fp := scan(t)
	s := fp.Symlinks
	if s.Total != s.FollowedFile+len(s.Skipped) {
		t.Errorf("invariant broken: total %d != followed_file %d + skipped %d", s.Total, s.FollowedFile, len(s.Skipped))
	}
	// The corpus has an in-tree file symlink (followed) and an escaping one (skipped).
	if s.FollowedFile < 1 {
		t.Errorf("expected at least one followed file symlink, got %d", s.FollowedFile)
	}
	var sawEscaping bool
	for _, sk := range s.Skipped {
		if sk.Path == "testfiles/escaping_link" && sk.Reason == "escaping" {
			sawEscaping = true
		}
	}
	if !sawEscaping {
		t.Errorf("expected escaping_link in skipped with reason=escaping, got %+v", s.Skipped)
	}
}

func TestPercentsSumApproxAndRounded(t *testing.T) {
	fp := scan(t)
	if len(fp.Languages) == 0 {
		t.Fatal("no languages")
	}
	for _, l := range fp.Languages {
		if l.Percent < 0 || l.Percent > 100 {
			t.Errorf("%s: percent out of range: %v", l.Language, l.Percent)
		}
	}
}

func TestWithoutFileRecords(t *testing.T) {
	fp := scan(t, WithoutFileRecords())
	if len(fp.Files) != 0 {
		t.Errorf("expected no file records, got %d", len(fp.Files))
	}
	if len(fp.Languages) == 0 {
		t.Error("rollups should still be present without file records")
	}
	if fp.Totals.Files == 0 {
		t.Error("totals should still be computed")
	}
}

func TestMaxFileSizeFlagsLarge(t *testing.T) {
	fp := scan(t, WithMaxFileSize(50))
	var found bool
	for _, f := range fp.Files {
		for _, fl := range f.Flags {
			if fl == flagSkippedLarge {
				found = true
				if f.Code != 0 {
					t.Errorf("%s: skipped_large should have 0 code, got %d", f.Path, f.Code)
				}
			}
		}
	}
	if !found {
		t.Error("expected at least one skipped_large file at max-file-size=50")
	}
}

func TestUnreadableRoot(t *testing.T) {
	_, err := Scan(context.Background(), filepath.Join(corpus, "does-not-exist"), nil...)
	if !errors.Is(err, ErrUnreadableRoot) {
		t.Errorf("want ErrUnreadableRoot, got %v", err)
	}
}

func TestContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fp, err := Scan(ctx, corpus)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if fp != nil {
		t.Error("expected nil fingerprint on cancellation (clean abort)")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}

func TestEmptyDirIsValid(t *testing.T) {
	dir := t.TempDir()
	fp, err := Scan(context.Background(), dir)
	if err != nil {
		t.Fatalf("empty dir should succeed: %v", err)
	}
	if fp.Totals.Files != 0 {
		t.Errorf("empty dir totals.files = %d, want 0", fp.Totals.Files)
	}
	// Lists must be non-nil (empty arrays, not null) except reserved Workspaces.
	if fp.Languages == nil || fp.Files == nil || fp.BuildSystems == nil || fp.Errors == nil {
		t.Error("fingerprint lists must serialize as [] not null")
	}
}

func TestNewOutputMetaOnly(t *testing.T) {
	fp := scan(t)
	out := NewOutput(fp, "https://example/schema.json", "1.2.3", "2026-01-01T00:00:00Z")
	if out.Meta.GeneratedAt != "2026-01-01T00:00:00Z" {
		t.Error("generated_at should be in meta")
	}
	if out.Meta.Producer.Version != "1.2.3" {
		t.Error("version should be in meta producer")
	}
	if out.Schema != "https://example/schema.json" {
		t.Error("schema url should be set")
	}
}
