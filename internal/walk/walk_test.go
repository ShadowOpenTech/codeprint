package walk

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func rels(refs []FileRef) map[string]bool {
	m := map[string]bool{}
	for _, r := range refs {
		m[r.Rel] = true
	}
	return m
}

func TestWalkIgnoreAndHidden(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "keep.go", "package x\n")
	write(t, dir, "ignored.log", "noise\n")
	write(t, dir, "sub/keep2.py", "x=1\n")
	write(t, dir, "build/out.o", "bin\n")
	write(t, dir, ".gitignore", "*.log\nbuild/\n")
	write(t, dir, ".hidden", "secret\n")
	write(t, dir, ".cpignore_target.txt", "viacp\n")
	write(t, dir, ".codeprintignore", ".cpignore_target.txt\n")

	t.Run("default includes hidden, applies ignores", func(t *testing.T) {
		refs, err := Walk(dir, Config{IncludeHidden: true, MaxFileSize: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		got := rels(refs)
		if !got["keep.go"] || !got["sub/keep2.py"] {
			t.Errorf("expected kept files, got %v", got)
		}
		if got["ignored.log"] {
			t.Error(".gitignore *.log not applied")
		}
		if got["build/out.o"] {
			t.Error(".gitignore build/ not applied")
		}
		if got[".cpignore_target.txt"] {
			t.Error(".codeprintignore not applied")
		}
		if !got[".hidden"] {
			t.Error("hidden file should be included by default")
		}
	})

	t.Run("no-hidden excludes dotfiles", func(t *testing.T) {
		refs, err := Walk(dir, Config{IncludeHidden: false, MaxFileSize: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		if rels(refs)[".hidden"] {
			t.Error("hidden file should be excluded with IncludeHidden=false")
		}
	})
}

func TestWalkExtraIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.txt", "a\n")
	write(t, dir, "b.txt", "b\n")
	ignore := filepath.Join(t.TempDir(), "extra.ignore")
	if err := os.WriteFile(ignore, []byte("b.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	refs, err := Walk(dir, Config{IncludeHidden: true, MaxFileSize: 1 << 20, IgnoreFile: ignore})
	if err != nil {
		t.Fatal(err)
	}
	got := rels(refs)
	if !got["a.txt"] || got["b.txt"] {
		t.Errorf("extra ignore file not applied: %v", got)
	}
}

func TestWalkMaxFileSize(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "small.txt", "hi\n")
	write(t, dir, "big.txt", "0123456789ABCDEF\n")
	refs, err := Walk(dir, Config{IncludeHidden: true, MaxFileSize: 5})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range refs {
		switch r.Rel {
		case "big.txt":
			if !r.Oversized {
				t.Error("big.txt should be oversized")
			}
		case "small.txt":
			if r.Oversized {
				t.Error("small.txt should not be oversized")
			}
		}
	}
}

func TestWalkUnreadableRoot(t *testing.T) {
	_, err := Walk(filepath.Join(t.TempDir(), "nope"), Config{MaxFileSize: 1 << 20})
	if err == nil {
		t.Error("expected error for nonexistent root")
	}
}

func TestWithinRoot(t *testing.T) {
	root := "/a/b"
	cases := map[string]bool{
		"/a/b":   true,
		"/a/b/c": true,
		"/a/bc":  false,
		"/a":     false,
		"/x":     false,
	}
	for target, want := range cases {
		if got := withinRoot(root, target); got != want {
			t.Errorf("withinRoot(%q,%q) = %v, want %v", root, target, got, want)
		}
	}
}
