// Package detect identifies a file's language and linguist-derived properties.
// enry is authoritative for language identity; the Detector interface keeps the
// implementation swappable (feasibility risk mitigation).
package detect

import (
	"path/filepath"

	enry "github.com/go-enry/go-enry/v2"
)

// Result holds detection output for a single file.
type Result struct {
	Language    string // enry/linguist language ID; "" if undetected
	IsBinary    bool
	IsVendor    bool
	IsGenerated bool
}

// Detector identifies language and properties from a path and its content.
type Detector interface {
	Detect(path string, content []byte) Result
}

// EnryDetector is the default detector, backed by go-enry.
type EnryDetector struct{}

// Detect classifies a file using enry. path is repo-relative; content may be
// nil (e.g. oversized files), in which case detection falls back to filename.
func (EnryDetector) Detect(path string, content []byte) Result {
	base := filepath.Base(path)
	return Result{
		Language:    enry.GetLanguage(base, content),
		IsBinary:    len(content) > 0 && enry.IsBinary(content),
		IsVendor:    enry.IsVendor(path),
		IsGenerated: enry.IsGenerated(path, content),
	}
}
