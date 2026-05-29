// Package emit serializes a fingerprint. M1 implements canonical JSON; pretty
// and summary modes are added in M4.
package emit

import (
	"encoding/json"
	"io"
)

// JSON writes v as compact canonical JSON followed by a newline.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// JSONIndent writes v as indented JSON (for humans reading files / --pretty-json).
func JSONIndent(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
