// Package emit serializes a fingerprint as canonical JSON, a human "pretty"
// report, or a one-line summary. Human output sanitizes control characters in
// attacker-controlled strings (filenames) to prevent terminal injection
// (security-audit finding #2).
package emit

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"

	"github.com/ShadowOpenTech/codeprint/pkg/codeprint"
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

// Sanitize removes control characters (except tab) from s so untrusted
// filenames cannot inject terminal escape sequences.
func Sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1 // drop
		}
		return r
	}, s)
}

// Summary writes a one-line digest, e.g.:
// Go 62% • Python 28% • 48,212 LOC • 412 files • 3 frameworks
func Summary(w io.Writer, out codeprint.Output) error {
	fp := out.Fingerprint
	var parts []string
	top := topLanguages(fp.Languages, 3)
	for _, l := range top {
		parts = append(parts, fmt.Sprintf("%s %.0f%%", Sanitize(l.Language), l.Percent))
	}
	parts = append(
		parts,
		fmt.Sprintf("%s LOC", humanInt(fp.Totals.Code+fp.Totals.Comment+fp.Totals.Blank)),
		fmt.Sprintf("%s files", humanInt(fp.Totals.Files)),
		fmt.Sprintf("%d frameworks", len(fp.FrameworkHints)),
	)
	_, err := fmt.Fprintln(w, strings.Join(parts, " • "))
	return err
}

// Pretty writes a human-readable report. useColor enables ANSI styling; dur is
// the wall-clock scan time (human output only — never in the JSON).
func Pretty(w io.Writer, out codeprint.Output, root string, dur time.Duration, useColor bool) error {
	color.NoColor = !useColor
	head := color.New(color.FgCyan, color.Bold).SprintFunc()
	label := color.New(color.FgHiBlack).SprintFunc()
	fp := out.Fingerprint

	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n\n", head("codeprint"), Sanitize(root))

	if len(fp.Languages) > 0 {
		fmt.Fprintf(&b, "  %s\n", label("Languages"))
		langs := append([]codeprint.LanguageRollup(nil), fp.Languages...)
		sort.Slice(langs, func(i, j int) bool { return langs[i].Code > langs[j].Code })
		for _, l := range langs {
			loc := l.Code + l.Comment + l.Blank
			fmt.Fprintf(&b, "  %-14s %5.1f%%  %8s LOC  %5d files\n",
				Sanitize(l.Language), l.Percent, humanInt(loc), l.Files)
		}
		b.WriteByte('\n')
	}

	fmt.Fprintf(&b, "  %-11s %s files · %s LOC (%s code · %s comment · %s blank)\n",
		label("Totals"), humanInt(fp.Totals.Files),
		humanInt(fp.Totals.Code+fp.Totals.Comment+fp.Totals.Blank),
		humanInt(fp.Totals.Code), humanInt(fp.Totals.Comment), humanInt(fp.Totals.Blank))

	if len(fp.BuildSystems) > 0 {
		fmt.Fprintf(&b, "  %-11s %s\n", label("Build"), strings.Join(ecosystems(fp.BuildSystems), ", "))
	}
	if len(fp.FrameworkHints) > 0 {
		fmt.Fprintf(&b, "  %-11s %d hints (%s)\n", label("Frameworks"),
			len(fp.FrameworkHints), strings.Join(frameworkNames(fp.FrameworkHints, 5), ", "))
	}
	if len(fp.Errors) > 0 {
		fmt.Fprintf(&b, "  %-11s %d files\n", label("Errors"), len(fp.Errors))
	}
	fmt.Fprintf(&b, "  %-11s %s\n", label("Scanned in"), dur.Round(time.Millisecond))

	_, err := io.WriteString(w, b.String())
	return err
}

func topLanguages(langs []codeprint.LanguageRollup, n int) []codeprint.LanguageRollup {
	s := append([]codeprint.LanguageRollup(nil), langs...)
	sort.Slice(s, func(i, j int) bool { return s[i].Code > s[j].Code })
	if len(s) > n {
		s = s[:n]
	}
	return s
}

func ecosystems(bs []codeprint.BuildSystem) []string {
	seen := map[string]bool{}
	var out []string
	for _, b := range bs {
		if !seen[b.Ecosystem] {
			seen[b.Ecosystem] = true
			out = append(out, b.Ecosystem)
		}
	}
	sort.Strings(out)
	return out
}

func frameworkNames(hints []codeprint.FrameworkHint, max int) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range hints {
		if seen[h.Name] {
			continue
		}
		seen[h.Name] = true
		out = append(out, h.Name)
	}
	sort.Strings(out)
	if len(out) > max {
		out = append(out[:max], "…")
	}
	return out
}

// humanInt formats n with thousands separators.
func humanInt(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}
