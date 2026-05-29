package loc

import "testing"

func TestCountCommentAwareGo(t *testing.T) {
	src := []byte("package main\n\n// a comment\nfunc main() {}\n")
	c := Count("x.go", "Go", src)
	if !c.CommentAware {
		t.Fatal("Go should be comment-aware")
	}
	if c.Comment != 1 {
		t.Errorf("comment = %d, want 1", c.Comment)
	}
	if c.Blank != 1 {
		t.Errorf("blank = %d, want 1", c.Blank)
	}
	if c.Code < 2 {
		t.Errorf("code = %d, want >= 2", c.Code)
	}
}

func TestCountFallbackUnknownLanguage(t *testing.T) {
	src := []byte("line one\n\nline three\n")
	c := Count("x.unknownext", "TotallyMadeUpLang", src)
	if c.CommentAware {
		t.Error("unknown language should not be comment-aware")
	}
	if c.Code != 2 {
		t.Errorf("code = %d, want 2", c.Code)
	}
	if c.Blank != 1 {
		t.Errorf("blank = %d, want 1", c.Blank)
	}
}

func TestCountFallbackEmpty(t *testing.T) {
	c := Count("x", "Nope", nil)
	if c.Code != 0 || c.Blank != 0 {
		t.Errorf("empty content should yield zero counts, got %+v", c)
	}
}
