
SRC_ROOT := $(shell git rev-parse --show-toplevel)/collector

TOOLS_BIN_DIR    := $(SRC_ROOT)/.tools

# Define the tool version
BUILDER_VERSION = 0.130.0
BUILDER := $(TOOLS_BIN_DIR)/builder_$(BUILDER_VERSION)

# Determine the current operating system and architecture
CURRENT_OS := $(shell go env GOOS)
CURRENT_ARCH := $(shell go env GOHOSTARCH)
GOBIN := $(shell go env GOPATH)/bin

# Define the URL format for the tool executable
BUILDER_URL = https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv$(BUILDER_VERSION)/ocb_$(BUILDER_VERSION)_$(CURRENT_OS)_$(CURRENT_ARCH)

GOCMD?= go
MDATAGENCMD?= mdatagen

PROCESSOR_DIR := processors/odigossamplingprocessor


.PHONY: download_builder
download_builder:
	mkdir -p $(TOOLS_BIN_DIR)
	if [ ! -f "$(BUILDER)" ]; then \
		curl -o $(BUILDER) -L $(BUILDER_URL); \
		chmod +x $(BUILDER); \
	fi

.PHONY: genodigoscol
genodigoscol: download_builder
	$(BUILDER) --ldflags="-checklinkname=0" --config builder-config.yaml

.PHONY: test-odigossamplingprocessor
test-odigossamplingprocessor:
	@echo "Running tests for odigossamplingprocessor at $(PROCESSOR_DIR)"
	cd $(PROCESSOR_DIR) && GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) $(GOCMD) test ./...

.PHONY: build-odigoscol
build-odigoscol: download_builder test-odigossamplingprocessor
	$(BUILDER) --ldflags="-checklinkname=0" --config builder-config.yaml --skip-generate --skip-get-modules

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
	cd ./internal/tools && go install golang.org/x/tools/cmd/goimports
	cd ./internal/tools && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	cd ./internal/tools && go install golang.org/x/vuln/cmd/govulncheck@master

.PHONY: generate
generate: install-tools
	@find . -type f -name 'metadata.yaml' -print0 | xargs -0 -I {} sh -c 'cd $$(dirname {}); $(GOBIN)/mdatagen metadata.yaml'
	$(MAKE) fmt

ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort | grep -E '^./' | grep -v "licenses")
.PHONY: fmt
fmt: $(ALL_GO_MOD_DIRS:%=fmt/%)
fmt/%: install-tools
	cd $* && $(GOBIN)/goimports -w -local github.com/odigos-io ./

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
	@if [ -d "$(PWD)/odigosotelcol/licenses" ]; then \
		rm -rf $(PWD)/odigosotelcol/licenses; \
	fi
	cp LICENSE odigosotelcol/. && cd odigosotelcol && $(TOOLS_BIN_DIR)/go-licenses save . --save_path=licenses
