package codeprint

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/ShadowOpenTech/codeprint/internal/buildsys"
	"github.com/ShadowOpenTech/codeprint/internal/classify"
	"github.com/ShadowOpenTech/codeprint/internal/detect"
	"github.com/ShadowOpenTech/codeprint/internal/framework"
	"github.com/ShadowOpenTech/codeprint/internal/loc"
	"github.com/ShadowOpenTech/codeprint/internal/progress"
	"github.com/ShadowOpenTech/codeprint/internal/walk"
)

// Exported sentinel errors. Consumers branch with errors.Is; the CLI maps them
// to sysexits codes (see docs/architecture/error-strategy.md).
var (
	// ErrUnreadableRoot is returned when the workspace root cannot be read.
	ErrUnreadableRoot = errors.New("codeprint: workspace root unreadable")
)

// flagSkippedLarge marks files past the max-file-size ceiling.
const flagSkippedLarge = "skipped_large"

// Scan walks root and returns its fingerprint. ctx cancellation aborts the
// scan cleanly, returning (nil, ctx.Err()) — never a partial result.
func Scan(ctx context.Context, root string, opts ...Option) (*Fingerprint, error) {
	cfg := newConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	refs, symInfo, err := walk.Walk(root, walk.Config{
		IncludeHidden: cfg.includeHidden,
		MaxFileSize:   cfg.maxFileSize,
		IgnoreFile:    cfg.ignoreFile,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrUnreadableRoot, root, err)
	}

	return core(ctx, newDiskProvider(refs, symInfo), cfg)
}

// ScanFiles fingerprints an in-memory set of files (path -> content) without
// touching the filesystem. Useful for in-process consumers that already hold
// file contents (and the WASM playground). Filesystem-only behavior (ignore
// rules, symlink handling) does not apply.
func ScanFiles(ctx context.Context, files map[string][]byte, opts ...Option) (*Fingerprint, error) {
	cfg := newConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return core(ctx, &memProvider{files: files, max: cfg.maxFileSize}, cfg)
}

// core runs the shared post-discovery pipeline (process → assemble → buildsys →
// symlinks → framework) over any provider, so the disk and in-memory paths
// produce identical fingerprints from the same file set.
func core(ctx context.Context, p provider, cfg *config) (*Fingerprint, error) {
	processed, scanErrs, err := process(ctx, p, cfg)
	if err != nil {
		return nil, err
	}

	fp := assemble(processed, scanErrs, cfg)

	// Build-system detection: flat full-tree marker scan over all paths.
	paths := p.paths()
	markers, dockerfile := buildsys.Detect(paths)
	for _, m := range markers {
		fp.BuildSystems = append(fp.BuildSystems, BuildSystem{Ecosystem: m.Ecosystem, Path: m.Path})
	}
	fp.Container.DockerfilePresent = dockerfile

	// Symlink report: surface traversal gaps (skipped dirs / escaping links).
	fp.Symlinks = p.symlinks()

	// Framework hints: parse the detected manifests (best-effort, F-4).
	for _, h := range framework.Detect(markers, p.read) {
		fp.FrameworkHints = append(fp.FrameworkHints, FrameworkHint{
			Name:       h.Name,
			Ecosystem:  h.Ecosystem,
			Confidence: Confidence(h.Confidence),
			Evidence:   Evidence{Path: h.Path, Keyword: h.Keyword},
		})
	}

	return fp, nil
}

// diskProvider reads content from the filesystem on demand.
type diskProvider struct {
	relToAbs map[string]string
	refs     []walk.FileRef
	sym      walk.SymlinkInfo
}

func newDiskProvider(refs []walk.FileRef, sym walk.SymlinkInfo) *diskProvider {
	m := make(map[string]string, len(refs))
	for _, r := range refs {
		m[r.Rel] = r.Abs
	}
	return &diskProvider{relToAbs: m, refs: refs, sym: sym}
}

func (d *diskProvider) metas() []fileMeta {
	out := make([]fileMeta, len(d.refs))
	for i, r := range d.refs {
		out[i] = fileMeta{rel: r.Rel, size: r.Size, oversized: r.Oversized}
	}
	return out
}

func (d *diskProvider) read(rel string) ([]byte, error) {
	abs, ok := d.relToAbs[rel]
	if !ok {
		return nil, nil // soft: absent path yields no content (no framework hint)
	}
	return os.ReadFile(abs) //nolint:gosec // abs derived from our own contained walk
}

func (d *diskProvider) paths() []string {
	p := make([]string, len(d.refs))
	for i, r := range d.refs {
		p[i] = r.Rel
	}
	return p
}

func (d *diskProvider) symlinks() SymlinkReport {
	rep := SymlinkReport{Total: d.sym.Total, FollowedFile: d.sym.FollowedFile, Skipped: []SkippedSymlink{}}
	for _, s := range d.sym.Skipped {
		rep.Skipped = append(rep.Skipped, SkippedSymlink{Path: s.Path, Reason: s.Reason})
	}
	return rep
}

