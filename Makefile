# === ./Makefile ===
# Build + Lint para DeskSlice (Go + golangci-lint + commentlint)

# --- Config base ---
GO      ?= go
OS      ?= $(shell $(GO) env GOOS)
ARCH    ?= $(shell $(GO) env GOARCH)
BIN_EXT :=
ifeq ($(OS),windows)
  BIN_EXT := .exe
endif

# --- Detección del directorio de binarios de Go ---
GOBIN ?= $(shell $(GO) env GOBIN)
ifeq ($(strip $(GOBIN)),)
  GO_BIN_DIR := $(shell $(GO) env GOPATH)/bin
else
  GO_BIN_DIR := $(GOBIN)
endif

GOLANGCI_LINT_VER ?= latest
LINT_TIMEOUT      ?= 5m
GOLANGCI_LINT_BIN := $(GO_BIN_DIR)/golangci-lint$(BIN_EXT)

export PATH := $(GO_BIN_DIR):$(PATH)
export GOCACHE := $(abspath ./.gocache)
export GOLANGCI_LINT_CACHE := $(abspath ./.golangci-cache)

# Build tags opcionales: make BUILD_TAGS=foo build
BUILD_TAGS ?=

GO_BUILD := GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -gcflags 'all=-e' -tags '$(BUILD_TAGS)'

DIST_DIR_BASE := ./dist
ifeq ($(OS),windows)
  ifeq ($(ARCH),amd64)
    DIST_DIR := $(DIST_DIR_BASE)/win64
  else
    DIST_DIR := $(DIST_DIR_BASE)/$(OS)_$(ARCH)
  endif
else ifeq ($(OS),linux)
  ifeq ($(ARCH),amd64)
    DIST_DIR := $(DIST_DIR_BASE)/linux_x86-64
  else
    DIST_DIR := $(DIST_DIR_BASE)/$(OS)_$(ARCH)
  endif
else
  DIST_DIR := $(DIST_DIR_BASE)/$(OS)_$(ARCH)
endif

APP_NAME := codex_remote
APP_ENTRY := ./cmd/$(APP_NAME)
APP_BIN := $(DIST_DIR)/$(APP_NAME)$(BIN_EXT)

LDFLAGS_PROD := -s -w
GCFLAGS_DEV  := all=-N -l

.PHONY: all build build-dev clean tidy fmt vet lint commentlint tools test run print doctor \
        build-win build-linux build-linux-arm64 build-matrix

all: build

build: fmt vet tidy commentlint lint $(APP_BIN)
	@echo "✅ Build OK: $(APP_BIN)"

build-dev: fmt vet tidy commentlint lint
	@echo "🧪 build-dev (race + debug) para $(OS)/$(ARCH)"
	@mkdir -p $(DIST_DIR)
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -race -gcflags '$(GCFLAGS_DEV)' -tags '$(BUILD_TAGS)' -o $(APP_BIN) $(APP_ENTRY)

tools:
	@if [ ! -x "$(GOLANGCI_LINT_BIN)" ]; then \
	  echo "⬇️  instalando golangci-lint@$(GOLANGCI_LINT_VER) en $(GO_BIN_DIR) ..."; \
	  $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VER); \
	else \
	  echo "✅ golangci-lint encontrado en $(GOLANGCI_LINT_BIN)"; \
	fi

fmt:
	@echo "🧹 gofmt -s"
	@find . -name "*.go" \
	  -not -path "./vendor/*" \
	  -not -path "./dist/*" \
	  -not -path "./_tmp/*" \
	  -not -path "./.gocache/*" \
	  -not -path "./.golangci-cache/*" \
	  -not -path "./data/*" \
	  -not -path "./**/*.pb.go" \
	  -print0 | xargs -0 gofmt -s -w
	@echo "🧹 go fmt ./..."
	@$(GO) fmt ./...

vet:
	@echo "🔍 go vet"
	@$(GO) vet ./...

tidy:
	@$(GO) mod tidy
	@echo "📦 go mod tidy OK"

commentlint:
	@echo "📝 commentlint"
	@$(GO) run ./third_party/commentlint ./...

lint: commentlint
	@echo "🔎 golangci-lint"
	@$(MAKE) tools >/dev/null
	@"$(GOLANGCI_LINT_BIN)" run --timeout $(LINT_TIMEOUT) ./...

test:
	@$(GO) test ./...

run: build
	@$(APP_BIN)

$(APP_BIN):
	@echo "🚀 build $(APP_NAME) para $(OS)/$(ARCH)"
	@mkdir -p $(DIST_DIR)
	@$(GO_BUILD) -ldflags '$(LDFLAGS_PROD)' -o $@ $(APP_ENTRY)

build-win:
	@$(MAKE) OS=windows ARCH=amd64 build

build-linux:
	@$(MAKE) OS=linux ARCH=amd64 build

build-linux-arm64:
	@$(MAKE) OS=linux ARCH=arm64 build

build-matrix: fmt vet tidy lint
	@$(MAKE) build-win
	@$(MAKE) build-linux
	@$(MAKE) build-linux-arm64
	@echo "🧳 build-matrix OK en ./dist/*"

clean:
	@echo "🧹 clean"
	@rm -rf ./dist ./.gocache ./.golangci-cache

print:
	@echo "GO=$(GO)"
	@echo "OS=$(OS)"
	@echo "ARCH=$(ARCH)"
	@echo "DIST_DIR=$(DIST_DIR)"
	@echo "APP_BIN=$(APP_BIN)"

doctor:
	@echo "GO_BIN_DIR=$(GO_BIN_DIR)"
	@echo "GOBIN=$$($(GO) env GOBIN)"
	@echo "GOPATH=$$($(GO) env GOPATH)"
	@echo "golangci-lint bin: $(GOLANGCI_LINT_BIN)"
	@ls -l "$(GOLANGCI_LINT_BIN)" 2>/dev/null || echo "❌ golangci-lint no instalado"
