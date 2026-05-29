package codeprint

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// expectedCorpus mirrors testdata/corpus/expected.json (ground-truth oracle).
type expectedCorpus struct {
	Files map[string]struct {
		Language string `json:"language"`
		Kind     string `json:"kind"`
	} `json:"files"`
	BuildSystems []struct {
		Ecosystem string `json:"ecosystem"`
		Path      string `json:"path"`
	} `json:"build_systems"`
	Container struct {
		DockerfilePresent bool `json:"dockerfile_present"`
	} `json:"container"`
}

func loadExpected(t *testing.T) expectedCorpus {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(corpus, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var e expectedCorpus
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	return e
}

// TestCorpusAccuracy is the TP/FP/FN gate: every labelled file and build system
// must match. The corpus is controlled, so the bar is 100%.
func TestCorpusAccuracy(t *testing.T) {
	exp := loadExpected(t)
	fp, err := Scan(context.Background(), corpus)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]FileRecord{}
	for _, f := range fp.Files {
		got[f.Path] = f
	}

	// File classification + language.
	for path, want := range exp.Files {
		f, ok := got[path]
		if !ok {
			t.Errorf("FN: labelled file %q absent from output", path)
			continue
		}
		if want.Kind != "" && string(f.Kind) != want.Kind {
			t.Errorf("kind %q: got %q want %q", path, f.Kind, want.Kind)
		}
		if want.Language != "" && f.Language != want.Language {
			t.Errorf("language %q: got %q want %q", path, f.Language, want.Language)
		}
	}

	// Build systems: exact set match (flat list with paths).
	gotBS := map[string]string{}
	for _, b := range fp.BuildSystems {
		gotBS[b.Path] = b.Ecosystem
	}
	for _, want := range exp.BuildSystems {
		if gotBS[want.Path] != want.Ecosystem {
			t.Errorf("build system %q: got %q want %q", want.Path, gotBS[want.Path], want.Ecosystem)
		}
	}
	if len(gotBS) != len(exp.BuildSystems) {
		t.Errorf("build system count: got %d want %d", len(gotBS), len(exp.BuildSystems))
	}

	if fp.Container.DockerfilePresent != exp.Container.DockerfilePresent {
		t.Errorf("dockerfile_present: got %v want %v", fp.Container.DockerfilePresent, exp.Container.DockerfilePresent)
	}
}
