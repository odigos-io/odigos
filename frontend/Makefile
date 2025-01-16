gqlgen:
	 go get github.com/99designs/gqlgen@v0.17.49 && go run github.com/99designs/gqlgen generate

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
	@if [ -d "$(PWD)/LICENSES" ]; then \
		rm -rf $(PWD)/LICENSES; \
	fi
	$(TOOLS)/go-licenses save . --save_path=LICENSES
