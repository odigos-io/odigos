# OBI (go.opentelemetry.io/obi) materialization and go.mod replace.
# Kept out of the main Makefile so Docker can COPY only this file for the
# materialize layer (independent of build/generate target changes).
# Use via `make -f obi.mk <target>` or through the main Makefile (`include`).
#
# The go module cache tarball for OBI does NOT include the bpf2go-generated
# *_bpfel.go / *.o files, so we materialize them locally and add a local-path
# go.mod replace before `go build`. Two methods are supported; targets
# dispatch via OBI_SETUP (pin|release):
#   - materialize-obi: fetch OBI under .obi/ (+ bpf2go for pin). No go.mod.
#   - go-mod-replace:  go mod edit -replace only (no download).
# Keep the go.opentelemetry.io/obi version in go.mod (require) in sync with the
# selected method: the OBI_COMMIT pseudo-version for pin, or the OBI_VERSION
# tag for release. Pinned to the same OBI commit as OSS odigos.

.PHONY: materialize-obi materialize-obi-pin materialize-obi-release go-mod-replace

OBI_MODULE := go.opentelemetry.io/obi
OBI_SETUP ?= pin
OBI_LOCAL_DIR := $(abspath $(CURDIR)/.obi)

# --- pin method (clone commit + bpf2go codegen) ---
OBI_COMMIT ?= c76a93c8775c4c66dae4af24c66b9a0f409a2e49
OBI_GIT_URL := https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation
OBI_CLONE_DIR := $(OBI_LOCAL_DIR)/opentelemetry-ebpf-instrumentation-$(OBI_COMMIT)
# Relative path for the transient go.mod replace (portable in docker build / bind mounts).
OBI_PIN_REPLACE_PATH := ./.obi/opentelemetry-ebpf-instrumentation-$(OBI_COMMIT)

# --- release method (source-generated tarball) ---
OBI_VERSION ?= v0.10.0
OBI_EXTRACTED := $(OBI_LOCAL_DIR)/obi-$(OBI_VERSION)-source-generated
OBI_ARCHIVE := obi-$(OBI_VERSION)-source-generated.tar.gz
OBI_RELEASE_BASE := https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/releases/download
OBI_RELEASE_REPLACE_PATH := ./.obi/obi-$(OBI_VERSION)-source-generated

# Local-path replace used by go-mod (relative so Docker bind mounts / workdirs resolve).
ifeq ($(OBI_SETUP),release)
OBI_REPLACE_PATH := $(OBI_RELEASE_REPLACE_PATH)
else
OBI_REPLACE_PATH := $(OBI_PIN_REPLACE_PATH)
endif

# Materialize OBI under .obi/ with bpf objects ready (no go.mod). Dispatches via OBI_SETUP.
# Intended for Docker layer caching: depends only on OBI_COMMIT / OBI_VERSION, not go.mod.
materialize-obi: materialize-obi-$(OBI_SETUP)

# Clone the pinned OBI commit and run bpf2go codegen (generate/all in-tree when docker is unavailable -
# e.g. inside a docker build; docker-generate on the host). Does not touch go.mod.
materialize-obi-pin:
	@mkdir -p $(OBI_LOCAL_DIR)
	@if [ ! -d "$(OBI_CLONE_DIR)/.git" ]; then \
		git init -q "$(OBI_CLONE_DIR)"; \
		git -C "$(OBI_CLONE_DIR)" remote add origin "$(OBI_GIT_URL)"; \
	fi
	@git -C "$(OBI_CLONE_DIR)" fetch --depth 1 origin "$(OBI_COMMIT)"
	@git -C "$(OBI_CLONE_DIR)" checkout -q --detach FETCH_HEAD
	@if [ -f /.dockerenv ] || ! command -v docker >/dev/null 2>&1; then \
		echo "==> Generating OBI bpf2go artifacts in-tree (no docker)"; \
		$(MAKE) -C "$(OBI_CLONE_DIR)" generate/all; \
	else \
		echo "==> Generating OBI bpf2go artifacts via docker-generate"; \
		$(MAKE) -C "$(OBI_CLONE_DIR)" docker-generate; \
	fi

# Download and verify OBI's source-generated release tarball, extract under .obi/. Does not touch go.mod.
materialize-obi-release:
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

# Wire materialized OBI into go.mod (replace only — no download).
# Useful after COPY overwrites go.mod in Docker; modules are already in the cache.
go-mod-replace:
	@go mod edit -replace "$(OBI_MODULE)=$(OBI_REPLACE_PATH)"