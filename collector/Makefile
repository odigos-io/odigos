
SRC_ROOT := $(shell git rev-parse --show-toplevel)/collector

TOOLS_BIN_DIR    := $(SRC_ROOT)/.tools

# Define the tool version
BUILDER_VERSION = 0.148.0
BUILDER := $(TOOLS_BIN_DIR)/builder_$(BUILDER_VERSION)
X_TOOLS_VERSION ?= v0.43.0
GOVULNCHECK_VERSION ?= v1.1.4

# Determine the current operating system and architecture
CURRENT_OS := $(shell go env GOOS)
CURRENT_ARCH := $(shell go env GOHOSTARCH)
GOBIN := $(shell go env GOPATH)/bin

# Define the URL format for the tool executable
BUILDER_URL = https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv$(BUILDER_VERSION)/ocb_$(BUILDER_VERSION)_$(CURRENT_OS)_$(CURRENT_ARCH)

GOCMD?= go
MDATAGENCMD?= mdatagen

PROCESSOR_DIR := processors/odigossamplingprocessor
TAILSAMPLING_PROCESSOR_DIR := processors/odigostailsamplingprocessor

.PHONY: download_builder
download_builder:
	mkdir -p $(TOOLS_BIN_DIR)
	if [ ! -f "$(BUILDER)" ]; then \
		curl -o $(BUILDER) -L $(BUILDER_URL); \
		chmod +x $(BUILDER); \
	fi

.PHONY: genodigoscol
genodigoscol: download_builder
	$(BUILDER) --ldflags="-checklinkname=0" --config builder-config.yaml --skip-compilation

.PHONY: test-odigossamplingprocessor
test-odigossamplingprocessor:
	@echo "Running tests for odigossamplingprocessor at $(PROCESSOR_DIR)"
	cd $(PROCESSOR_DIR) && GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) $(GOCMD) test ./...

.PHONY: test-odigostailsamplingprocessor
test-odigostailsamplingprocessor:
	@echo "Running tests for odigostailsamplingprocessor at $(TAILSAMPLING_PROCESSOR_DIR)"
	cd $(TAILSAMPLING_PROCESSOR_DIR) && GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) $(GOCMD) test ./...

.PHONY: build-odigoscol
build-odigoscol: download_builder test-odigossamplingprocessor test-odigostailsamplingprocessor
	$(BUILDER) \
	--ldflags="-s -w -checklinkname=0" \
	--config builder-config.yaml \
	--skip-generate \
	--skip-get-modules

.PHONY: install-tools
install-tools:
	cd ./internal/tools && go install github.com/client9/misspell/cmd/misspell
	cd ./internal/tools && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	cd ./internal/tools && go install github.com/google/addlicense
	cd ./internal/tools && go install github.com/jstemmer/go-junit-report
	cd ./internal/tools && go install go.opentelemetry.io/collector/cmd/mdatagen
	cd ./internal/tools && go install github.com/ory/go-acc
	cd ./internal/tools && go install github.com/pavius/impi/cmd/impi
	cd ./internal/tools && go install github.com/tcnksm/ghr
	cd ./internal/tools && go install golang.org/x/tools/cmd/goimports@$(X_TOOLS_VERSION)
	cd ./internal/tools && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@$(X_TOOLS_VERSION)
	cd ./internal/tools && go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: generate
generate: install-tools
	# mdatagen resolves import root by running:
	#   git rev-parse --show-toplevel
	# and then reading go.mod at that path.
	#
	# This repository is multi-module and does not have a root go.mod, so we
	# scope each mdatagen invocation to the nearest module root by overriding
	# GIT_WORK_TREE (while keeping GIT_DIR pointing to the real repo .git).
	# This avoids creating any temporary go.mod files.
	@REPO_ROOT="$$(git rev-parse --show-toplevel)"; \
	find . -type f -name 'metadata.yaml' -print0 | while IFS= read -r -d '' META_FILE; do \
		META_DIR="$$(dirname "$$META_FILE")"; \
		MOD_ROOT="$$META_DIR"; \
		while [ "$$MOD_ROOT" != "." ] && [ "$$MOD_ROOT" != "/" ] && [ ! -f "$$MOD_ROOT/go.mod" ]; do \
			MOD_ROOT="$$(dirname "$$MOD_ROOT")"; \
		done; \
		if [ ! -f "$$MOD_ROOT/go.mod" ]; then \
			echo "No go.mod found for $$META_FILE"; \
			exit 1; \
		fi; \
		MOD_ROOT_ABS="$$(cd "$$MOD_ROOT" && pwd)"; \
		( cd "$$META_DIR" && GIT_DIR="$$REPO_ROOT/.git" GIT_WORK_TREE="$$MOD_ROOT_ABS" $(GOBIN)/mdatagen metadata.yaml ) || exit 1; \
	done
	$(MAKE) fmt

ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort | grep -E '^./' | grep -v "licenses")
.PHONY: fmt
fmt: $(ALL_GO_MOD_DIRS:%=fmt/%)
fmt/%: install-tools
	cd $* && goimports -w -local github.com/odigos-io ./

ALL_GO_MODS := $(shell find . -type f -name 'go.mod' ! -path './exporters/googlecloudstorageexporter/*' ! -path './exporters/azureblobstorageexporter/*' ! -path './internal/tools/*' ! -path './odigosotelcol/*' | sort | grep -v "LICENSES")
GO_MODS_TO_TEST := $(ALL_GO_MODS:%=test/%)

.PHONY: test
test: $(GO_MODS_TO_TEST)
test/%: GO_MOD=$*
test/%:
	cd $(shell dirname $(GO_MOD)) && $(GOCMD) test --ldflags="-checklinkname=0" -v ./...

.PHONY: go-licenses
go-licenses:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/google/go-licenses@latest

.PHONY: licenses
licenses: go-licenses
	# `github.com/segmentio/asm` is licensed under MIT-0 (MIT No Attribution),
	# which is a permissive open-source license. However, `go-licenses` does not automatically
	# detect its license file in subpackages, so we explicitly ignore it to prevent errors.
	# we're using the same pattern as in the frontend/Makefile
	@if [ -d "$(PWD)/odigosotelcol/licenses" ]; then \
		rm -rf $(PWD)/odigosotelcol/licenses; \
	fi
	cp LICENSE odigosotelcol/. && cd odigosotelcol && $(TOOLS_BIN_DIR)/go-licenses save . --save_path=licenses --ignore=github.com/segmentio/asm
