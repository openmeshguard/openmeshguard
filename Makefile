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
	@set -e; \
	tests="$$( $(GO) test ./internal/output -list '^(TestReportSchemaFixtures|TestGeneratedScanOutputMatchesSchema|TestExternalScanOutputMatchesSchema)$$' )"; \
	echo "$$tests" | grep -q '^TestReportSchemaFixtures$$'; \
	echo "$$tests" | grep -q '^TestGeneratedScanOutputMatchesSchema$$'; \
	echo "$$tests" | grep -q '^TestExternalScanOutputMatchesSchema$$'
	$(GO) test ./internal/output -run '^(TestReportSchemaFixtures|TestGeneratedScanOutputMatchesSchema|TestExternalScanOutputMatchesSchema)$$' -count=1

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
	./test/e2e/kind-up.sh

e2e: build
	./test/e2e/run.sh

kind-down:
	./test/e2e/kind-down.sh
