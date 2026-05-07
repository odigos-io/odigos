.PHONY: build-odiglet generate setup-obi

# go.opentelemetry.io/obi (module path in go mod edit -replace; must match go.mod)
OBI_MODULE := go.opentelemetry.io/obi
# Upstream source-generated release (https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/pull/1378). Keep in sync with go.mod require go.opentelemetry.io/obi.
OBI_VERSION ?= v0.8.0
OBI_LOCAL_DIR := $(abspath $(CURDIR)/.obi)
OBI_EXTRACTED := $(OBI_LOCAL_DIR)/obi-$(OBI_VERSION)-source-generated
OBI_ARCHIVE := obi-$(OBI_VERSION)-source-generated.tar.gz
OBI_RELEASE_BASE := https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/releases/download

# Libraries with a generate target in the module cache
LIBRARIES := go.opentelemetry.io/auto \
			github.com/odigos-io/runtime-detector

# Download and verify OBI source-generated tarball, extract under .obi/, go mod replace.
# Phony: prerequisite of `generate` (so `build-odiglet` / `make generate` / `test` refresh OBI first).
setup-obi:
	@mkdir -p $(OBI_LOCAL_DIR)/dl
	@set -e; \
		download_dir="$(OBI_LOCAL_DIR)/dl"; \
		versioned_release_url="$(OBI_RELEASE_BASE)/$(OBI_VERSION)"; \
		tarball_name="$(OBI_ARCHIVE)"; \
		cd "$$download_dir"; \
		curl -fsSL "$$versioned_release_url/SHA256SUMS" -o SHA256SUMS; \
		curl -fsSL "$$versioned_release_url/$$tarball_name" -o "$$tarball_name"; \
		expected_sha256=$$(grep -F "$$tarball_name" SHA256SUMS | awk '{print $$1}'); \
		computed_sha256=$$(openssl dgst -sha256 "$$tarball_name" | awk '{print $$NF}'); \
		test "$$expected_sha256" = "$$computed_sha256" || (echo "OBI archive checksum mismatch" && exit 1); \
		rm -rf "$(OBI_EXTRACTED)"; \
		tar -xzf "$$tarball_name" -C $(OBI_LOCAL_DIR)
	@go mod edit -replace "$(OBI_MODULE)=$(OBI_EXTRACTED)"
	@go mod tidy

# OBI: `setup-obi` (source-generated tarball + go mod replace), not `go generate` on OBI here.
# Then `go mod download` and each $(LIBRARIES) upstream `make generate`.
generate: setup-obi
	go mod download
	@for lib in $(LIBRARIES); do \
		LIB_PATH=$$(go list -m -f '{{.Dir}}' $${lib%@*}) && \
		chmod -R +w $$LIB_PATH && \
		make -C "$$LIB_PATH" generate; \
	done

build-odiglet: generate
	CGO_ENABLED=0 go build -ldflags="$(LD_FLAGS)" -o odiglet cmd/main.go

debug-build-odiglet: generate
	go build -o odiglet -gcflags "all=-N -l" cmd/main.go

test: generate
	go test -a -v ./...

TOOLS := $(PWD)/.tools

.PHONY: tools
tools:
	@if [ ! -d "$(TOOLS)" ]; then \
		mkdir $(TOOLS); \
	fi

.PHONY: go-licenses
go-licenses: tools
	GOBIN=$(TOOLS) go install github.com/google/go-licenses@latest

.PHONY: licenses
licenses: go-licenses
	@if [ -d "$(PWD)/licenses" ]; then \
		rm -rf $(PWD)/licenses; \
	fi
	$(TOOLS)/go-licenses save . --save_path=licenses
