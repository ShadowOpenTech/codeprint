package buildsys

import "testing"

func TestDetect(t *testing.T) {
	paths := []string{
		"go.mod",
		"frontend/package.json",
		"api/requirements.txt",
		"svc/pom.xml",
		"app/build.gradle.kts",
		"rust/Cargo.toml",
		"Gemfile",
		"win/App.csproj",
		"sln/Thing.sln",
		"php/composer.json",
		"ios/Package.swift",
		"docker/Dockerfile",
		"src/main.go", // not a marker
	}
	markers, dockerfile := Detect(paths)

	want := map[string]string{
		"go.mod":                "go-modules",
		"frontend/package.json": "npm",
		"api/requirements.txt":  "pip",
		"svc/pom.xml":           "maven",
		"app/build.gradle.kts":  "gradle",
		"rust/Cargo.toml":       "cargo",
		"Gemfile":               "bundler",
		"win/App.csproj":        "nuget",
		"sln/Thing.sln":         "nuget",
		"php/composer.json":     "composer",
		"ios/Package.swift":     "swift-pm",
	}
	got := map[string]string{}
	for _, m := range markers {
		got[m.Path] = m.Ecosystem
	}
	for path, eco := range want {
		if got[path] != eco {
			t.Errorf("%q: got %q want %q", path, got[path], eco)
		}
	}
	if len(got) != len(want) {
		t.Errorf("marker count: got %d want %d (%v)", len(got), len(want), got)
	}
	if !dockerfile {
		t.Error("Dockerfile should be detected")
	}
}

func TestDetectSortedByPath(t *testing.T) {
	markers, _ := Detect([]string{"z/go.mod", "a/Cargo.toml", "m/Gemfile"})
	for i := 1; i < len(markers); i++ {
		if markers[i-1].Path > markers[i].Path {
			t.Errorf("markers not sorted by path: %v", markers)
		}
	}
}

func TestDetectNoMarkers(t *testing.T) {
	markers, dockerfile := Detect([]string{"src/a.go", "README.md"})
	if len(markers) != 0 {
		t.Errorf("expected no markers, got %v", markers)
	}
	if dockerfile {
		t.Error("no Dockerfile expected")
	}
}
