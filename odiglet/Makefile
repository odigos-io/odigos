.PHONY: build-odiglet generate generate-libs bpf2go-install

TOOLS := $(PWD)/.tools
BPF2GO := $(TOOLS)/bpf2go
# https://github.com/cilium/ebpf/cmd/bpf2go version (used by OBI)
BPF2GO_CILIUM_EBPF_VERSION ?= v0.21.0

# go.opentelemetry.io/obi (module path for OBI eBPF go generate; must match go.mod)
OBI_MODULE := go.opentelemetry.io/obi
# Docker: OBI bpf2go runs in ghcr.io/open-telemetry/obi-generator; overlay copies into $GOMODCACHE (see Dockerfile).
SKIP_OBI_GENERATE ?=
GOBUILD := CGO_ENABLED=0 go build

# OBI Makefile: CLANG, CFLAGS, and go generate for bpf2go
# https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/blob/main/Makefile
OBI_CLANG ?= clang
OBI_CFLAGS := -std=gnu17 -O2 -g -Wunaligned-access -Wpacked -Wpadded -Wall -Werror

# Libraries with a generate target in the module cache
LIBRARIES := go.opentelemetry.io/auto \
			github.com/odigos-io/runtime-detector

build-odiglet: generate
	$(GOBUILD) -ldflags="$(LD_FLAGS)" -o odiglet cmd/main.go

debug-build-odiglet: generate
	CGO_ENABLED=0 go build -a -o odiglet -gcflags "all=-N -l" cmd/main.go

generate: generate-libs

# Populate $GOMODCACHE with generated bpf2go / eBPF artifacts needed to compile odiglet:
#  1) bpf2go-install — put bpf2go in .tools/ (clears GOOS/GOARCH when cross-building).
#  2) go mod download — materialize dependency modules under $GOMODCACHE.
#  3) For each $(LIBRARIES), chmod +w that module dir and run its upstream Makefile generate.
#  4) Unless SKIP_OBI_GENERATE: resolve $(OBI_MODULE) in the cache and run go generate ./... (bpf2go).
#     With SKIP_OBI_GENERATE=1 (Dockerfile): OBI outputs are copied in from obi-generator; skip step 4.
generate-libs: bpf2go-install
	go mod download
	@for lib in $(LIBRARIES); do \
		LIB_PATH=$$(go list -m -f '{{.Dir}}' $${lib%@*}) && \
		chmod -R +w $$LIB_PATH && \
		make -C "$$LIB_PATH" generate; \
	done
ifeq ($(strip $(SKIP_OBI_GENERATE)),)
	@OBI_DIR=$$(go list -m -f '{{.Dir}}' $(OBI_MODULE)) && \
	chmod -R u+w "$$OBI_DIR" && \
	cd "$$OBI_DIR" && \
	  export BPF2GO_MAKEBASE="$$OBI_DIR" && \
	  env BPF2GO="$(abspath $(BPF2GO))" \
	      BPF_CLANG="$(OBI_CLANG)" \
	      BPF_CFLAGS="$(OBI_CFLAGS)" \
	      go generate ./...
endif

bpf2go-install: $(BPF2GO)

$(BPF2GO):
	@go mod download
	@mkdir -p $(TOOLS)
	@env -u GOOS -u GOARCH GOBIN=$(abspath $(TOOLS)) go install github.com/cilium/ebpf/cmd/bpf2go@$(BPF2GO_CILIUM_EBPF_VERSION)

test: generate
	CGO_ENABLED=0 go test -a -v ./...

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
