.PHONY: all
all: build

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

.PHONY: build
build: fmt vet ## Build the browser-proxy binary.
	go build -o bin/odigos-browser-proxy ./cmd

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: go-licenses
go-licenses: $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/google/go-licenses@latest

.PHONY: licenses
licenses: go-licenses
	rm -rf $(PWD)/licenses
	$(LOCALBIN)/go-licenses save . --save_path=licenses
