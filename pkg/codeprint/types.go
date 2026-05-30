// Package codeprint produces a stable, versioned JSON fingerprint of a codebase:
// languages, LOC, file classification, build systems, and framework hints.
//
// The types in this file are the single source of truth for the published JSON
// schema (schema/codeprint-v1.schema.json), generated from these struct tags.
//
// # Determinism
//
// The Fingerprint payload is byte-identical for identical input: all lists are
// sorted by a documented stable key and no volatile data appears in it. Volatile
// data (timestamps, producer version) lives only in the Meta envelope, which
// consumers must ignore when diffing.
package codeprint

// SchemaVersion is the current schema major.minor carried in Meta.
const SchemaVersion = "1.0"

// Output is the top-level envelope written to stdout or --out.
//
// The envelope shape is fixed: only Fingerprint is subject to the determinism
// rules; Meta is intentionally volatile.
type Output struct {
	Schema      string      `json:"$schema,omitempty" jsonschema:"description=URL of the JSON schema this document conforms to"`
	Meta        Meta        `json:"_meta" jsonschema:"description=Volatile envelope metadata; ignore when diffing fingerprints"`
	Fingerprint Fingerprint `json:"fingerprint" jsonschema:"description=The deterministic codebase fingerprint"`
}

// Meta is the volatile envelope. Nothing here participates in determinism.
type Meta struct {
	SchemaVersion string   `json:"schema_version" jsonschema:"description=Schema version (major.minor)"`
	Producer      Producer `json:"producer" jsonschema:"description=Tool that produced this fingerprint"`
	GeneratedAt   string   `json:"generated_at,omitempty" jsonschema:"description=RFC3339 production time; the only timestamp, volatile"`
}

// Producer identifies the binary that produced a fingerprint, for provenance.
type Producer struct {
	Name    string `json:"name" jsonschema:"description=Producer name (codeprint)"`
	Version string `json:"version" jsonschema:"description=Producer semantic version"`
}

// Fingerprint is the deterministic payload describing a codebase.
type Fingerprint struct {
	Totals         Totals           `json:"totals" jsonschema:"description=Repository-wide totals"`
	Languages      []LanguageRollup `json:"languages" jsonschema:"description=Per-language rollups, sorted by language ID"`
	Files          []FileRecord     `json:"files" jsonschema:"description=Per-file records, sorted by path; omitted when WithoutFileRecords"`
	BuildSystems   []BuildSystem    `json:"build_systems" jsonschema:"description=Detected build systems, sorted by path"`
	FrameworkHints []FrameworkHint  `json:"framework_hints" jsonschema:"description=Framework hints, sorted by (ecosystem,name); always a list"`
	Container      Container        `json:"container" jsonschema:"description=Container-related signals"`
	Symlinks       SymlinkReport    `json:"symlinks" jsonschema:"description=Symlink traversal summary; surfaces completeness gaps"`
	Errors         []ScanError      `json:"errors" jsonschema:"description=Non-fatal per-file errors, sorted by path"`
	// Workspaces is reserved for v2 per-workspace fingerprints (monorepos).
	// It is always null in v1; the field name is locked so v2 adds population,
	// not a schema-shape change.
	Workspaces []Workspace `json:"workspaces" jsonschema:"description=Reserved for v2; always null in v1"`
}

// Totals are repository-wide aggregate counts.
type Totals struct {
	Files   int   `json:"files" jsonschema:"description=Total files seen (incl. binary/vendored/etc.)"`
	Code    int   `json:"code" jsonschema:"description=Total code lines"`
	Comment int   `json:"comment" jsonschema:"description=Total comment lines"`
	Blank   int   `json:"blank" jsonschema:"description=Total blank lines"`
	Bytes   int64 `json:"bytes" jsonschema:"description=Total bytes"`
}

// LanguageRollup aggregates counts for one language across the repo.
type LanguageRollup struct {
	Language     string  `json:"language" jsonschema:"description=Language ID (go-enry/linguist)"`
	Files        int     `json:"files" jsonschema:"description=Files of this language"`
	Code         int     `json:"code" jsonschema:"description=Code lines"`
	Comment      int     `json:"comment" jsonschema:"description=Comment lines"`
	Blank        int     `json:"blank" jsonschema:"description=Blank lines"`
	Percent      float64 `json:"percent" jsonschema:"description=Share of total code LOC, rounded to 2 decimals"`
	CommentAware bool    `json:"comment_aware" jsonschema:"description=False when no comment-syntax mapping exists (counts fall back to non-blank=code)"`
}

