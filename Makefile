# ──────────────────────────────────────────────
# Core variables
# ──────────────────────────────────────────────
# TODO: Use installed version, if not applicatble, use latest git tag
LATEST_GIT_TAG = $(shell git tag --list 'v*' --sort=-v:refname | head -n1 2>/dev/null)
TAG ?= $(shell odigos version --cluster 2>/dev/null || echo $LATEST_GIT_TAG)
ODIGOS_CLI_VERSION ?= $(shell odigos version --cli)
CLUSTER_NAME ?= local-dev-cluster
CENTRAL_BACKEND_URL ?=
ONPREM_TOKEN ?=
ODIGOS_NAMESPACE ?= odigos-system
ORG ?= registry.odigos.io
SHORT_COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
PARALLEL ?= $(shell nproc)
PLATFORMS ?=

GOLANGCI_LINT_VERSION ?= v2.1.6
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
GO_MODULES := $(shell find . -type f -name "go.mod" -not -path "*/vendor/*" -exec dirname {} \; | grep -v "licenses")

LINT_CMD = golangci-lint run -c ../.golangci.yml
ifdef FIX_LINT
    LINT_CMD += --fix
endif

DOCKERFILE=Dockerfile
IMG_PREFIX ?=
IMG_SUFFIX ?=
BUILD_DIR  ?= .


# ──────────────────────────────────────────────
# Build / push helpers
# ──────────────────────────────────────────────
define bake-load
	TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX) \
	docker buildx bake $(1) --load --no-cache \
	$(if $(PLATFORMS),--set '*.platform=$(PLATFORMS)',) \
	$(if $(LD_FLAGS),--set '*.args.LD_FLAGS=$(LD_FLAGS)',)
endef

define bake-push
	@TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX) \
	docker buildx bake $(1) --push \
	$(if $(PLATFORMS),--set '*.platform=$(PLATFORMS)',) \
	$(if $(LD_FLAGS),--set '*.args.LD_FLAGS=$(LD_FLAGS)',)
endef

# Pattern rules for every service image
build-%: FORCE ; $(call bake-load,$*)
push-%: FORCE ; $(call bake-push,$*)

.PHONY: FORCE
FORCE:;

# Convenience groups matching the HCL
.PHONY: build-images push-images build-images-rhel push-images-rhel build-cli build-cli-rhel
build-images:       ; $(call bake-load,images)
push-images:        ; $(call bake-push,images)
build-images-rhel:  ; $(call bake-load,images-rhel)
push-images-rhel:   ; $(call bake-push,images-rhel)

build-cli:
	$(call bake-load,cli \
	  --set cli.args.SHORT_COMMIT=$(SHORT_COMMIT) \
	  --set cli.args.DATE=$(DATE))
build-cli-rhel: build-cli

push-cli:
	$(call bake-load,cli \
	  --set cli.args.SHORT_COMMIT=$(SHORT_COMMIT) \
	  --set cli.args.DATE=$(DATE))
push-cli-rhel: push-cli

# ──────────────────────────────────────────────
# Kind helpers
# ──────────────────────────────────────────────
.PHONY:
load-to-kind-%: FORCE
	kind load docker-image $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG)

load-to-kind: FORCE
	$(MAKE) -j $(nproc) load-to-kind-instrumentor load-to-kind-autoscaler \
		load-to-kind-scheduler load-to-kind-odiglet \
		load-to-kind-collector load-to-kind-ui load-to-kind-cli \
		ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX)


# ──────────────────────────────────────────────
# Lint / docs / misc sections
# ──────────────────────────────────────────────
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
	MODULE=common $(MAKE) lint FIX_LINT=true
	MODULE=k8sutils $(MAKE) lint FIX_LINT=true
	MODULE=profiles $(MAKE) lint FIX_LINT=true
	MODULE=destinations $(MAKE) lint FIX_LINT=true
	MODULE=procdiscovery $(MAKE) lint FIX_LINT=true

