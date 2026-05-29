// Package classify assigns the primary kind of a file plus secondary flags.
//
// Priority (F-2): binary > vendored > generated > minified > test > source.
// A file may also carry non-exclusive flags (e.g. a generated test file is
// kind=generated with flags=["test"]).
//
// Kinds are plain strings matching the schema enum values; pkg/codeprint
// converts them to its Kind type. classify intentionally does not import
// pkg/codeprint (that would create an import cycle).
package classify

import (
	"bytes"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ShadowOpenTech/codeprint/internal/detect"
)

// Kind string values, matching codeprint.Kind enum.
const (
	Source    = "source"
	Test      = "test"
	Generated = "generated"
	Vendored  = "vendored"
	Binary    = "binary"
	Minified  = "minified"
)

// FlagTest marks a non-test-kind file that is nonetheless a test (e.g. a
// generated test file).
const FlagTest = "test"

// Classify returns the primary kind and sorted secondary flags for a file.
//
// Priority refinement vs spec F-2: explicit minified detection is checked
// BEFORE vendored/generated. enry's linguist heuristics fold minified files
// into "vendored" (.min.* extension) and "generated" (very long lines), which
// would make our dedicated `minified` kind unreachable. Since `minified` is the
// more specific, more actionable signal for scanners (skip built assets), it
// takes precedence. Order: binary > minified > vendored > generated > test >
// source. (Logged for review in BLOCKERS.md.)
func Classify(path string, content []byte, det detect.Result) (kind string, flags []string) {
	test := IsTest(path)

	switch {
	case det.IsBinary:
		kind = Binary
	case isMinified(path, content):
		kind = Minified
	case det.IsVendor:
		kind = Vendored
	case det.IsGenerated || hasGeneratedHeader(content):
		kind = Generated
	case test:
		kind = Test
	default:
		kind = Source
	}

	if test && kind != Test {
		flags = append(flags, FlagTest)
	}
	sort.Strings(flags)
	return kind, flags
}

// CountsLOC reports whether a file of this kind contributes to LOC totals.
// Binary, vendored, and minified files are counted but excluded from LOC (F-1).
func CountsLOC(kind string) bool {
	switch kind {
	case Binary, Vendored, Minified:
		return false
	default:
		return true
	}
}

// IsTest applies path heuristics for test files across ecosystems.
func IsTest(path string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(base)

	switch {
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.HasSuffix(lower, ".spec.ts"), strings.HasSuffix(lower, ".spec.js"),
		strings.HasSuffix(lower, ".test.ts"), strings.HasSuffix(lower, ".test.js"),
		strings.HasSuffix(lower, ".spec.tsx"), strings.HasSuffix(lower, ".test.tsx"):
		return true
	case strings.HasSuffix(base, "Test.java"), strings.HasSuffix(base, "Tests.java"), strings.HasSuffix(base, "IT.java"):
		return true
	case strings.HasPrefix(lower, "test_") && strings.HasSuffix(lower, ".py"),
		strings.HasSuffix(lower, "_test.py"):
		return true
	case strings.HasSuffix(lower, "_spec.rb"), strings.HasSuffix(lower, "_test.rb"):
		return true
	}

	// Directory-based heuristics.
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Dir(path)), "/") {
		switch seg {
		case "test", "tests", "spec", "__tests__":
			return true
		}
	}
	return false
}

// minified extensions and heuristic thresholds.
var minifiedExts = map[string]bool{".min.js": true, ".min.css": true}

func isMinified(path string, content []byte) bool {
	lower := strings.ToLower(path)
	for ext := range minifiedExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	if len(content) == 0 {
		return false
	}
	newlines := bytes.Count(content, []byte("\n"))
	lines := newlines + 1
	avgLineLen := float64(len(content)) / float64(lines)
	newlineRatio := float64(newlines) / float64(len(content))
	return avgLineLen > 500 && newlineRatio < 0.01
}

// generatedHeaderRe matches the canonical generated-file markers, anchored to a
// line start (after optional comment punctuation) so prose mentioning "do not
// edit" does not trigger a false positive. Covers the Go "Code generated ...
// DO NOT EDIT" convention and the @generated token.
var generatedHeaderRe = regexp.MustCompile(`(?im)^.{0,8}code generated .*do not edit|@generated`)

func hasGeneratedHeader(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	head := content
	if len(head) > 2048 {
		head = head[:2048]
	}
	return generatedHeaderRe.Match(head)
}
