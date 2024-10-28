.PHONY: build-odiglet generate

LIBRARIES := go.opentelemetry.io/auto \
			github.com/odigos-io/runtime-detector

build-odiglet: generate
	CGO_ENABLED=0 go build -o odiglet cmd/main.go

debug-build-odiglet: generate
	go build -o odiglet -gcflags "all=-N -l" cmd/main.go

generate:
	go mod download
	@for lib in $(LIBRARIES); do \
		LIB_PATH=$$(go list -m -f '{{.Dir}}' $${lib%@*}) && \
		make -C "$$LIB_PATH" generate; \
	done

test: generate
	go test -v ./...
