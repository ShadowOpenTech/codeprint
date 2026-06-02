package codeprint

import (
	"context"
	"encoding/json"
	"testing"
)

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

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
	ja, _ := jsonMarshal(a)
	jb, _ := jsonMarshal(b)
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