// Kind is the single primary classification of a file.
type Kind string

// File classification kinds, in F-2 priority order.
const (
	KindSource    Kind = "source"
	KindTest      Kind = "test"
	KindGenerated Kind = "generated"
	KindVendored  Kind = "vendored"
	KindBinary    Kind = "binary"
	KindMinified  Kind = "minified"
)

// FileRecord describes a single file. Code/Comment/Blank are 0 for files
// excluded from LOC totals (binary, vendored, minified, oversized).
type FileRecord struct {
	Path     string   `json:"path" jsonschema:"description=Repo-relative path (never absolute)"`
	Language string   `json:"language" jsonschema:"description=Language ID; empty if undetected"`
	Kind     Kind     `json:"kind" jsonschema:"enum=source,enum=test,enum=generated,enum=vendored,enum=binary,enum=minified"`
	Flags    []string `json:"flags" jsonschema:"description=Non-exclusive properties, e.g. test, skipped_large; sorted"`
	Code     int      `json:"code" jsonschema:"description=Code lines"`
	Comment  int      `json:"comment" jsonschema:"description=Comment lines"`
	Blank    int      `json:"blank" jsonschema:"description=Blank lines"`
	Bytes    int64    `json:"bytes" jsonschema:"description=File size in bytes"`
}

// BuildSystem is a detected build/package ecosystem and the marker that triggered it.
type BuildSystem struct {
	Ecosystem string `json:"ecosystem" jsonschema:"description=Ecosystem ID, e.g. npm, go-modules, maven"`
	Path      string `json:"path" jsonschema:"description=Repo-relative path to the marker file"`
}

// Confidence is the certainty level of a framework hint.
type Confidence string

// Framework-hint confidence levels.
const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// FrameworkHint is a best-effort framework signal. Always emitted in a list,
// never as a singular framework.
type FrameworkHint struct {
	Name       string     `json:"name" jsonschema:"description=Framework name, e.g. django, react, spring-boot"`
	Ecosystem  string     `json:"ecosystem" jsonschema:"description=Owning ecosystem"`
	Confidence Confidence `json:"confidence" jsonschema:"enum=high,enum=medium,enum=low"`
	Evidence   Evidence   `json:"evidence" jsonschema:"description=What triggered the hint"`
}

// Evidence records where a framework hint came from.
type Evidence struct {
	Path    string `json:"path" jsonschema:"description=Repo-relative path to the manifest"`
	Keyword string `json:"keyword" jsonschema:"description=Trigger keyword found in the manifest"`
}

// Container holds container-related repository signals.
type Container struct {
	DockerfilePresent bool `json:"dockerfile_present" jsonschema:"description=True if a Dockerfile was found"`
}

// SymlinkReport summarizes how symlinks were handled during the scan. It makes
// traversal gaps explicit: skipped directories and escaping links mean some
// content was intentionally not walked. total = followed_file + len(skipped).
type SymlinkReport struct {
	Total        int              `json:"total" jsonschema:"description=Total symlinks encountered"`
	FollowedFile int              `json:"followed_file" jsonschema:"description=In-tree file symlinks that were followed and counted"`
	Skipped      []SkippedSymlink `json:"skipped" jsonschema:"description=Symlinks not traversed, sorted by path"`
}

// SkippedSymlink is a symlink that was not followed. The resolved target is
// never recorded (it may point outside the workspace).
type SkippedSymlink struct {
	Path   string `json:"path" jsonschema:"description=Repo-relative path of the symlink"`
	Reason string `json:"reason" jsonschema:"enum=directory,enum=escaping,enum=unresolvable"`
}

// ScanError is a non-fatal per-file failure. Its presence does not change the
// exit code. Reason is the OS error only, never file content.
type ScanError struct {
	Path   string `json:"path" jsonschema:"description=Repo-relative path that failed"`
	Reason string `json:"reason" jsonschema:"description=OS/internal error reason; never file content"`
}

// Workspace is the reserved v2 per-workspace shape. Intentionally empty in v1.
type Workspace struct{}
