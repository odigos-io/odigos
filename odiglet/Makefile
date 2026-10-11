.PHONY: build-odiglet generate compile-odiglet debug-build-odiglet tools go-licenses licenses

include obi.mk

# Libraries with a generate target in the module cache
LIBRARIES := go.opentelemetry.io/auto \
			github.com/odigos-io/runtime-detector

go-mod: go-mod-replace
	@go mod download

# go-mod + each $(LIBRARIES) upstream `make generate` in parallel. Does not clone/materialize OBI.
generate: go-mod
	@pids=""; \
	for lib in $(LIBRARIES); do \
		( \
			LIB_PATH=$$(go list -m -f '{{.Dir}}' $${lib%@*}) && \
			chmod -R +w $$LIB_PATH && \
			$(MAKE) -C "$$LIB_PATH" generate \
		) & \
		pids="$$pids $$!"; \
	done; \
	fail=0; \
	for pid in $$pids; do \
		wait $$pid || fail=1; \
	done; \
	exit $$fail

# Compile only — no materialize / generate / go-mod. For Docker after those layers have run.
compile-odiglet:
	CGO_ENABLED=0 go build -ldflags="$(LD_FLAGS)" -o odiglet cmd/main.go

build-odiglet:
	$(MAKE) materialize-obi
	$(MAKE) generate
	$(MAKE) compile-odiglet

debug-build-odiglet:
	$(MAKE) materialize-obi
	$(MAKE) generate
	go build -o odiglet -gcflags "all=-N -l" cmd/main.go

test: materialize-obi generate
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
