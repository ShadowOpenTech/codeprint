// Package framework extracts framework hints from build manifests. Hints are
// always a list with confidence + evidence — never a singular framework (F-4).
//
// Parsing is best-effort: a malformed manifest yields no hint and is not a scan
// error (soft degradation, per error-strategy). Returns plain structs;
// pkg/codeprint converts them (avoids an import cycle).
package framework

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ShadowOpenTech/codeprint/internal/buildsys"
)

// Hint is a detected framework signal.
type Hint struct {
	Name       string
	Ecosystem  string
	Confidence string // "high" | "medium" | "low"
	Path       string // manifest path (evidence)
	Keyword    string // trigger keyword (evidence)
}

const confHigh = "high"

// Detect reads each marker's manifest and returns framework hints, sorted by
// (ecosystem, name, path) and de-duplicated.
func Detect(absRoot string, markers []buildsys.Marker) []Hint {
	var hints []Hint
	for _, m := range markers {
		// m.Path is a repo-relative marker discovered by our own contained walk
		// (symlink-escapes already excluded), joined under absRoot — not user input.
		content, err := os.ReadFile(filepath.Join(absRoot, m.Path)) //nolint:gosec // contained walk path joined under absRoot
		if err != nil {
			continue // soft: unreadable manifest yields no hint
		}
		base := filepath.Base(m.Path)
		hints = append(hints, parse(m.Ecosystem, base, m.Path, content)...)
	}
	return dedupeSort(hints)
}

func parse(ecosystem, base, path string, content []byte) []Hint {
	switch ecosystem {
	case "npm":
		return npmHints(path, content)
	case "pip":
		if base == "pyproject.toml" {
			return pyprojectHints(path, content)
		}
		return requirementsHints(path, content)
	case "maven":
		return mavenHints(path, content)
	case "gradle":
		return gradleHints(path, content)
	case "bundler":
		return bundlerHints(path, content)
	case "cargo":
		return cargoHints(path, content)
	case "composer":
		return composerHints(path, content)
	default:
		return nil
	}
}

// --- catalogs: dependency key (lowercased) -> framework name ---

var npmCatalog = map[string]string{
	"react": "react", "vue": "vue", "@angular/core": "angular",
	"next": "next", "nuxt": "nuxt", "svelte": "svelte",
	"express": "express", "fastify": "fastify", "@nestjs/core": "nest",
}

var pyCatalog = map[string]string{"django": "django", "flask": "flask", "fastapi": "fastapi"}

var cargoCatalog = map[string]string{"actix-web": "actix", "actix": "actix", "rocket": "rocket", "axum": "axum"}

func hint(ecosystem, name, path, keyword string) Hint {
	return Hint{Name: name, Ecosystem: ecosystem, Confidence: confHigh, Path: path, Keyword: keyword}
}

// --- npm: package.json ---

func npmHints(path string, content []byte) []Hint {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(content, &pkg) != nil {
		return nil
	}
	var out []Hint
	for _, deps := range []map[string]string{pkg.Dependencies, pkg.DevDependencies} {
		for dep := range deps {
			if name, ok := npmCatalog[strings.ToLower(dep)]; ok {
				out = append(out, hint("npm", name, path, dep))
			}
		}
	}
	return out
}

// --- pip: requirements.txt ---

var reqLineRe = regexp.MustCompile(`^([A-Za-z0-9._-]+)`)

func requirementsHints(path string, content []byte) []Hint {
	var out []Hint
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := reqLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if name, ok := pyCatalog[strings.ToLower(m[1])]; ok {
			out = append(out, hint("pip", name, path, m[1]))
		}
	}
	return out
}

// --- pip: pyproject.toml ---

func pyprojectHints(path string, content []byte) []Hint {
	var doc map[string]any
	if toml.Unmarshal(content, &doc) != nil {
		return nil
	}
	var out []Hint
	add := func(dep string) {
		key := regexp.MustCompile(`^[A-Za-z0-9._-]+`).FindString(strings.TrimSpace(dep))
		if name, ok := pyCatalog[strings.ToLower(key)]; ok {
			out = append(out, hint("pip", name, path, key))
		}
	}
	// PEP 621: [project].dependencies = ["Django>=4", ...]
	if proj, ok := doc["project"].(map[string]any); ok {
		if deps, ok := proj["dependencies"].([]any); ok {
			for _, d := range deps {
				if s, ok := d.(string); ok {
					add(s)
				}
			}
		}
	}
	// Poetry: [tool.poetry.dependencies] table keys.
	if tool, ok := doc["tool"].(map[string]any); ok {
		if poetry, ok := tool["poetry"].(map[string]any); ok {
			if deps, ok := poetry["dependencies"].(map[string]any); ok {
				for k := range deps {
					add(k)
				}
			}
		}
	}
	return out
}

