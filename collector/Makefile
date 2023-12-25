
SRC_ROOT := $(shell git rev-parse --show-toplevel)/collector

TOOLS_BIN_DIR    := $(SRC_ROOT)/.tools

BUILDER := $(TOOLS_BIN_DIR)/builder

# Determine the current operating system and architecture
CURRENT_OS := $(shell go env GOOS)
CURRENT_ARCH := $(shell go env GOARCH)

# Define the tool version
BUILDER_VERSION = 0.91.0

# Define the URL format for the tool executable
BUILDER_URL = https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd%2Fbuilder%2Fv$(BUILDER_VERSION)/ocb_$(BUILDER_VERSION)_$(CURRENT_OS)_$(CURRENT_ARCH)

.PHONY: download_builder
download_builder:
	mkdir -p $(TOOLS_BIN_DIR)
	curl -o $(BUILDER) -L $(BUILDER_URL)
	chmod +x $(BUILDER)

.PHONY: genodigoscol
genodigoscol: download_builder
	$(BUILDER) --config builder-config.yaml