.PHONY: cli-docs
cli-docs:
	rm -rf docs/cli/*
	cd scripts/cli-docgen && KUBECONFIG=KUBECONFIG go run -tags embed_manifests main.go
	for file in docs/cli/*; do mv $${file} $${file%.md}.mdx; done

.PHONY: rbac-docs
rbac-docs:
	cd scripts/rbac-docgen && go run main.go

.PHONY: check-clean-work-tree
check-clean-work-tree:
	if [ -n "$$(git status --porcelain)" ]; then \
		git status; \
		git --no-pager diff; \
		echo 'Working tree is not clean, did you forget to run "make go-mod-tidy"?'; \
		exit 1; \
	fi


# ──────────────────────────────────────────────
# Custom image variations that still need
# manual docker build commands
# ──────────────────────────────────────────────
.PHONY: build-odiglet-with-agents
build-odiglet-with-agents:
	docker build -t $(ORG)/odigos-odiglet$(IMG_SUFFIX):$(TAG) . -f odiglet/$(DOCKERFILE) \
	--build-context nodejs-agent-src=../opentelemetry-node \
	--build-arg ODIGOS_VERSION=$(TAG) \
	--build-arg VERSION=$(TAG) \
	--build-arg RELEASE=$(TAG) \
	--build-arg SUMMARY="Odiglet for Odigos" \
	--build-arg DESCRIPTION="Odiglet is the core component of Odigos managing auto-instrumentation."

.PHONY: verify-nodejs-agent
verify-nodejs-agent:
	@if [ ! -f ../opentelemetry-node/package.json ]; then \
		echo "Error: To build odiglet agents from source, first clone the agents code locally"; \
		exit 1; \
	fi

.PHONY: debug-odiglet
debug-odiglet:
	docker build -t $(ORG)/odigos-odiglet:$(TAG) . -f odiglet/debug.Dockerfile
	kind load docker-image $(ORG)/odigos-odiglet:$(TAG)
	kubectl delete pod -n $(ODIGOS_NAMESPACE) -l app.kubernetes.io/name=odiglet
	kubectl wait --for=condition=ready pod -n $(ODIGOS_NAMESPACE) -l app.kubernetes.io/name=odiglet --timeout=180s
	kubectl port-forward -n $(ODIGOS_NAMESPACE) daemonset/odiglet 2345:2345

.PHONY: build-operator-index
build-operator-index:
	opm index add --bundles $(ORG)/odigos-bundle:$(TAG) --tag $(ORG)/odigos-index:$(TAG) --container-tool=docker

<<<<<<< HEAD
push-image/%:
	docker buildx build --platform linux/amd64,linux/arm64/v8 -t $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG) $(BUILD_DIR) -f $(DOCKERFILE) \
	--build-arg SERVICE_NAME="$*" \
	--build-arg VERSION=$(TAG) \
	--build-arg RELEASE=$(TAG) \
	--build-arg SUMMARY="$(SUMMARY)" \
	--build-arg DESCRIPTION="$(DESCRIPTION)"

.PHONY: push-operator
push-operator:
	$(MAKE) push-image/operator DOCKERFILE=operator/$(DOCKERFILE) SUMMARY="Odigos Operator" DESCRIPTION="Kubernetes Operator for Odigos installs Odigos" TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: push-odiglet
push-odiglet:
	$(MAKE) push-image/odiglet DOCKERFILE=odiglet/$(DOCKERFILE) SUMMARY="Odiglet for Odigos" DESCRIPTION="Odiglet is the core component of Odigos managing auto-instrumentation." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: push-autoscaler
push-autoscaler:
	$(MAKE) push-image/autoscaler SUMMARY="Autoscaler for Odigos" DESCRIPTION="Autoscaler manages the installation of Odigos components." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: push-instrumentor
push-instrumentor:
	$(MAKE) push-image/instrumentor SUMMARY="Instrumentor for Odigos" DESCRIPTION="Instrumentor manages auto-instrumentation for workloads with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: push-scheduler
push-scheduler:
	$(MAKE) push-image/scheduler SUMMARY="Scheduler for Odigos" DESCRIPTION="Scheduler manages the installation of OpenTelemetry Collectors with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: push-collector
push-collector:
	$(MAKE) push-image/collector DOCKERFILE=collector/$(DOCKERFILE) BUILD_DIR=collector SUMMARY="Odigos Collector" DESCRIPTION="The Odigos build of the OpenTelemetry Collector." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: push-ui
push-ui:
	$(MAKE) push-image/ui DOCKERFILE=frontend/$(DOCKERFILE) SUMMARY="UI for Odigos" DESCRIPTION="UI provides the frontend webapp for managing an Odigos installation." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: push-images
push-images:
	make push-autoscaler push-scheduler push-odiglet push-instrumentor push-collector push-ui TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX) DOCKERFILE=$(DOCKERFILE)

.PHONY: push-images-rhel
push-images-rhel:
	$(MAKE) push-images IMG_SUFFIX=-ubi9 DOCKERFILE=Dockerfile.rhel TAG=$(TAG) ORG=$(ORG)

load-to-kind-%:
	kind load docker-image $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG)

.PHONY: load-to-kind
load-to-kind:
	make -j 6 load-to-kind-instrumentor load-to-kind-autoscaler load-to-kind-scheduler load-to-kind-odiglet load-to-kind-collector load-to-kind-ui load-to-kind-cli ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX) DOCKERFILE=$(DOCKERFILE)

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
	-kubectl -n odigos-system patch daemonset odigos-data-collection -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"kubectl.kubernetes.io/restartedAt\":\"$(date +%Y-%m-%dT%H:%M:%S%z)\"}}}}}"
=======
>>>>>>> 1908cb4f (Makefile cleanup and refactor)

# ──────────────────────────────────────────────
# Deploy helpers
# ──────────────────────────────────────────────
.PHONY: deploy
deploy-%: FORCE
	$(MAKE) build-$* ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX)
	$(MAKE) load-to-kind-$* ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX)
	$(MAKE) restart-$*

deploy: deploy-odiglet deploy-autoscaler deploy-collector \
        deploy-instrumentor deploy-scheduler deploy-ui

# Use this target to deploy odiglet with local clones of the agents.
# To work, the agents must be cloned in the same directory as the odigos (e.g. in '../opentelemetry-node')
# There you can make code changes to the agents and deploy them with the odiglet.
.PHONY: deploy-odiglet-with-agents
deploy-odiglet-with-agents: verify-nodejs-agent build-odiglet-with-agents load-to-kind-odiglet restart-odiglet


# ──────────────────────────────────────────────
# Restart helpers
# ──────────────────────────────────────────────
.PHONY: restart-ui restart-odiglet restart-autoscaler restart-instrumentor restart-scheduler restart-collector
restart-ui:          ; -kubectl rollout restart deployment  odigos-ui         	-n $(ODIGOS_NAMESPACE)
restart-odiglet:     ; -kubectl rollout restart daemonset   odiglet           	-n $(ODIGOS_NAMESPACE)
restart-autoscaler:  ; -kubectl rollout restart deployment  odigos-autoscaler 	-n $(ODIGOS_NAMESPACE)
restart-instrumentor:; -kubectl rollout restart deployment  odigos-instrumentor -n $(ODIGOS_NAMESPACE)
restart-scheduler:   ; -kubectl rollout restart deployment  odigos-scheduler  	-n $(ODIGOS_NAMESPACE)
restart-collector:
	-kubectl rollout restart deployment odigos-gateway -n $(ODIGOS_NAMESPACE)
	# DaemonSets don't directly support the rollout restart command in the same way Deployments do.
	# However, you can achieve the same result by updating an environment variable or any other field in the DaemonSet's pod template, triggering a rolling update of the pods managed by the DaemonSet
	-kubectl -n $(ODIGOS_NAMESPACE) patch daemonset odigos-data-collection \
	  -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"kubectl.kubernetes.io/restartedAt\":\"$(shell date +%Y-%m-%dT%H:%M:%S%z)\"}}}}}"


# ──────────────────────────────────────────────
# CLI install / upgrade / diagnose targets
# ──────────────────────────────────────────────
.PHONY: cli-install cli-uninstall cli-upgrade cli-build cli-diagnose

# installs odigos from the local source, with local changes to api and cli directorie reflected in the odigos deployment
cli-install:
	@echo "Installing odigos from source. version: $(ODIGOS_CLI_VERSION)"
	cd ./cli ; go run -tags=embed_manifests . install \
		--version $(ODIGOS_CLI_VERSION) --nowait \
		$(if $(CLUSTER_NAME),--cluster-name $(CLUSTER_NAME)) \
		$(if $(CENTRAL_BACKEND_URL),--central-backend-url $(CENTRAL_BACKEND_URL)) \
		$(FLAGS)

cli-uninstall:
	@echo "Uninstalling odigos from source. version: $(ODIGOS_CLI_VERSION)"
	cd ./cli ; go run -tags=embed_manifests . uninstall

cli-upgrade:
	@echo "Upgrading odigos from source. version: $(ODIGOS_CLI_VERSION)"
	cd ./cli ; go run -tags=embed_manifests . upgrade --version $(ODIGOS_CLI_VERSION) --yes

cli-build:
	@echo "Building the cli executable for tests"
	cd cli && go build -tags=embed_manifests -o odigos .

cli-diagnose:
	@echo "Diagnosing cluster data for debugging"
	cd ./cli ; go run -tags=embed_manifests . diagnose


# ──────────────────────────────────────────────
# Helm install helper
# ──────────────────────────────────────────────
.PHONY: helm-install
helm-install:
	@echo "Installing odigos using helm"
	helm upgrade --install odigos ./helm/odigos \
		--create-namespace --namespace $(ODIGOS_NAMESPACE) \
		--set image.tag=$(ODIGOS_CLI_VERSION) \
		--set clusterName=$(CLUSTER_NAME) \
		--set centralProxy.centralBackendURL=$(CENTRAL_BACKEND_URL) \
		--set onPremToken=$(ONPREM_TOKEN) \
		--set centralProxy.enabled=$(if $(and $(CLUSTER_NAME),$(CENTRAL_BACKEND_URL)),true,false)

.PHONY: helm-install-central
helm-install-central:
	@echo "Installing Odigos Central using Helm..."
	helm upgrade --install odigos-central ./helm/odigos-central \
		--create-namespace \
		--namespace odigos-central \
		--set image.tag=$(ODIGOS_CLI_VERSION) \
		--set onPremToken=$(ONPREM_TOKEN) \

helm-uninstall:
	@echo "Uninstalling odigos using helm"
	helm uninstall odigos --namespace $(ODIGOS_NAMESPACE)


# ──────────────────────────────────────────────
# API / CRD helpers
# ──────────────────────────────────────────────
.PHONY: api-all crd-apply
api-all:   ; $(MAKE) -C api all
crd-apply: ; $(MAKE) api-all && $(MAKE) cli-upgrade && echo "Applying changes to CRDs in api directory"


# ──────────────────────────────────────────────
# Go module helpers
# ──────────────────────────────────────────────
ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort | grep -v "licenses")

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%)
go-mod-tidy/%: DIR=$*
go-mod-tidy/%:
	@cd $* && go mod tidy -compat=1.21

.PHONY: update-dep
update-dep: $(ALL_GO_MOD_DIRS:%=update-dep/%)
update-dep/%: DIR=$*
update-dep/%:
	cd $* && go get $(MODULE)@$(VERSION)

UNSTABLE_COLLECTOR_VERSION=v0.126.0
STABLE_COLLECTOR_VERSION=v1.32.0
STABLE_OTEL_GO_VERSION=v1.35.0
UNSTABLE_OTEL_GO_VERSION=v0.60.0

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
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/exporter VERSION=$(UNSTABLE_COLLECTOR_VERSION)
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


# ──────────────────────────────────────────────
# Debug & test destinations
# ──────────────────────────────────────────────
.PHONY: dev-debug-destination dev-nop-destination dev-dynamic-destination dev-backpressue-destination
dev-debug-destination:       ; kubectl apply -f ./tests/debug-exporter.yaml
dev-nop-destination:         ; kubectl apply -f ./tests/nop-exporter.yaml
dev-dynamic-destination:     ; kubectl apply -f ./tests/dynamic-exporter.yaml
dev-backpressue-destination: ; kubectl apply -f ./tests/backpressure-exporter.yaml

.PHONY: push-workload-lifecycle-images
push-workload-lifecycle-images:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/nodejs-unsupported-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/nodejs-http-server/unsupported-version.Dockerfile tests/e2e/workload-lifecycle/services/nodejs-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/nodejs-very-old-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/nodejs-http-server/very-old-version.Dockerfile tests/e2e/workload-lifecycle/services/nodejs-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/nodejs-minimum-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/nodejs-http-server/minimum-version.Dockerfile tests/e2e/workload-lifecycle/services/nodejs-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/nodejs-latest-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/nodejs-http-server/latest-version.Dockerfile tests/e2e/workload-lifecycle/services/nodejs-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/nodejs-dockerfile-env:v0.0.1 -f tests/e2e/workload-lifecycle/services/nodejs-http-server/dockerfile-env.Dockerfile tests/e2e/workload-lifecycle/services/nodejs-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/nodejs-manifest-env:v0.0.1 -f tests/e2e/workload-lifecycle/services/nodejs-http-server/manifest-env.Dockerfile tests/e2e/workload-lifecycle/services/nodejs-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/cpp-http-server:v0.0.1 -f tests/e2e/workload-lifecycle/services/cpp-http-server/Dockerfile tests/e2e/workload-lifecycle/services/cpp-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/java-supported-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/java-http-server/java-supported-version.Dockerfile tests/e2e/workload-lifecycle/services/java-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/java-azul:v0.0.1 -f tests/e2e/workload-lifecycle/services/java-http-server/java-azul.Dockerfile tests/e2e/workload-lifecycle/services/java-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/java-supported-docker-env:v0.0.1 -f tests/e2e/workload-lifecycle/services/java-http-server/java-supported-docker-env.Dockerfile tests/e2e/workload-lifecycle/services/java-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/java-supported-manifest-env:v0.0.1 -f tests/e2e/workload-lifecycle/services/java-http-server/java-supported-manifest-env.Dockerfile tests/e2e/workload-lifecycle/services/java-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/java-latest-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/java-http-server/java-latest-version.Dockerfile tests/e2e/workload-lifecycle/services/java-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/java-old-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/java-http-server/java-old-version.Dockerfile tests/e2e/workload-lifecycle/services/java-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/python-latest-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/python-http-server/Dockerfile.python-latest tests/e2e/workload-lifecycle/services/python-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/python-other-agent:v0.0.1 -f tests/e2e/workload-lifecycle/services/python-http-server/Dockerfile.python-other-agent tests/e2e/workload-lifecycle/services/python-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/python-alpine:v0.0.1 -f tests/e2e/workload-lifecycle/services/python-http-server/Dockerfile.python-alpine tests/e2e/workload-lifecycle/services/python-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/python-not-supported:v0.0.1 -f tests/e2e/workload-lifecycle/services/python-http-server/Dockerfile.python-not-supported-version tests/e2e/workload-lifecycle/services/python-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/python-min-version:v0.0.1 -f tests/e2e/workload-lifecycle/services/python-http-server/Dockerfile.python-min-version tests/e2e/workload-lifecycle/services/python-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/dotnet8-musl:v0.0.1 -f tests/e2e/workload-lifecycle/services/dotnet-http-server/net8-musl.Dockerfile tests/e2e/workload-lifecycle/services/dotnet-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/dotnet6-musl:v0.0.1 -f tests/e2e/workload-lifecycle/services/dotnet-http-server/net6-musl.Dockerfile tests/e2e/workload-lifecycle/services/dotnet-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/dotnet8-glibc:v0.0.1 -f tests/e2e/workload-lifecycle/services/dotnet-http-server/net8-glibc.Dockerfile tests/e2e/workload-lifecycle/services/dotnet-http-server
	docker buildx build --push --platform linux/amd64,linux/arm64 -t public.ecr.aws/odigos/dotnet6-glibc:v0.0.1 -f tests/e2e/workload-lifecycle/services/dotnet-http-server/net6-glibc.Dockerfile tests/e2e/workload-lifecycle/services/dotnet-http-server


# ──────────────────────────────────────────────
# ECR & EKS Helpers
# ──────────────────────────────────────────────
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
	make -j $(nproc) build-tag-push-ecr-image/odiglet DOCKERFILE=odiglet/$(DOCKERFILE) SUMMARY="Odiglet for Odigos" DESCRIPTION="Odiglet is the core component of Odigos managing auto-instrumentation. This container requires a root user to run and manage eBPF programs." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j $(nproc) build-tag-push-ecr-image/autoscaler SUMMARY="Autoscaler for Odigos" DESCRIPTION="Autoscaler manages the installation of Odigos components." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j $(nproc) build-tag-push-ecr-image/instrumentor SUMMARY="Instrumentor for Odigos" DESCRIPTION="Instrumentor manages auto-instrumentation for workloads with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j $(nproc) build-tag-push-ecr-image/scheduler SUMMARY="Scheduler for Odigos" DESCRIPTION="Scheduler manages the installation of OpenTelemetry Collectors with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j $(nproc) build-tag-push-ecr-image/collector DOCKERFILE=collector/$(DOCKERFILE) BUILD_DIR=collector SUMMARY="Odigos Collector" DESCRIPTION="The Odigos build of the OpenTelemetry Collector." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	make -j $(nproc) build-tag-push-ecr-image/ui DOCKERFILE=frontend/$(DOCKERFILE) SUMMARY="UI for Odigos" DESCRIPTION="UI provides the frontend webapp for managing an Odigos installation." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)
	echo "✅ Deployed Odigos to EKS, now install the CLI"
