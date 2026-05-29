// Package walk traverses a workspace, applying ignore rules, hidden-file
// policy, symlink containment (NFR-8), and the max-file-size gate. It yields
// lightweight file references; content is read by the caller's workers.
package walk

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/denormal/go-gitignore"
)

// Config controls traversal.
type Config struct {
	IncludeHidden bool
	MaxFileSize   int64
	IgnoreFile    string // extra ignore file, layered on .gitignore/.codeprintignore
}

// FileRef is a discovered file. Rel is repo-relative (forward slashes); Abs is
// the absolute path to read; Oversized marks files past MaxFileSize (not read).
type FileRef struct {
	Rel       string
	Abs       string
	Size      int64
	Oversized bool
}

// Walk returns the files under root that pass the ignore/hidden/symlink rules,
// in filesystem order (the caller sorts for determinism). It returns an error
// only when root itself is unreadable.
func Walk(root string, cfg Config) ([]FileRef, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absRoot); err != nil {
		return nil, err
	}

	matchers := buildMatchers(absRoot, cfg)
	var refs []FileRef

	walkErr := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Unreadable dir/file: skip it; root-level errors are handled above.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == absRoot {
			return nil
		}
		base := d.Name()

		// Always skip the VCS directory.
		if d.IsDir() && base == ".git" {
			return fs.SkipDir
		}
		// Hidden-file policy.
		if !cfg.IncludeHidden && strings.HasPrefix(base, ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		rel := relSlash(absRoot, path)

		// Ignore rules (nested .gitignore + .codeprintignore + extra file).
		if isIgnored(matchers, rel, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Symlinks: contain to the workspace root.
		if d.Type()&fs.ModeSymlink != 0 {
			ref, ok := resolveSymlink(absRoot, path, rel, cfg.MaxFileSize)
			if ok {
				refs = append(refs, ref)
			}
			// Symlinked directories are not descended in M1 (avoids loops).
			return nil
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		refs = append(refs, fileRef(rel, path, info.Size(), cfg.MaxFileSize))
		return nil
	})
	return refs, walkErr
}

func fileRef(rel, abs string, size, max int64) FileRef {
	return FileRef{Rel: rel, Abs: abs, Size: size, Oversized: size > max}
}

// resolveSymlink follows a symlink only if its target stays within absRoot and
// is a regular file. Escaping or unreadable links are skipped (ok=false).
func resolveSymlink(absRoot, path, rel string, max int64) (FileRef, bool) {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return FileRef{}, false // dangling or unresolvable
	}
	if !withinRoot(absRoot, target) {
		return FileRef{}, false // escapes the workspace — NFR-8 containment
	}
	info, err := os.Stat(target)
	if err != nil || !info.Mode().IsRegular() {
		return FileRef{}, false
	}
	return fileRef(rel, path, info.Size(), max), true
}

// withinRoot reports whether target is inside (or equal to) absRoot.
func withinRoot(absRoot, target string) bool {
	if target == absRoot {
		return true
	}
	return strings.HasPrefix(target, absRoot+string(os.PathSeparator))
}

func relSlash(absRoot, path string) string {
	rel, err := filepath.Rel(absRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// matchers holds the layered ignore matchers; nil entries are skipped.
type matchers struct {
	git    gitignore.GitIgnore
	cprint gitignore.GitIgnore
	extra  gitignore.GitIgnore
}

func buildMatchers(absRoot string, cfg Config) matchers {
	var m matchers
	m.git, _ = gitignore.NewRepository(absRoot)
	m.cprint, _ = gitignore.NewRepositoryWithFile(absRoot, ".codeprintignore")
	if cfg.IgnoreFile != "" {
		m.extra, _ = gitignore.NewFromFile(cfg.IgnoreFile)
	}
	return m
}

func isIgnored(m matchers, rel string, isDir bool) bool {
	for _, gi := range []gitignore.GitIgnore{m.git, m.cprint, m.extra} {
		if gi == nil {
			continue
		}
		if match := gi.Relative(rel, isDir); match != nil && match.Ignore() {
			return true
		}
	}
	return false
}
