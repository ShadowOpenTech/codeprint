package codeprint

import "sort"

// fileMeta is a discovered file without its content (content is read lazily).
type fileMeta struct {
	rel       string
	size      int64
	oversized bool
}

// provider supplies files to fingerprint and reads content lazily, so the disk
// path never holds all content in memory at once.
type provider interface {
	metas() []fileMeta
	read(rel string) ([]byte, error)
	paths() []string         // all rel paths (for build-system + framework detection)
	symlinks() SymlinkReport // disk has a real report; in-memory is empty
}

// memProvider fingerprints an in-memory file set (used by ScanFiles / WASM).
type memProvider struct {
	files map[string][]byte
	max   int64
}

func (m *memProvider) metas() []fileMeta {
	out := make([]fileMeta, 0, len(m.files))
	for p, c := range m.files {
		out = append(out, fileMeta{rel: p, size: int64(len(c)), oversized: int64(len(c)) > m.max})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}

func (m *memProvider) read(rel string) ([]byte, error) { return m.files[rel], nil }

func (m *memProvider) paths() []string {
	p := make([]string, 0, len(m.files))
	for k := range m.files {
		p = append(p, k)
	}
	sort.Strings(p)
	return p
}

func (m *memProvider) symlinks() SymlinkReport {
	return SymlinkReport{Skipped: []SkippedSymlink{}}
}
