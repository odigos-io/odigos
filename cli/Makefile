LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: go-licenses
go-licenses: $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/google/go-licenses@latest

.PHONY: licenses
licenses: go-licenses
	rm -rf $(PWD)/licenses
	$(LOCALBIN)/go-licenses save ./... --save_path=licenses