// processedFile is the per-file result before assembly.
type processedFile struct {
	rec          FileRecord
	commentAware bool
	countsLOC    bool
}

var detector detect.Detector = detect.EnryDetector{}

// process runs detection/classification/counting across the provider's files
// with a bounded worker pool, recovering per-file panics into scan errors.
// Content is read lazily via p.read in each worker.
func process(ctx context.Context, p provider, cfg *config) ([]processedFile, []ScanError, error) {
	metas := p.metas()
	var (
		mu       sync.Mutex
		files    = make([]processedFile, 0, len(metas))
		scanErrs []ScanError
		sem      = make(chan struct{}, cfg.concurrency)
		wg       sync.WaitGroup
	)

	for _, meta := range metas {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil, nil, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(meta fileMeta) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					scanErrs = append(scanErrs, ScanError{Path: meta.rel, Reason: fmt.Sprintf("internal: %v", r)})
					mu.Unlock()
				}
			}()

			var (
				content []byte
				err     error
			)
			if !meta.oversized {
				content, err = p.read(meta.rel)
			}
			pf, serr := processOne(meta, content, err)
			progress.Report()
			mu.Lock()
			if serr != nil {
				scanErrs = append(scanErrs, *serr)
			} else {
				files = append(files, pf)
			}
			mu.Unlock()
		}(meta)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return files, scanErrs, nil
}

// processOne handles a single file: detect, classify, count over already-read
// content. readErr is the error (if any) from reading the file's content.
func processOne(meta fileMeta, content []byte, readErr error) (processedFile, *ScanError) {
	rec := FileRecord{Path: meta.rel, Bytes: meta.size, Flags: []string{}}

	if meta.oversized {
		rec.Kind = KindSource
		rec.Flags = []string{flagSkippedLarge}
		// Language by filename only (no content read).
		rec.Language = detector.Detect(meta.rel, nil).Language
		return processedFile{rec: rec}, nil
	}

	if readErr != nil {
		return processedFile{}, &ScanError{Path: meta.rel, Reason: readErr.Error()}
	}

	det := detector.Detect(meta.rel, content)
	rec.Language = det.Language
	kind, flags := classify.Classify(meta.rel, content, det)
	rec.Kind = Kind(kind)
	if len(flags) > 0 {
		rec.Flags = flags
	}

	pf := processedFile{rec: rec}
	if classify.CountsLOC(kind) {
		c := loc.Count(meta.rel, det.Language, content)
		rec.Code, rec.Comment, rec.Blank = c.Code, c.Comment, c.Blank
		pf.commentAware = c.CommentAware
		pf.countsLOC = true
	}
	pf.rec = rec
	return pf, nil
}

// assemble builds the deterministic Fingerprint from processed files.
func assemble(files []processedFile, scanErrs []ScanError, cfg *config) *Fingerprint {
	fp := &Fingerprint{
		Languages:      []LanguageRollup{},
		Files:          []FileRecord{},
		BuildSystems:   []BuildSystem{},
		FrameworkHints: []FrameworkHint{},
		Errors:         []ScanError{},
		Workspaces:     nil,
	}

	type agg struct {
		files, code, comment, blank int
		commentAware                bool
	}
	langs := map[string]*agg{}

	for _, pf := range files {
		r := pf.rec
		fp.Totals.Files++
		fp.Totals.Bytes += r.Bytes
		if pf.countsLOC {
			fp.Totals.Code += r.Code
			fp.Totals.Comment += r.Comment
			fp.Totals.Blank += r.Blank
			if r.Language != "" {
				a := langs[r.Language]
				if a == nil {
					a = &agg{commentAware: pf.commentAware}
					langs[r.Language] = a
				}
				a.files++
				a.code += r.Code
				a.comment += r.Comment
				a.blank += r.Blank
			}
		}
		if cfg.emitFiles {
			fp.Files = append(fp.Files, r)
		}
	}

	// Language rollups: collect from the map then sort by ID (never emit map order).
	for name, a := range langs {
		percent := 0.0
		if fp.Totals.Code > 0 {
			percent = math.Round(float64(a.code)/float64(fp.Totals.Code)*10000) / 100
		}
		fp.Languages = append(fp.Languages, LanguageRollup{
			Language:     name,
			Files:        a.files,
			Code:         a.code,
			Comment:      a.comment,
			Blank:        a.blank,
			Percent:      percent,
			CommentAware: a.commentAware,
		})
	}
	sort.Slice(fp.Languages, func(i, j int) bool { return fp.Languages[i].Language < fp.Languages[j].Language })

	sort.Slice(fp.Files, func(i, j int) bool { return fp.Files[i].Path < fp.Files[j].Path })

	if len(scanErrs) > 0 {
		fp.Errors = append(fp.Errors, scanErrs...)
		sort.Slice(fp.Errors, func(i, j int) bool { return fp.Errors[i].Path < fp.Errors[j].Path })
	}

	return fp
}
