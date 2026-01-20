GO ?= go

BINARY ?= codeagent
CMD_PKG := ./cmd/codeagent

TOOLS_BIN := $(CURDIR)/bin
TOOLCHAIN ?= go1.22.0
GOLANGCI_LINT_VERSION := v1.56.2
STATICCHECK_VERSION := v0.4.7

GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
STATICCHECK := $(TOOLS_BIN)/staticcheck

.PHONY: build test lint clean install

build:
	$(GO) build -o $(BINARY) $(CMD_PKG)

test:
	$(GO) test ./...

$(GOLANGCI_LINT):
	@mkdir -p $(TOOLS_BIN)
	GOTOOLCHAIN=$(TOOLCHAIN) GOBIN=$(TOOLS_BIN) $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(STATICCHECK):
	@mkdir -p $(TOOLS_BIN)
	GOTOOLCHAIN=$(TOOLCHAIN) GOBIN=$(TOOLS_BIN) $(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)

lint: $(GOLANGCI_LINT) $(STATICCHECK)
	GOTOOLCHAIN=$(TOOLCHAIN) $(GOLANGCI_LINT) run ./...
	GOTOOLCHAIN=$(TOOLCHAIN) $(STATICCHECK) ./...

clean:
	@python3 -c 'import glob, os; paths=["codeagent","codeagent.exe","codeagent-wrapper","codeagent-wrapper.exe","coverage.out","cover.out","coverage.html"]; paths += glob.glob("coverage*.out") + glob.glob("cover_*.out") + glob.glob("*.test"); [os.remove(p) for p in paths if os.path.exists(p)]'

install:
	$(GO) install $(CMD_PKG)
