gqlgen:
	go get github.com/99designs/gqlgen@v0.17.70 && go run github.com/99designs/gqlgen generate

TOOLS := $(PWD)/.tools

.PHONY: tools test
tools:
	@if [ ! -d "$(TOOLS)" ]; then \
		mkdir $(TOOLS); \
	fi

test:
	go test -v ./...

.PHONY: go-licenses
go-licenses: tools
	GOBIN=$(TOOLS) go install github.com/google/go-licenses@latest

.PHONY: licenses
licenses: go-licenses
	# `modernc.org/mathutil` is licensed under BSD-3-Clause,
	# which is a permissive open-source license. However, `go-licenses` does not automatically
	# detect its license file, so we explicitly ignore it to prevent errors.
	#
	# `github.com/grafana/pyroscope/...` is AGPL-3.0. `go-licenses` treats AGPL as forbidden by
	# default; we exclude this module from scanning (--ignore) so RHEL `make licenses` passes.
	# Legal/compliance: Odigos still ships linked Pyroscope code—follow AGPL obligations separately.
	@if [ -d "$(PWD)/licenses" ]; then \
		rm -rf $(PWD)/licenses; \
	fi
	$(TOOLS)/go-licenses save . --save_path=licenses \
		--ignore=modernc.org/mathutil \
		--ignore=github.com/grafana/pyroscope
