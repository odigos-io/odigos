TAG ?= $(shell odigos version --cluster 2>/dev/null || odigos version --cli 2>/dev/null || helm search repo odigos 2>/dev/null | awk '$$1 == "odigos/odigos" {print $$3}')
ODIGOS_CLI_VERSION ?= $(shell odigos version --cli)
CLUSTER_NAME ?= local-dev-cluster
CENTRAL_BACKEND_URL ?=
ORG ?= registry.odigos.io
# Override ORG for staging pushes
ifeq ($(STAGING_ORG),true)
    ORG = us-central1-docker.pkg.dev/odigos-cloud/staging-components
endif
GOLANGCI_LINT_VERSION ?= v2.10.1
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
GO_MODULES := $(shell find . -type f -name "go.mod" -not -path "*/vendor/*" -exec dirname {} \; | grep -v "licenses")
LINT_CMD = golangci-lint run -c ../.golangci.yml
ifdef FIX_LINT
    LINT_CMD += --fix
endif
DOCKERFILE=Dockerfile
IMG_PREFIX?=
IMG_SUFFIX?=
RHEL_IMAGE_SUFFIX ?= -rhel-certified
BUILD_DIR=.
HOST_PLATFORM ?= $(shell ARCH=$$(uname -m); \
	if [ "$$ARCH" = "x86_64" ]; then echo "linux/amd64"; \
	elif [ "$$ARCH" = "arm64" ] || [ "$$ARCH" = "aarch64" ]; then echo "linux/arm64"; \
	else echo "linux/amd64"; fi)
HOST_GOOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_GOARCH ?= $(shell ARCH=$$(uname -m); \
	if [ "$$ARCH" = "x86_64" ]; then echo "amd64"; \
	elif [ "$$ARCH" = "arm64" ] || [ "$$ARCH" = "aarch64" ]; then echo "arm64"; \
	else echo "$$ARCH"; fi)

# RHEL-certified CLI image naming
CLI_RHEL_IMAGE_NAME ?= odigos-cli-rhel-certified

.PHONY: install-golangci-lint
install-golangci-lint:
	@if ! which golangci-lint >/dev/null || [ "$$(golangci-lint version 2>&1 | head -n 1 | awk '{print "v"$$4}')" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	else \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) is already installed"; \
	fi

.PHONY: lint
lint: install-golangci-lint
ifdef MODULE
	@echo "Running lint for module: $(MODULE)"
	@if [ ! -d "$(MODULE)" ]; then \
		echo "Error: Directory $(MODULE) does not exist"; \
		exit 1; \
	fi
	@if [ ! -f "$(MODULE)/go.mod" ]; then \
		echo "Error: $(MODULE) is not a Go module (no go.mod found)"; \
		exit 1; \
	fi
	@cd $(MODULE) && $(LINT_CMD) ./...
else
	@echo "No MODULE specified, running lint for all Go modules..."
	@for module in $(GO_MODULES); do \
		echo "Running lint for $$module"; \
		(cd $$module && $(LINT_CMD) ./...) || exit 1; \
	done
endif

.PHONY: lint-fix
lint-fix:
	MODULE=common make lint FIX_LINT=true
	MODULE=k8sutils make lint FIX_LINT=true
	MODULE=profiles make lint FIX_LINT=true
	MODULE=destinations make lint FIX_LINT=true
	MODULE=procdiscovery make lint FIX_LINT=true

