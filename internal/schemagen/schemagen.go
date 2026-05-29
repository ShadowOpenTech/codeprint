// Package schemagen generates the canonical JSON schema from the codeprint
// public types. The generated file is committed and CI fails on drift.
package schemagen

import (
	"bytes"
	"encoding/json"

	"github.com/ShadowOpenTech/codeprint/pkg/codeprint"
	"github.com/invopop/jsonschema"
)

// Generate returns the canonical schema bytes for the Output type as
// pretty-printed JSON Schema 2020-12, with a trailing newline.
func Generate() ([]byte, error) {
	r := &jsonschema.Reflector{
		// Deterministic, self-contained schema: no $ref indirection ordering
		// surprises, no network lookups for comments.
		DoNotReference: true,
	}
	s := r.Reflect(&codeprint.Output{})

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
