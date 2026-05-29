package main

import "testing"

func TestResolveFormatExplicit(t *testing.T) {
	for _, f := range []string{"json", "pretty", "summary"} {
		got, err := resolveFormat(options{format: f})
		if err != nil || got != f {
			t.Errorf("resolveFormat(%q) = %q, %v", f, got, err)
		}
	}
}

func TestResolveFormatInvalid(t *testing.T) {
	if _, err := resolveFormat(options{format: "xml"}); err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestResolveFormatAuto(t *testing.T) {
	// Under `go test`, stdout is not a TTY, so auto resolves to json.
	got, err := resolveFormat(options{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "json" {
		t.Errorf("auto format with non-TTY stdout = %q, want json", got)
	}
}

func TestBuildOptions(t *testing.T) {
	o := options{concurrency: 4, maxFileSize: 1024, noHidden: true, ignoreFile: ".x", noFiles: true}
	if got := len(buildOptions(o)); got != 5 {
		t.Errorf("expected 5 options, got %d", got)
	}
	if got := len(buildOptions(options{})); got != 0 {
		t.Errorf("expected 0 options for defaults, got %d", got)
	}
}

func TestWantColorRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if wantColor(options{}) {
		t.Error("NO_COLOR set should disable color")
	}
	if wantColor(options{noColor: true}) {
		t.Error("--no-color should disable color")
	}
}
