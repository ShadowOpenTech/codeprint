// Package loc counts code/comment/blank lines. gocloc provides comment-aware
// counting; for languages with no gocloc mapping we fall back to
// "non-blank = code" and mark the result not comment-aware.
package loc

import (
	"bytes"

	"github.com/hhatto/gocloc"
)

// Counts is the line breakdown for a file.
type Counts struct {
	Code         int
	Comment      int
	Blank        int
	CommentAware bool // false when no gocloc language mapping was available
}

// defined is gocloc's language table. It is read-only after construction, so a
// single shared instance is safe for concurrent reads.
var defined = gocloc.NewDefinedLanguages()

// Count returns the line breakdown for content. enryLang is the language ID
// from detection; it is mapped to a gocloc language for comment-aware counting.
// Unmapped languages fall back to non-blank=code with CommentAware=false.
func Count(path, enryLang string, content []byte) Counts {
	if lang, ok := defined.Langs[enryLang]; ok {
		cf := gocloc.AnalyzeReader(path, lang, bytes.NewReader(content), gocloc.NewClocOptions())
		if cf != nil {
			return Counts{
				Code:         int(cf.Code),
				Comment:      int(cf.Comments),
				Blank:        int(cf.Blanks),
				CommentAware: true,
			}
		}
	}
	return fallback(content)
}

// fallback counts blank vs non-blank lines without comment awareness.
func fallback(content []byte) Counts {
	if len(content) == 0 {
		return Counts{}
	}
	var code, blank int
	for _, line := range bytes.Split(content, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			blank++
		} else {
			code++
		}
	}
	// A trailing newline produces a final empty element; treat it as no line.
	if len(content) > 0 && content[len(content)-1] == '\n' && blank > 0 {
		blank--
	}
	return Counts{Code: code, Blank: blank, CommentAware: false}
}
