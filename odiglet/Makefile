.PHONY: build-odiglet generate

LIBRARIES := go.opentelemetry.io/auto \
			github.com/odigos-io/runtime-detector

BUILD_TAGS ?=
GO_BUILD_TAGS := $(if $(strip $(BUILD_TAGS)),-tags $(BUILD_TAGS),)

build-odiglet: generate
	CGO_ENABLED=0 go build $(GO_BUILD_TAGS) -ldflags="$(LD_FLAGS)" -o odiglet cmd/main.go

debug-build-odiglet: generate
	go build $(GO_BUILD_TAGS) -o odiglet -gcflags "all=-N -l" cmd/main.go

generate:
	go mod download
	@for lib in $(LIBRARIES); do \
		LIB_PATH=$$(go list -m -f '{{.Dir}}' $${lib%@*}) && \
		chmod -R +w $$LIB_PATH && \
		make -C "$$LIB_PATH" generate; \
	done

test: generate
	go test -v ./...

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
