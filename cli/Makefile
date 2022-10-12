.PHONY: generate-client
generate-client:
	go mod vendor
	./hack/update-codegen.sh
	cp -r ./vendor/github.com/keyval-dev/odigos/cli/pkg/generated ./pkg/
	rm -rf ./vendor
