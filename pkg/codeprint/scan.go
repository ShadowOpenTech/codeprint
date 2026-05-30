package codeprint

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
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

	processed, scanErrs, err := process(ctx, refs, cfg)
	if err != nil {
		return nil, err
	}

	fp := assemble(processed, scanErrs, cfg)

	// Build-system detection: flat full-tree marker scan over all walked paths.
	paths := make([]string, len(refs))
	for i, ref := range refs {
		paths[i] = ref.Rel
	}
	markers, dockerfile := buildsys.Detect(paths)
	for _, m := range markers {
		fp.BuildSystems = append(fp.BuildSystems, BuildSystem{Ecosystem: m.Ecosystem, Path: m.Path})
	}
	fp.Container.DockerfilePresent = dockerfile

	// Symlink report: surface traversal gaps (skipped dirs / escaping links).
	fp.Symlinks = SymlinkReport{
		Total:        symInfo.Total,
		FollowedFile: symInfo.FollowedFile,
		Skipped:      []SkippedSymlink{},
	}
	for _, s := range symInfo.Skipped {
		fp.Symlinks.Skipped = append(fp.Symlinks.Skipped, SkippedSymlink{Path: s.Path, Reason: s.Reason})
	}

	// Framework hints: parse the detected manifests (best-effort, F-4).
	absRoot, _ := filepath.Abs(root)
	for _, h := range framework.Detect(absRoot, markers) {
		fp.FrameworkHints = append(fp.FrameworkHints, FrameworkHint{
			Name:       h.Name,
			Ecosystem:  h.Ecosystem,
			Confidence: Confidence(h.Confidence),
			Evidence:   Evidence{Path: h.Path, Keyword: h.Keyword},
		})
	}

	return fp, nil
}

// processedFile is the per-file result before assembly.
type processedFile struct {
	rec          FileRecord
	commentAware bool
	countsLOC    bool
}

var detector detect.Detector = detect.EnryDetector{}

// process runs detection/classification/counting across refs with a bounded
// worker pool, recovering per-file panics into scan errors.
func process(ctx context.Context, refs []walk.FileRef, cfg *config) ([]processedFile, []ScanError, error) {
	var (
		mu       sync.Mutex
		files    = make([]processedFile, 0, len(refs))
		scanErrs []ScanError
		sem      = make(chan struct{}, cfg.concurrency)
		wg       sync.WaitGroup
	)

	for _, ref := range refs {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil, nil, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(ref walk.FileRef) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					scanErrs = append(scanErrs, ScanError{Path: ref.Rel, Reason: fmt.Sprintf("internal: %v", r)})
					mu.Unlock()
				}
			}()

			pf, serr := processOne(ref)
			progress.Report()
			mu.Lock()
			if serr != nil {
				scanErrs = append(scanErrs, *serr)
			} else {
				files = append(files, pf)
			}
			mu.Unlock()
		}(ref)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return files, scanErrs, nil
}

// processOne handles a single file: read, detect, classify, count.
func processOne(ref walk.FileRef) (processedFile, *ScanError) {
	rec := FileRecord{Path: ref.Rel, Bytes: ref.Size, Flags: []string{}}

	if ref.Oversized {
		rec.Kind = KindSource
		rec.Flags = []string{flagSkippedLarge}
		// Language by filename only (no content read).
		rec.Language = detector.Detect(ref.Rel, nil).Language
		return processedFile{rec: rec}, nil
	}

	content, err := os.ReadFile(ref.Abs)
	if err != nil {
		return processedFile{}, &ScanError{Path: ref.Rel, Reason: err.Error()}
	}

	det := detector.Detect(ref.Rel, content)
	rec.Language = det.Language
	kind, flags := classify.Classify(ref.Rel, content, det)
	rec.Kind = Kind(kind)
	if len(flags) > 0 {
		rec.Flags = flags
	}

	pf := processedFile{rec: rec}
	if classify.CountsLOC(kind) {
		c := loc.Count(ref.Rel, det.Language, content)
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
