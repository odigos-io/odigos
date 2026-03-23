.PHONY: build-odiglet generate generate-obi

# Libraries that need generate run in module cache (must have a generate target that works without git).
LIBRARIES := go.opentelemetry.io/auto \
			github.com/odigos-io/runtime-detector

# OBI (go.opentelemetry.io/obi) does not ship generated eBPF files. Dockerfile uses the official
# obi-generator image (ghcr.io/open-telemetry/obi-generator); here we build bpf2go for local make.
OBI_MODULE := go.opentelemetry.io/obi
OBI_BPF_CFLAGS := -std=gnu17 -O2 -g -Wunaligned-access -Wpacked -Wpadded -Wall -Werror
TOOLS := $(PWD)/.tools
BPF2GO := $(TOOLS)/bpf2go

build-odiglet: generate
	CGO_ENABLED=0 go build -ldflags="$(LD_FLAGS)" -o odiglet cmd/main.go

debug-build-odiglet: generate
	go build -o odiglet -gcflags "all=-N -l" cmd/main.go

generate: generate-libs generate-obi

generate-libs:
	go mod download
	@for lib in $(LIBRARIES); do \
		LIB_PATH=$$(go list -m -f '{{.Dir}}' $${lib%@*}) && \
		chmod -R +w $$LIB_PATH && \
		make -C "$$LIB_PATH" generate; \
	done

# Build bpf2go from odiglet's cilium/ebpf (for local make; Dockerfile uses obi-generator image).
$(BPF2GO):
	@mkdir -p $(TOOLS) && go build -o $(BPF2GO) github.com/cilium/ebpf/cmd/bpf2go

# Generate eBPF Go bindings for OBI. Set SKIP_OBI_GENERATE=1 when Dockerfile already did it.
generate-obi:
	@if [ -n "$${SKIP_OBI_GENERATE}" ]; then echo "Skipping OBI generate (SKIP_OBI_GENERATE set)"; exit 0; fi
	go mod download
	@OBI_DIR=$$(go list -m -f '{{.Dir}}' $(OBI_MODULE)) && \
	chmod -R +w "$$OBI_DIR" && \
	$(MAKE) $(BPF2GO) && \
	(cd "$$OBI_DIR" && BPF2GO="$(abspath $(BPF2GO))" BPF_CLANG=clang BPF_CFLAGS="$(OBI_BPF_CFLAGS)" go generate ./...)

test: generate
	go test -v ./...

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