.PHONY: cli-docs
cli-docs:
	rm -rf docs/snippets/shared/cli/*
	cd scripts/cli-docgen && KUBECONFIG=KUBECONFIG go run -tags embed_manifests main.go
	for file in docs/snippets/shared/cli/*.md; do \
		mv $${file} $${file%.md}.mdx; \
	done

.PHONY: rbac-docs
rbac-docs:
	cd scripts/rbac-docgen && go run main.go

TOOLS_DIR := .tools
HELM_SCHEMA_VERSION := latest
HELM_SCHEMA_BIN := $(TOOLS_DIR)/helm-schema

.PHONY: helm-schema
helm-schema: $(HELM_SCHEMA_BIN)
	cd helm/odigos && ../../$(HELM_SCHEMA_BIN) --uncomment -k required

$(HELM_SCHEMA_BIN):
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(CURDIR)/$(TOOLS_DIR) go install github.com/dadav/helm-schema/cmd/helm-schema@$(HELM_SCHEMA_VERSION)

.PHONY: helm-schema-clean
helm-schema-clean:
	rm -f $(HELM_SCHEMA_BIN)

build-rhel-%:
	@echo "Building for architecture: $(HOST_PLATFORM)"
	@SERVICE="$*"; \
	BAKE_TARGET="odigos-$$SERVICE-rhel"; \
	IMAGE_SUFFIX="$(if $(IMG_SUFFIX),$(IMG_SUFFIX),$(RHEL_IMAGE_SUFFIX))" ODIGOS_TAG="$(TAG)" \
	docker buildx bake --file docker-bake.hcl "$$BAKE_TARGET" \
		--set "$$BAKE_TARGET.args.ODIGOS_VERSION=$(TAG)" \
		--set "$$BAKE_TARGET.args.VERSION=$(TAG)" \
		--set "$$BAKE_TARGET.args.RELEASE=$(TAG)" \
		--set "$$BAKE_TARGET.args.LD_FLAGS=$(LD_FLAGS)" \
		--set "*.platform=$(HOST_PLATFORM)" \
		--set "*.output=type=docker"

build-%:
	@echo "Building for architecture: $(HOST_PLATFORM)"
	@SERVICE="$*"; \
	BAKE_TARGET="odigos-$$SERVICE"; \
	IMAGE_SUFFIX="$(IMG_SUFFIX)" ODIGOS_TAG="$(TAG)" \
	docker buildx bake --file docker-bake.hcl "$$BAKE_TARGET" \
		--set "$$BAKE_TARGET.args.ODIGOS_VERSION=$(TAG)" \
		--set "$$BAKE_TARGET.args.VERSION=$(TAG)" \
		--set "$$BAKE_TARGET.args.RELEASE=$(TAG)" \
		--set "$$BAKE_TARGET.args.LD_FLAGS=$(LD_FLAGS)" \
		--set "*.platform=$(HOST_PLATFORM)" \
		--set "*.output=type=docker"

.PHONY: build-operator-index
build-operator-index:
	opm index add --bundles $(ORG)/odigos-bundle:$(TAG) --tag $(ORG)/odigos-index:$(TAG) --container-tool=docker

.PHONY: verify-nodejs-agent
verify-nodejs-agent:
	@if [ ! -f ../opentelemetry-node/package.json ]; then \
		echo "Error: To build odiglet agents from source, first clone the agents code locally"; \
		exit 1; \
	fi

.PHONY: build-images
build-images:
	@echo "Building for architecture: $(HOST_PLATFORM)"
	IMAGE_SUFFIX="$(IMG_SUFFIX)" ODIGOS_TAG="$(TAG)" \
	docker buildx bake --file docker-bake.hcl stack \
		--set "*.args.ODIGOS_VERSION=$(TAG)" \
		--set "*.args.VERSION=$(TAG)" \
		--set "*.args.RELEASE=$(TAG)" \
		--set "*.args.LD_FLAGS=$(LD_FLAGS)" \
		--set "*.platform=$(HOST_PLATFORM)" \
		--set "*.output=type=docker"

.PHONY: build-rhel-images
build-rhel-images:
	@echo "Building for architecture: $(HOST_PLATFORM)"
	IMAGE_SUFFIX="$(if $(IMG_SUFFIX),$(IMG_SUFFIX),$(RHEL_IMAGE_SUFFIX))" ODIGOS_TAG="$(TAG)" \
	docker buildx bake --file docker-bake.hcl stack-rhel \
		--set "*.args.ODIGOS_VERSION=$(TAG)" \
		--set "*.args.VERSION=$(TAG)" \
		--set "*.args.RELEASE=$(TAG)" \
		--set "*.args.LD_FLAGS=$(LD_FLAGS)" \
		--set "*.platform=$(HOST_PLATFORM)" \
		--set "*.output=type=docker"

.PHONY: build-images-rhel
build-images-rhel:
	$(MAKE) build-rhel-images TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX="$(IMG_SUFFIX)"

load-to-kind-%:
	kind load docker-image $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG)

.PHONY: load-to-kind
load-to-kind:
	make -j 6 load-to-kind-instrumentor load-to-kind-autoscaler load-to-kind-scheduler load-to-kind-odiglet load-to-kind-collector load-to-kind-ui load-to-kind-cli load-to-kind-agents ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX) DOCKERFILE=$(DOCKERFILE)

.PHONY: restart-ui
restart-ui:
	-kubectl rollout restart deployment odigos-ui -n odigos-system

.PHONY: restart-odiglet
restart-odiglet:
	-kubectl rollout restart daemonset odiglet -n odigos-system

.PHONY: restart-autoscaler
restart-autoscaler:
	-kubectl rollout restart deployment odigos-autoscaler -n odigos-system

.PHONY: restart-instrumentor
restart-instrumentor:
	-kubectl rollout restart deployment odigos-instrumentor -n odigos-system

.PHONY: restart-scheduler
restart-scheduler:
	-kubectl rollout restart deployment odigos-scheduler -n odigos-system

.PHONY: restart-collector
restart-collector:
	-kubectl rollout restart deployment odigos-gateway -n odigos-system
	# DaemonSets don't directly support the rollout restart command in the same way Deployments do. However, you can achieve the same result by updating an environment variable or any other field in the DaemonSet's pod template, triggering a rolling update of the pods managed by the DaemonSet
	# Restart the odiglet DaemonSet because data-collection Collector is part of it
	-kubectl -n odigos-system patch daemonset odiglet -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"kubectl.kubernetes.io/restartedAt\":\"$(date +%Y-%m-%dT%H:%M:%S%z)\"}}}}}"

deploy-%:
	$(MAKE) build-$* ORG=$(ORG) TAG=$(TAG) DOCKERFILE=$(DOCKERFILE) IMG_SUFFIX=$(IMG_SUFFIX)
	$(MAKE) load-to-kind-$* ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX)
	@if [ "$*" != "agents" ]; then \
		$(MAKE) restart-$* ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX); \
	fi

.PHONY: deploy
deploy:
	make deploy-odiglet deploy-autoscaler deploy-collector deploy-instrumentor deploy-scheduler deploy-ui

.PHONY: debug-odiglet
debug-odiglet:
	docker build -t $(ORG)/odigos-odiglet:$(TAG) . -f odiglet/debug.Dockerfile
	kind load docker-image $(ORG)/odigos-odiglet:$(TAG)
	kubectl delete pod -n odigos-system -l app.kubernetes.io/name=odiglet
	kubectl wait --for=condition=ready pod -n odigos-system -l app.kubernetes.io/name=odiglet --timeout=180s
	kubectl port-forward -n odigos-system daemonset/odiglet 2345:2345

ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort | grep -v "licenses")

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%)
go-mod-tidy/%: DIR=$*
go-mod-tidy/%:
	@cd $(DIR) && go mod tidy -compat=1.21

.PHONY: update-dep
update-dep: $(ALL_GO_MOD_DIRS:%=update-dep/%)
update-dep/%: DIR=$*
update-dep/%:
	cd $(DIR) && go get $(MODULE)@$(VERSION)

UNSTABLE_COLLECTOR_VERSION=v0.148.0
STABLE_COLLECTOR_VERSION=v1.54.0
STABLE_OTEL_GO_VERSION=v1.38.0
UNSTABLE_OTEL_GO_VERSION=v0.63.0

.PHONY: update-otel
update-otel:
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/cmd/mdatagen VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/component VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/component/componenttest VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/config/configtelemetry VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/confmap VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/confmap/provider/envprovider VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/connector VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/connector/forwardconnector VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/consumer VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/consumer/consumertest VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/exporter VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/exporter/debugexporter VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/exporter/exportertest VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/exporter/nopexporter VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/exporter/otlpexporter VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/exporter/otlphttpexporter VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/extension VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/extension/zpagesextension VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/otelcol VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/pdata VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor/batchprocessor VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor/memorylimiterprocessor VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor/processortest VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/receiver VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/receiver/otlpreceiver VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/receiver/receivertest VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/otel VERSION=$(STABLE_OTEL_GO_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc VERSION=$(STABLE_OTEL_GO_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/otel/metric VERSION=$(STABLE_OTEL_GO_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/otel/sdk/metric VERSION=$(STABLE_OTEL_GO_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/otel/trace VERSION=$(STABLE_OTEL_GO_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc VERSION=$(UNSTABLE_OTEL_GO_VERSION)
	$(MAKE) update-dep MODULE=github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) go-mod-tidy

.PHONY: check-clean-work-tree
check-clean-work-tree:
	if [ -n "$$(git status --porcelain)" ]; then \
		git status; \
		git --no-pager diff; \
		echo 'Working tree is not clean, did you forget to run "make go-mod-tidy"?'; \
		exit 1; \
	fi

# installs odigos from the local source, with local changes to api and cli directorie reflected in the odigos deployment
.PHONY: cli-install
cli-install:
	@echo "Installing odigos from source. version: $(ODIGOS_CLI_VERSION)"
	cd ./cli ; go run -tags=embed_manifests . install \
		--version $(ODIGOS_CLI_VERSION) \
		--nowait \
		$(if $(CLUSTER_NAME),--cluster-name $(CLUSTER_NAME)) \
		$(if $(CENTRAL_BACKEND_URL),--central-backend-url $(CENTRAL_BACKEND_URL)) \
		$(FLAGS)


.PHONY: cli-uninstall
cli-uninstall:
	@echo "Uninstalling odigos from source. version: $(ODIGOS_CLI_VERSION)"
	cd ./cli ; go run -tags=embed_manifests . uninstall

.PHONY: cli-upgrade
cli-upgrade:
	@echo "Upgrading odigos from source. version: $(ODIGOS_CLI_VERSION)"
	cd ./cli ; go run -tags=embed_manifests . upgrade --version $(ODIGOS_CLI_VERSION) --yes

.PHONY: cli-build
cli-build:
	@echo "Building host binary for $(HOST_GOOS)/$(HOST_GOARCH)"
	@IMAGE=$(ORG)/odigos-cli-builder:$(TAG)-$(HOST_GOOS)-$(HOST_GOARCH); \
	docker buildx bake --file docker-bake.hcl odigos-cli \
		--set "odigos-cli.target=builder" \
		--set "*.platform=$(HOST_PLATFORM)" \
		--set "odigos-cli.args.ODIGOS_TAG=$(TAG)" \
		--set "odigos-cli.args.TARGETOS=$(HOST_GOOS)" \
		--set "odigos-cli.args.TARGETARCH=$(HOST_GOARCH)" \
		--set "odigos-cli.tags=$$IMAGE" \
		--set "*.output=type=docker"; \
	CID=$$(docker create $$IMAGE); \
	trap 'docker rm -f "$$CID" >/dev/null 2>&1 || true' EXIT; \
	docker cp "$$CID:/workspace/build/odigos" ./cli/odigos; \
	chmod +x ./cli/odigos; \
	docker rm -f "$$CID" >/dev/null 2>&1; \
	trap - EXIT

.PHONY: cli-diagnose
cli-diagnose:
	@echo "Diagnosing cluster data for debugging"
	cd ./cli ; go run -tags=embed_manifests . diagnose

.PHONY: helm-install
helm-install:
	@echo "Installing odigos using helm"
	helm upgrade --install odigos ./helm/odigos \
		--create-namespace \
		--namespace odigos-system \
		--set image.tag=$(ODIGOS_CLI_VERSION) \
		--set clusterName=$(CLUSTER_NAME) \
		--set centralProxy.centralBackendURL=$(CENTRAL_BACKEND_URL) \
		--set onPremToken=$(ONPREM_TOKEN)

.PHONY: helm-install-central
helm-install-central:
	@echo "Installing Odigos Central using Helm..."
	helm upgrade --install odigos-central ./helm/odigos-central \
		--create-namespace \
		--namespace odigos-central \
		--set image.tag=$(ODIGOS_CLI_VERSION) \
		--set onPremToken=$(ONPREM_TOKEN) \
		--set auth.adminUsername=$(CENTRAL_ADMIN_USER) \
		--set auth.adminPassword=$(CENTRAL_ADMIN_PASSWORD) \
	kubectl label namespace odigos-central odigos.io/central-system-object="true" --overwrite


.PHONY: api-all
api-all:
	make -C api all

.PHONY: crd-apply
crd-apply: api-all cli-upgrade
	@echo "Applying changes to CRDs in api directory"

.PHONY: dev-tests-kind-cluster
dev-tests-kind-cluster:
	@echo "Creating a kind cluster for development"
	kind delete cluster
	kind create cluster --config=tests/common/apply/kind-config.yaml

.PHONY: dev-tests-setup
dev-tests-setup: TAG := e2e-test
dev-tests-setup: dev-tests-kind-cluster cli-build build-cli-image build-images load-to-kind

# Use this target to avoid rebuilding the images if all that changed is the e2e test code
.PHONY: dev-tests-setup-no-build
dev-tests-setup-no-build: TAG := e2e-test
dev-tests-setup-no-build: dev-tests-kind-cluster load-to-kind

# Use this for debug to add a destination which only prints samples of telemetry items to the cluster gateway collector logs
.PHONY: dev-debug-destination
dev-debug-destination:
	kubectl apply -f ./tests/debug-exporter.yaml

.PHONY: dev-add-nop-destination
dev-nop-destination:
	kubectl apply -f ./tests/nop-exporter.yaml

.PHONY: dev-add-dynamic-destination
dev-dynamic-destination:
	kubectl apply -f ./tests/dynamic-exporter.yaml

.PHONY: dev-add-backpressue-destination
dev-backpressue-destination:
	kubectl apply -f ./tests/backpressure-exporter.yaml

# Use these to deploy Odigos into an EKS cluster

.PHONY: ecr-login
ecr-login:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws

build-tag-push-ecr-image/%:
	docker build --platform linux/amd64 -t $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG) $(BUILD_DIR) -f $(DOCKERFILE) \
	--build-arg SERVICE_NAME="$*" \
	--build-arg ODIGOS_VERSION=$(TAG) \
	--build-arg VERSION=$(TAG) \
	--build-arg RELEASE=$(TAG) \
	--build-arg SUMMARY="$(SUMMARY)" \
	--build-arg DESCRIPTION="$(DESCRIPTION)"
	docker tag $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG) $(IMG_PREFIX)/odigos-$*$(IMG_SUFFIX):$(TAG)
	docker push $(IMG_PREFIX)/odigos-$*$(IMG_SUFFIX):$(TAG)

.PHONY: publish-to-ecr
publish-to-ecr:
	if [ -z "$(IMG_PREFIX)" ]; then \
		echo "❌ IMG_PREFIX is not set"; \
		exit 1; \
	fi
	make ecr-login
	make -j 3 build-tag-push-ecr-image/odiglet DOCKERFILE=odiglet/$(DOCKERFILE) SUMMARY="Odiglet for Odigos" DESCRIPTION="Odiglet is the core component of Odigos managing auto-instrumentation. This container requires a root user to run and manage eBPF programs." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j 3 build-tag-push-ecr-image/autoscaler SUMMARY="Autoscaler for Odigos" DESCRIPTION="Autoscaler manages the installation of Odigos components." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j 3 build-tag-push-ecr-image/instrumentor SUMMARY="Instrumentor for Odigos" DESCRIPTION="Instrumentor manages auto-instrumentation for workloads with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j 3 build-tag-push-ecr-image/scheduler SUMMARY="Scheduler for Odigos" DESCRIPTION="Scheduler manages the installation of OpenTelemetry Collectors with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j 3 build-tag-push-ecr-image/collector DOCKERFILE=collector/$(DOCKERFILE) SUMMARY="Odigos Collector" DESCRIPTION="The Odigos build of the OpenTelemetry Collector." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j 3 build-tag-push-ecr-image/ui DOCKERFILE=frontend/$(DOCKERFILE) SUMMARY="UI for Odigos" DESCRIPTION="UI provides the frontend webapp for managing an Odigos installation." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	echo "✅ Deployed Odigos to EKS, now install the CLI"

.PHONY: build-cli-image
build-cli-image:
	@echo "Building for architecture: $(HOST_PLATFORM)"
	docker build -f cli/Dockerfile \
		--platform=$(HOST_PLATFORM) \
		--build-arg ODIGOS_TAG=$(TAG) \
		-t $(ORG)/odigos-cli$(IMG_SUFFIX):$(TAG) .

.PHONY: build-cli-image-rhel
build-cli-image-rhel:
	docker build -f cli/Dockerfile \
		--platform=$(HOST_PLATFORM) \
		--target rhel \
		--build-arg ODIGOS_TAG=$(TAG) \
		--build-arg VERSION=$(TAG) \
		--build-arg RELEASE=$(TAG) \
		--build-arg SUMMARY="Odigos CLI" \
		--build-arg DESCRIPTION="Odigos CLI to install and manage Odigos in your Kubernetes cluster." \
		-t $(ORG)/$(CLI_RHEL_IMAGE_NAME):$(TAG) .

# install gatekeeper to prevent:
# 1. privileged containers
# 2. hostPath volumes (except for some specific paths which are allowed on most clusters)
# 3. hostNamespace (hostNetwork, hostPID, hostIPC)
# 4. allowPrivilegeEscalation is enforced to explicitly set to false
install-gatekeeper:
	helm repo add gatekeeper https://open-policy-agent.github.io/gatekeeper/charts
	helm repo update
	helm install gatekeeper gatekeeper/gatekeeper --namespace gatekeeper-system --create-namespace
	@max_retries=5; \
	backoff=2; \
	attempt=1; \
	until kubectl apply -f tests/gatekeeper/constraints/; do \
		if [ $$attempt -ge $$max_retries ]; then \
			echo "kubectl apply failed after $$attempt attempts."; \
			exit 1; \
		fi; \
		echo "kubectl apply failed. Retrying in $$backoff seconds..."; \
		sleep $$backoff; \
		backoff=$$((backoff * 2)); \
		attempt=$$((attempt + 1)); \
	done

