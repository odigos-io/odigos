.PHONY: test mdatagen

test:
	AWS_EC2_METADATA_DISABLED=true go test ./...

mdatagen:
	GIT_DIR="$(shell git rev-parse --absolute-git-dir)" GIT_WORK_TREE="$(CURDIR)" mdatagen metadata.yaml
