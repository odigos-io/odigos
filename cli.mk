# CLI-related targets. Included from the root Makefile.
# Shared vars from the parent Makefile:
#   TAG                 - image / version tag (auto-detected from cluster/cli/helm if unset)
#   ORG                 - registry org (default: docker.io/keyval); STAGING_ORG=true for staging GCR.
#                         Local deploy/load-to-kind auto-select registry.odigos.io + enterprise image
#                         names when the cluster is an enterprise install (see scripts/resolve-dev-image.sh).
#   IMG_SUFFIX          - image name suffix (empty by default; -rhel-certified when RHEL=true)
#   ODIGOS_CLI_VERSION  - sets Helm image.tag for install/upgrade (default: `odigos version --cli`)
#   CLUSTER_NAME        - cluster name for install (default: local-dev-cluster)
#   CENTRAL_BACKEND_URL - optional central backend URL for install
#   CLI_CHART_VERSION   - optional --chart-version for cli-install/upgrade (published chart; not local)
#   FLAGS               - extra CLI flags appended to cli-install

CLI_IMAGE ?= $(ORG)/odigos-cli$(IMG_SUFFIX):$(TAG)
CLI_SUMMARY ?= Odigos CLI
CLI_DESCRIPTION ?= Odigos CLI to install and manage Odigos in your Kubernetes cluster.
SHORT_COMMIT ?= $(shell git rev-parse --short HEAD)
DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# RHEL-certified CLI (ko + Dockerfile.rhel-base); matches .github/workflows/publish-modules-rhel publish-cli-rhel
CLI_RHEL_IMAGE_NAME ?= odigos-cli-rhel-certified
CLI_RHEL_KO_BASE_IMAGE ?= $(ORG)/odigos-cli-rhel-ko-base:$(TAG)

# Regenerate Mintlify CLI docs snippets under docs/snippets/shared/cli from cobra commands.
.PHONY: cli-docs
cli-docs:
	rm -rf docs/snippets/shared/cli/*
	cd scripts/cli-docgen && KUBECONFIG=KUBECONFIG go run -tags embed_manifests main.go
	for file in docs/snippets/shared/cli/*.md; do \
		mv $${file} $${file%.md}.mdx; \
	done

# Install Odigos from local cli/ source (go run), reflecting local api/cli changes.
#
# Chart vs image tag:
#   - Default chart is the local path ../helm/odigos (works without a baked/embedded chart).
#   - ODIGOS_CLI_VERSION maps to --set image.tag=... (a real published tag such as v1.32.2).
#     Do not pass it as --chart-version: local charts are often version 0.0.0, and that flag
#     also skips the embedded-chart path used by release builds.
#   - CLI_CHART_VERSION optionally sets --chart-version and fetches that chart from the Helm
#     repo instead of using the local chart. Distinct from CHART_VERSION used by build-cli-image.
#
# Examples:
#   make cli-install ODIGOS_CLI_VERSION=v1.32.2
#   make cli-install ODIGOS_CLI_VERSION=v1.32.2 CLUSTER_NAME=my-cluster
#   make cli-install CLI_CHART_VERSION=1.32.2 ODIGOS_CLI_VERSION=v1.32.2
#   make cli-install FLAGS='--set image.tag=v1.32.2'
.PHONY: cli-install
cli-install:
	@echo "Installing odigos from source. image.tag: $(or $(ODIGOS_CLI_VERSION),<chart appVersion>)"
	cd ./cli ; go run -tags=embed_manifests . install \
		$(if $(CLI_CHART_VERSION),--chart-version $(CLI_CHART_VERSION),--chart ../helm/odigos) \
		$(if $(ODIGOS_CLI_VERSION),--set image.tag=$(ODIGOS_CLI_VERSION)) \
		$(if $(CLUSTER_NAME),--set clusterName=$(CLUSTER_NAME)) \
		$(if $(CENTRAL_BACKEND_URL),--set centralProxy.centralBackendURL=$(CENTRAL_BACKEND_URL)) \
		$(FLAGS)

# Uninstall Odigos from the current cluster using local cli/ source.
.PHONY: cli-uninstall
cli-uninstall:
	@echo "Uninstalling odigos from source."
	cd ./cli ; go run -tags=embed_manifests . uninstall

# Upgrade an existing install using local cli/ source (same Helm flow as install).
# See cli-install for chart vs image.tag guidance.
# Override: make cli-upgrade ODIGOS_CLI_VERSION=v1.32.2
.PHONY: cli-upgrade
cli-upgrade:
	@echo "Upgrading odigos from source. image.tag: $(or $(ODIGOS_CLI_VERSION),<chart appVersion>)"
	cd ./cli ; go run -tags=embed_manifests . upgrade \
		$(if $(CLI_CHART_VERSION),--chart-version $(CLI_CHART_VERSION),--chart ../helm/odigos) \
		$(if $(ODIGOS_CLI_VERSION),--set image.tag=$(ODIGOS_CLI_VERSION)) \
		$(if $(CLUSTER_NAME),--set clusterName=$(CLUSTER_NAME)) \
		$(if $(CENTRAL_BACKEND_URL),--set centralProxy.centralBackendURL=$(CENTRAL_BACKEND_URL)) \
		$(FLAGS)

# Build the cli/odigos binary for local/e2e use, with helm charts embedded.
# Hardcodes chart/binary version to 0.0.0-e2e-test (not TAG). Output: cli/odigos.
.PHONY: cli-build
cli-build:
	@echo "Building the cli executable for tests"
	TAG=0.0.0-e2e-test; \
	TMPDIR=$$(mktemp -d); \
	cp -r ./helm/odigos $$TMPDIR/odigos; \
	sed -i.bak -E 's/^version:.*/version: '"$${TAG#v}"'/' $$TMPDIR/odigos/Chart.yaml; \
	helm package $$TMPDIR/odigos -d cli/pkg/helm/embedded; \
	cp -r ./helm/odigos-central $$TMPDIR/odigos-central; \
	sed -i.bak -E 's/^version:.*/version: '"$${TAG#v}"'/' $$TMPDIR/odigos-central/Chart.yaml; \
	helm package $$TMPDIR/odigos-central -d cli/pkg/helm/embedded; \
	cd cli && go build -tags=embed_manifests \
	  -ldflags "-X github.com/odigos-io/odigos/cli/pkg/helm.OdigosChartVersion=$${TAG#v}" \
	  -o odigos .; \
	rm -rf $$TMPDIR

