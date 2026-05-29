package emit

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ShadowOpenTech/codeprint/pkg/codeprint"
)

func sampleOutput() codeprint.Output {
	fp := &codeprint.Fingerprint{
		Totals: codeprint.Totals{Files: 3, Code: 100, Comment: 10, Blank: 5},
		Languages: []codeprint.LanguageRollup{
			{Language: "Go", Files: 2, Code: 80, Percent: 80},
			{Language: "Python", Files: 1, Code: 20, Percent: 20},
		},
		BuildSystems:   []codeprint.BuildSystem{{Ecosystem: "go-modules", Path: "go.mod"}},
		FrameworkHints: []codeprint.FrameworkHint{{Name: "gin", Ecosystem: "go-modules"}},
	}
	return codeprint.NewOutput(fp, "", "1.0.0", "2026-01-01T00:00:00Z")
}

func TestSanitize(t *testing.T) {
	in := "evil\x1b[31mred\x07\nname\tok"
	got := Sanitize(in)
	if strings.ContainsAny(got, "\x1b\x07\n") {
		t.Errorf("control chars not removed: %q", got)
	}
	if !strings.Contains(got, "\t") {
		t.Error("tab should be preserved")
	}
	// The ESC byte is stripped (neutralizing the sequence); leftover printable
	// chars like "[31m" are harmless and may remain.
	for _, want := range []string{"evil", "red", "name", "ok"} {
		if !strings.Contains(got, want) {
			t.Errorf("printable text %q missing from %q", want, got)
		}
	}
}

func TestSummary(t *testing.T) {
	var b bytes.Buffer
	if err := Summary(&b, sampleOutput()); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{"Go 80%", "Python 20%", "115 LOC", "3 files", "1 frameworks"} {
		if !strings.Contains(s, want) {
			t.Errorf("summary missing %q: %s", want, s)
		}
	}
}

func TestPretty(t *testing.T) {
	var b bytes.Buffer
	if err := Pretty(&b, sampleOutput(), "/repo", 830*time.Millisecond, false); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{"codeprint", "/repo", "Languages", "Go", "Python", "Totals", "Build", "go-modules", "Frameworks", "gin", "Scanned in"} {
		if !strings.Contains(s, want) {
			t.Errorf("pretty missing %q", want)
		}
	}
	// No ANSI escapes when color disabled.
	if strings.Contains(s, "\x1b[") {
		t.Errorf("expected no ANSI escapes with color off: %q", s)
	}
}

func TestHumanInt(t *testing.T) {
	cases := map[int]string{0: "0", 999: "999", 1000: "1,000", 38120: "38,120", 1234567: "1,234,567", -4200: "-4,200"}
	for n, want := range cases {
		if got := humanInt(n); got != want {
			t.Errorf("humanInt(%d) = %q want %q", n, got, want)
		}
	}
}
