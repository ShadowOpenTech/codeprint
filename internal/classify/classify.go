// Package classify assigns the primary kind of a file.
//
// M1 implements the enry-derived kinds (binary, vendored, generated, source).
// The minified and test heuristics, generated-header regex, and the flags list
// are added in M2; the priority order from spec F-2 is preserved as those land.
//
// Kinds are plain strings matching the schema enum values; pkg/codeprint
// converts them to its Kind type. classify intentionally does not import
// pkg/codeprint (that would create an import cycle).
package classify

import "github.com/ShadowOpenTech/codeprint/internal/detect"

// Kind string values, matching codeprint.Kind enum.
const (
	Source    = "source"
	Test      = "test"
	Generated = "generated"
	Vendored  = "vendored"
	Binary    = "binary"
	Minified  = "minified"
)

// Kind returns the primary classification for a file, given its detection
// result. Priority (F-2): binary > vendored > generated > (minified > test, M2) > source.
func Kind(det detect.Result) string {
	switch {
	case det.IsBinary:
		return Binary
	case det.IsVendor:
		return Vendored
	case det.IsGenerated:
		return Generated
	default:
		return Source
	}
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
