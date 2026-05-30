# codeprint — developer task runner. Not shipped; wraps go commands so devs and
# CI run the same thing. See docs/planning/env-setup.md.

GO        ?= go
GOBIN     := $(shell $(GO) env GOPATH)/bin
BIN       := bin/codeprint
SCHEMA    := schema/codeprint-v1.schema.json
PKGS      := ./...
# Pipeline packages the coverage gate applies to (docs/planning/code-quality.md).
COVERPKGS := github.com/ShadowOpenTech/codeprint/pkg/codeprint,github.com/ShadowOpenTech/codeprint/internal/...
COVER_MIN ?= 85.0

.PHONY: all build test cover lint fmt vet vuln schema schema-check determinism bench tools clean verify

all: verify

build: ## build the static binary
	CGO_ENABLED=0 $(GO) build -o $(BIN) ./cmd/codeprint

test: ## run all tests with race detector
	$(GO) test -race $(PKGS)

cover: ## run tests and enforce the pipeline coverage gate
	# -count=1 forces a full run: cached results produce partial cross-package
	# -coverpkg profiles, making the gate flaky. Always recompute.
	$(GO) test -count=1 -covermode=atomic -coverpkg=$(COVERPKGS) -coverprofile=coverage.out $(PKGS)
	@total=$$($(GO) tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
	echo "pipeline coverage: $$total% (min $(COVER_MIN)%)"; \
	awk "BEGIN{exit !($$total >= $(COVER_MIN))}" || { echo "coverage below $(COVER_MIN)%"; exit 1; }

# SRCDIRS excludes testdata/ (intentionally-varied corpus files must not be formatted/linted).
SRCDIRS := cmd internal pkg

fmt: ## apply gofumpt
	$(GOBIN)/gofumpt -w $(SRCDIRS)

lint: ## check formatting + run golangci-lint
	@test -z "$$($(GOBIN)/gofumpt -l $(SRCDIRS))" || { echo "gofumpt: files need formatting:"; $(GOBIN)/gofumpt -l $(SRCDIRS); exit 1; }
	$(GOBIN)/golangci-lint run

vet: ## go vet
	$(GO) vet $(PKGS)

vuln: ## govulncheck
	$(GOBIN)/govulncheck $(PKGS)

schema: ## regenerate the committed JSON schema from types
	$(GO) run ./cmd/schemagen -out $(SCHEMA)

schema-check: ## fail if the committed schema has drifted
	$(GO) run ./cmd/schemagen -check $(SCHEMA)

determinism: build ## scan a path twice and diff the fingerprint (set DIR=...)
	@scripts/determinism.sh $(if $(DIR),$(DIR),.)

bench: ## throughput benchmark (NFR-1)
	$(GO) test -bench=. -benchmem -run=^$$ ./pkg/codeprint

tools: ## install pinned dev tools
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

clean:
	rm -rf bin coverage.out

# verify mirrors CI: a green local run means a green CI run.
verify: fmt vet lint schema-check test cover vuln
	@echo "verify: all gates passed"
