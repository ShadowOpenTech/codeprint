#!/usr/bin/env bash
# Scan a path twice and diff the fingerprint payload (ignoring the volatile
# _meta envelope). Exits non-zero on any difference. See docs/architecture/data-model.md.
set -euo pipefail

DIR="${1:-.}"
BIN="${BIN:-bin/codeprint}"

if [ ! -x "$BIN" ]; then
  echo "determinism: $BIN not found; run 'make build' first" >&2
  exit 2
fi

a="$(mktemp)"; b="$(mktemp)"
trap 'rm -f "$a" "$b"' EXIT

# --format=json so output is the canonical machine form; strip _meta via the
# tool's own determinism (no jq dependency): compare the fingerprint object.
"$BIN" "$DIR" --format=json > "$a"
"$BIN" "$DIR" --format=json --concurrency=1 > "$b"

if diff -q <(grep -v '"generated_at"' "$a") <(grep -v '"generated_at"' "$b") >/dev/null; then
  echo "determinism: PASS (byte-identical across runs and concurrency levels)"
else
  echo "determinism: FAIL — output differs across runs" >&2
  diff <(grep -v '"generated_at"' "$a") <(grep -v '"generated_at"' "$b") | head -40 >&2
  exit 1
fi
