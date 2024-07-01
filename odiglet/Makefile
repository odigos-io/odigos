.PHONY: build-odiglet generate

build-odiglet: generate
	go build -o odiglet cmd/main.go

debug-build-odiglet: generate
	go build -o odiglet -gcflags "all=-N -l" cmd/main.go

generate:
	go mod download
	GO_AUTO_PATH=$$(go list -m -f '{{.Dir}}' "go.opentelemetry.io/auto") && \
	cd $$GO_AUTO_PATH && \
	make generate