.PHONY: build-odiglet generate generate-libs bpf2go-install

TOOLS := $(PWD)/.tools
BPF2GO := $(TOOLS)/bpf2go
# https://github.com/cilium/ebpf/cmd/bpf2go version (used by OBI)
BPF2GO_CILIUM_EBPF_VERSION ?= v0.21.0

# go.opentelemetry.io/obi (module path for OBI eBPF go generate; must match go.mod)
OBI_MODULE := go.opentelemetry.io/obi
# GODEBUG=goindex=0: see odiglet/docs/obi-bpf2go-module-cache.md
# go build -a: refresh compile after new sources under GOMODCACHE (persistent GOCACHE in Docker).
GOBUILD := CGO_ENABLED=0 go build -a

# Mutating a module under GOMODCACHE after go mod download (bpf2go .go) can desync the
# module file index; disable it so the loader uses go/build.ImportDir (merge with $GODEBUG if set).
COMMA := ,
export GODEBUG := goindex=0$(if $(GODEBUG),$(COMMA)$(GODEBUG),)

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

generate-libs: bpf2go-install
	go mod download
	@for lib in $(LIBRARIES); do \
		LIB_PATH=$$(go list -m -f '{{.Dir}}' $${lib%@*}) && \
		chmod -R +w $$LIB_PATH && \
		make -C "$$LIB_PATH" generate; \
	done
	@OBI_DIR=$$(go list -m -f '{{.Dir}}' $(OBI_MODULE)) && \
	chmod -R u+w "$$OBI_DIR" && \
	cd "$$OBI_DIR" && \
	  export BPF2GO_MAKEBASE="$$OBI_DIR" && \
	  env BPF2GO="$(abspath $(BPF2GO))" \
	      BPF_CLANG="$(OBI_CLANG)" \
	      BPF_CFLAGS="$(OBI_CFLAGS)" \
	      go generate ./...

bpf2go-install: $(BPF2GO)

$(BPF2GO):
	@go mod download
	@mkdir -p $(TOOLS)
	@GOBIN=$(abspath $(TOOLS)) go install github.com/cilium/ebpf/cmd/bpf2go@$(BPF2GO_CILIUM_EBPF_VERSION)

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