// --- maven: pom.xml ---

func mavenHints(path string, content []byte) []Hint {
	var pom struct {
		Dependencies struct {
			Dependency []struct {
				GroupID    string `xml:"groupId"`
				ArtifactID string `xml:"artifactId"`
			} `xml:"dependency"`
		} `xml:"dependencies"`
	}
	if xml.Unmarshal(content, &pom) != nil {
		return nil
	}
	var out []Hint
	for _, d := range pom.Dependencies.Dependency {
		if h, ok := javaHint(d.GroupID, d.ArtifactID, "maven", path); ok {
			out = append(out, h)
		}
	}
	return out
}

// javaHint maps maven/gradle coordinates to a framework.
func javaHint(group, artifact, ecosystem, path string) (Hint, bool) {
	g, a := strings.ToLower(group), strings.ToLower(artifact)
	switch {
	case strings.Contains(a, "spring-boot"):
		return hint(ecosystem, "spring-boot", path, artifact), true
	case strings.Contains(g, "springframework"):
		return hint(ecosystem, "spring", path, group), true
	case strings.Contains(g, "micronaut"):
		return hint(ecosystem, "micronaut", path, group), true
	case strings.Contains(g, "quarkus"):
		return hint(ecosystem, "quarkus", path, group), true
	}
	return Hint{}, false
}

// --- gradle: build.gradle(.kts) — regex over content ---

var gradleDepRe = regexp.MustCompile(`["']([\w.\-]+):([\w.\-]+)(?::[\w.\-]+)?["']`)

func gradleHints(path string, content []byte) []Hint {
	var out []Hint
	for _, m := range gradleDepRe.FindAllStringSubmatch(string(content), -1) {
		if h, ok := javaHint(m[1], m[2], "gradle", path); ok {
			out = append(out, h)
		}
	}
	if regexp.MustCompile(`com\.android\.(application|library)`).Match(content) {
		out = append(out, hint("gradle", "android", path, "com.android"))
	}
	return out
}

// --- bundler: Gemfile ---

var gemRe = regexp.MustCompile(`(?m)^\s*gem\s+["']([\w-]+)["']`)

func bundlerHints(path string, content []byte) []Hint {
	var out []Hint
	for _, m := range gemRe.FindAllStringSubmatch(string(content), -1) {
		if strings.EqualFold(m[1], "rails") {
			out = append(out, hint("bundler", "rails", path, m[1]))
		}
	}
	return out
}

// --- cargo: Cargo.toml ---

func cargoHints(path string, content []byte) []Hint {
	var doc map[string]any
	if toml.Unmarshal(content, &doc) != nil {
		return nil
	}
	deps, ok := doc["dependencies"].(map[string]any)
	if !ok {
		return nil
	}
	var out []Hint
	for dep := range deps {
		if name, ok := cargoCatalog[strings.ToLower(dep)]; ok {
			out = append(out, hint("cargo", name, path, dep))
		}
	}
	return out
}

// --- composer: composer.json ---

func composerHints(path string, content []byte) []Hint {
	var doc struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if json.Unmarshal(content, &doc) != nil {
		return nil
	}
	var out []Hint
	for _, reqs := range []map[string]string{doc.Require, doc.RequireDev} {
		for pkg := range reqs {
			low := strings.ToLower(pkg)
			switch {
			case strings.HasPrefix(low, "laravel/"):
				out = append(out, hint("composer", "laravel", path, pkg))
			case strings.HasPrefix(low, "symfony/"):
				out = append(out, hint("composer", "symfony", path, pkg))
			}
		}
	}
	return out
}

// dedupeSort removes duplicate (name,ecosystem,path) hints and sorts by
// (ecosystem, name, path) for determinism.
func dedupeSort(hints []Hint) []Hint {
	seen := map[string]bool{}
	var out []Hint
	for _, h := range hints {
		key := h.Ecosystem + "\x00" + h.Name + "\x00" + h.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ecosystem != out[j].Ecosystem {
			return out[i].Ecosystem < out[j].Ecosystem
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}