# Collect cluster diagnostics via local cli/ source (`odigos diagnose`).
.PHONY: cli-diagnose
cli-diagnose:
	@echo "Diagnosing cluster data for debugging"
	cd ./cli ; go run -tags=embed_manifests . diagnose

# Chart version embedded in the CLI (must be SemVer for helm package).
# Defaults to TAG; non-SemVer tags are normalized in cli/Dockerfile to 0.0.0-<tag>.
# Override explicitly when needed: make build-cli-image TAG=e2e-test CHART_VERSION=0.0.0-e2e-test
CHART_VERSION ?= $(TAG)

# Build the CLI container image with docker (cli/Dockerfile).
# Default tag: $(CLI_IMAGE) = $(ORG)/odigos-cli$(IMG_SUFFIX):$(TAG)
# Override: make build-cli-image TAG=e2e-test
#           make build-cli-image CLI_IMAGE=my.registry/odigos-cli:dev
# RHEL via Dockerfile target: make build-cli-image RHEL=true
.PHONY: build-cli-image
build-cli-image:
	docker build $(DOCKER_BUILD_OPTS) $(TARGET_FLAG) -t $(CLI_IMAGE) -f cli/Dockerfile . \
		--build-arg VERSION=$(TAG) \
		--build-arg CHART_VERSION=$(CHART_VERSION) \
		--build-arg RELEASE=$(TAG) \
		--build-arg SUMMARY="$(CLI_SUMMARY)" \
		--build-arg DESCRIPTION="$(CLI_DESCRIPTION)" \
		--build-arg SHORT_COMMIT=$(SHORT_COMMIT) \
		--build-arg DATE=$(DATE) \
		--build-arg RHEL=$(RHEL)

# Extract the /odigos binary from a CLI image to cli/odigos for use on the host.
# Default image: $(CLI_IMAGE). Requires the image to be local or pullable.
# Override: make extract-cli CLI_IMAGE=registry.odigos.io/odigos-cli:v1.2.3
# Use after build-cli-image, or in CI once the image is available for the runner arch.
.PHONY: extract-cli
extract-cli:
	@cid=$$(docker create $(CLI_IMAGE)); \
	docker cp $$cid:/odigos cli/odigos; \
	docker rm $$cid >/dev/null; \
	chmod +x cli/odigos
	@echo "Extracted $(CLI_IMAGE) -> cli/odigos"

# Build the RHEL-certified CLI image locally with ko (not pushed).
# Builds Dockerfile.rhel-base then ko-builds onto it. Tag: $(TAG).
# Override: make build-cli-image-rhel TAG=v1.2.3 ORG=registry.odigos.io
.PHONY: build-cli-image-rhel
build-cli-image-rhel:
	cd cli && $(MAKE) licenses
	cd cli && docker build -f Dockerfile.rhel-base -t $(CLI_RHEL_KO_BASE_IMAGE) .
	cd cli && \
	KO_DOCKER_REPO=$(ORG)/$(CLI_RHEL_IMAGE_NAME) \
	KO_DEFAULTBASEIMAGE=$(CLI_RHEL_KO_BASE_IMAGE) \
	KO_CONFIG_PATH=./.ko.yaml \
	VERSION=$(TAG) \
	SHORT_COMMIT=$(shell git rev-parse --short HEAD) \
	DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
	ko build --bare --tags $(TAG) --local .

# Multi-arch build and push of the RHEL-certified CLI image (amd64+arm64).
# Requires PUSH_IMAGE=true (safety guard). Pushes base + certified image to $(ORG).
# Override: make push-cli-image-rhel PUSH_IMAGE=true TAG=v1.2.3 ORG=...
.PHONY: push-cli-image-rhel
push-cli-image-rhel:
	@if [ "$(PUSH_IMAGE)" != "true" ]; then \
		echo "this command will push the image to the public registry; set PUSH_IMAGE=true" >&2; \
		exit 1; \
	fi
	cd cli && $(MAKE) licenses
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f cli/Dockerfile.rhel-base \
		-t $(CLI_RHEL_KO_BASE_IMAGE) \
		--push \
		cli
	cd cli && \
	KO_DOCKER_REPO=$(ORG)/$(CLI_RHEL_IMAGE_NAME) \
	KO_DEFAULTBASEIMAGE=$(CLI_RHEL_KO_BASE_IMAGE) \
	KO_CONFIG_PATH=./.ko.yaml \
	VERSION=$(TAG) \
	SHORT_COMMIT=$(shell git rev-parse --short HEAD) \
	DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
	ko build --bare --tags $(TAG) \
		--image-label name=odigos-cli \
		--image-label vendor=Odigos \
		--image-label maintainer=Odigos \
		--image-label version=$(TAG) \
		--image-label release=$(TAG) \
		--image-label summary="Odigos CLI" \
		--image-label description="Odigos CLI to install and manage Odigos in your Kubernetes cluster." \
		--platform=all .
