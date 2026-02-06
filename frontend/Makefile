.PHONY: test
test:
	go test ./... -count=1

.PHONY: bench
bench:
	go test -bench=. -benchtime=3s -count=1 ./services/ ./graph/

gqlgen:
	go get github.com/99designs/gqlgen@v0.17.70 && go run github.com/99designs/gqlgen generate

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
	# `modernc.org/mathutil` is licensed under BSD-3-Clause,
	# which is a permissive open-source license. However, `go-licenses` does not automatically
	# detect its license file, so we explicitly ignore it to prevent errors.
	@if [ -d "$(PWD)/licenses" ]; then \
		rm -rf $(PWD)/licenses; \
	fi
	$(TOOLS)/go-licenses save . --save_path=licenses --ignore=modernc.org/mathutil
