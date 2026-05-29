package emit

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONCompact(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, map[string]int{"a": 1}); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, `"a":1`) {
		t.Errorf("compact JSON missing expected content: %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("compact JSON should not be indented: %q", got)
	}
}

func TestJSONIndent(t *testing.T) {
	var b bytes.Buffer
	if err := JSONIndent(&b, map[string]int{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "\n  ") {
		t.Errorf("indented JSON should contain indentation: %q", b.String())
	}
}

func TestJSONNoHTMLEscape(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, map[string]string{"u": "a&b<c>"}); err != nil {
		t.Fatal(err)
	}
	// EscapeHTML is off: & < > stay literal. With escaping ON the literal
	// substring would instead appear as a&b<c>, so its presence
	// is sufficient proof escaping is disabled.
	if !strings.Contains(b.String(), "a&b<c>") {
		t.Errorf("expected literal & < > (escaping should be off), got: %q", b.String())
	}
}
