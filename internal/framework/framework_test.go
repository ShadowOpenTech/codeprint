package framework

import (
	"sort"
	"testing"
)

func names(hints []Hint) []string {
	var n []string
	for _, h := range hints {
		n = append(n, h.Ecosystem+"/"+h.Name)
	}
	sort.Strings(n)
	return n
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNpm(t *testing.T) {
	c := []byte(`{"dependencies":{"react":"^18","express":"^4","lodash":"^4"},"devDependencies":{"@nestjs/core":"^10"}}`)
	got := names(npmHints("package.json", c))
	want := []string{"npm/express", "npm/nest", "npm/react"}
	if !eq(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNpmMalformed(t *testing.T) {
	if h := npmHints("package.json", []byte("{not json")); h != nil {
		t.Errorf("malformed json should yield nil, got %v", h)
	}
}

func TestRequirements(t *testing.T) {
	c := []byte("Django==4.2\n# comment\nFlask>=2\nrequests\nfastapi~=0.1\n")
	got := names(requirementsHints("requirements.txt", c))
	want := []string{"pip/django", "pip/fastapi", "pip/flask"}
	if !eq(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestPyproject(t *testing.T) {
	pep621 := []byte("[project]\ndependencies = [\"Django>=4\", \"requests\"]\n")
	if !eq(names(pyprojectHints("pyproject.toml", pep621)), []string{"pip/django"}) {
		t.Error("PEP 621 dependencies not parsed")
	}
	poetry := []byte("[tool.poetry.dependencies]\nflask = \"^2\"\npython = \"^3.11\"\n")
	if !eq(names(pyprojectHints("pyproject.toml", poetry)), []string{"pip/flask"}) {
		t.Error("poetry dependencies not parsed")
	}
}

func TestMaven(t *testing.T) {
	c := []byte(`<project><dependencies>
	<dependency><groupId>org.springframework.boot</groupId><artifactId>spring-boot-starter-web</artifactId></dependency>
	<dependency><groupId>junit</groupId><artifactId>junit</artifactId></dependency>
	</dependencies></project>`)
	if !eq(names(mavenHints("pom.xml", c)), []string{"maven/spring-boot"}) {
		t.Errorf("maven spring-boot not detected: %v", names(mavenHints("pom.xml", c)))
	}
}

func TestGradle(t *testing.T) {
	c := []byte(`dependencies {
	implementation "org.springframework.boot:spring-boot-starter:3.0"
	implementation 'io.micronaut:micronaut-http:4.0'
	}
	apply plugin: 'com.android.application'`)
	got := names(gradleHints("build.gradle", c))
	want := []string{"gradle/android", "gradle/micronaut", "gradle/spring-boot"}
	if !eq(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBundler(t *testing.T) {
	c := []byte("source 'https://rubygems.org'\ngem \"rails\", \"~> 7\"\ngem 'puma'\n")
	if !eq(names(bundlerHints("Gemfile", c)), []string{"bundler/rails"}) {
		t.Error("rails not detected")
	}
}

func TestCargo(t *testing.T) {
	c := []byte("[dependencies]\nactix-web = \"4\"\nserde = \"1\"\n")
	if !eq(names(cargoHints("Cargo.toml", c)), []string{"cargo/actix"}) {
		t.Error("actix not detected")
	}
}

func TestComposer(t *testing.T) {
	c := []byte(`{"require":{"laravel/framework":"^10","php":">=8"},"require-dev":{"symfony/console":"^6"}}`)
	got := names(composerHints("composer.json", c))
	want := []string{"composer/laravel", "composer/symfony"}
	if !eq(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestConfidenceAndEvidence(t *testing.T) {
	h := npmHints("a/package.json", []byte(`{"dependencies":{"react":"^18"}}`))
	if len(h) != 1 || h[0].Confidence != confHigh || h[0].Path != "a/package.json" || h[0].Keyword != "react" {
		t.Errorf("evidence/confidence wrong: %+v", h)
	}
}
