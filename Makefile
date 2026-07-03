GO ?= go
BINARY ?= bin/openmeshguard
GOLANGCI_LINT_VERSION ?= v2.5.0
GOLANGCI_LINT ?= $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: build test lint schema-test kind-up e2e kind-down fmt-check resolver-purity

build:
	@mkdir -p bin
	$(GO) build -o $(BINARY) ./cmd/openmeshguard

test:
	$(GO) test ./...

lint: fmt-check
	$(GOLANGCI_LINT) run

schema-test:
	@$(GO) test ./internal/output -list '^TestReportSchemaFixtures$$' | grep -q '^TestReportSchemaFixtures$$'
	$(GO) test ./internal/output -run '^TestReportSchemaFixtures$$' -count=1

fmt-check:
	@files="$$(git ls-files '*.go')"; \
	if [ -n "$$files" ]; then \
		unformatted="$$(gofmt -l $$files)"; \
	else \
		unformatted=""; \
	fi; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

resolver-purity:
	$(GOLANGCI_LINT) run --enable-only depguard ./internal/resolver/...

kind-up:
	@echo "kind-up is deferred until M4"

e2e:
	@echo "e2e is deferred until M4"

kind-down:
	@echo "kind-down is deferred until M4"
