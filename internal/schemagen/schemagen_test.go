package schemagen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateDeterministic asserts schema generation is byte-stable across
// runs — a prerequisite for the drift gate to be meaningful.
func TestGenerateDeterministic(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("schema generation is not deterministic")
	}
}

// TestCommittedSchemaUpToDate fails if the committed schema has drifted from
// the types. Mirrors the CI drift gate locally.
func TestCommittedSchemaUpToDate(t *testing.T) {
	committed, err := os.ReadFile(filepath.Join("..", "..", "schema", "codeprint-v1.schema.json"))
	if err != nil {
		t.Fatalf("read committed schema: %v", err)
	}
	gen, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(committed, gen) {
		t.Fatal("committed schema is stale — run: make schema")
	}
}
