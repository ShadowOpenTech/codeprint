// Package buildsys detects build/package ecosystems by marker-file presence.
// Detection is a flat full-tree scan: every marker is reported with its path,
// not grouped into workspaces (per-workspace grouping is v2). No parsing here.
//
// Returns plain structs; pkg/codeprint converts them (avoids an import cycle).
package buildsys

import (
	"path/filepath"
	"sort"
	"strings"
)

// Marker is a detected ecosystem and the marker file that triggered it.
type Marker struct {
	Ecosystem string
	Path      string
}

// exactMarkers maps an exact basename to an ecosystem.
var exactMarkers = map[string]string{
	"package.json":     "npm",
	"requirements.txt": "pip",
	"pyproject.toml":   "pip",
	"Pipfile":          "pip",
	"pom.xml":          "maven",
	"build.gradle":     "gradle",
	"build.gradle.kts": "gradle",
	"settings.gradle":  "gradle",
	"go.mod":           "go-modules",
	"Gemfile":          "bundler",
	"packages.config":  "nuget",
	"Cargo.toml":       "cargo",
	"composer.json":    "composer",
	"Package.swift":    "swift-pm",
}

// suffixMarkers maps a filename suffix to an ecosystem (e.g. *.csproj).
var suffixMarkers = []struct{ suffix, ecosystem string }{
	{".csproj", "nuget"},
	{".sln", "nuget"},
	{"settings.gradle.kts", "gradle"},
}

// Detect scans repo-relative paths and returns detected markers (sorted by
// path) and whether a Dockerfile is present.
func Detect(paths []string) ([]Marker, bool) {
	var markers []Marker
	dockerfile := false

	for _, p := range paths {
		base := filepath.Base(p)
		if eco, ok := exactMarkers[base]; ok {
			markers = append(markers, Marker{Ecosystem: eco, Path: p})
			continue
		}
		for _, sm := range suffixMarkers {
			if strings.HasSuffix(base, sm.suffix) {
				markers = append(markers, Marker{Ecosystem: sm.ecosystem, Path: p})
				break
			}
		}
		if base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.") {
			dockerfile = true
		}
	}

	sort.Slice(markers, func(i, j int) bool {
		if markers[i].Path != markers[j].Path {
			return markers[i].Path < markers[j].Path
		}
		return markers[i].Ecosystem < markers[j].Ecosystem
	})
	return markers, dockerfile
}
