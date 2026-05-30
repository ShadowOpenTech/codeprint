# Releasing codeprint

The build is **release-ready**. The actual public tag + publish is intentionally
left to you (irreversible; JFrog needs credentials I don't have). This is the
exact procedure.

## Prerequisites

Done: LICENSE (Apache-2.0) is committed; goreleaser archives include it.
Distribution is **GitHub Releases + the Go module proxy** (`go install`) — both
free and public. No artifact registry or secrets to configure; the release uses
the built-in `GITHUB_TOKEN`.

## Cut Release 0 (v0.1.0)

```bash
# 1. Be on an up-to-date, green main.
git checkout main && git pull

# 2. Sanity: the full gate must pass locally.
make verify

# 3. Tag and push — this fires .github/workflows/release.yml (goreleaser).
git tag v0.1.0
git push origin v0.1.0
```

The release workflow then:
- re-checks schema drift,
- cross-builds the 5 platforms (`CGO_ENABLED=0`, stripped, version stamped),
- publishes archives + `checksums.txt` to **GitHub Releases**.

## Verify Release 0

- [ ] GitHub Release `v0.1.0` has 5 archives + checksums.
- [ ] `go install github.com/ShadowOpenTech/codeprint/cmd/codeprint@v0.1.0` works.
- [ ] `codeprint --version` reports `0.1.0`.

## Versioning

- v0.x is pre-1.0: the schema may change between releases (warn in changelog).
- Conventional Commits drive the changelog. A `feat!:`/`BREAKING CHANGE:` implies a major bump.
- Lock the schema contract at **v1.0.0** (immutable `$schema` URL, schemastore registration).
