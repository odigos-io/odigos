TAG ?= $(shell odigos version --cluster)
ODIGOS_CLI_VERSION ?= $(shell odigos version --cli)
ORG ?= keyval
GOLANGCI_LINT_VERSION ?= v1.63.4
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
GO_MODULES := $(shell find . -type f -name "go.mod" -not -path "*/vendor/*" -exec dirname {} \; | grep -v "licenses")
LINT_CMD = golangci-lint run -c ../.golangci.yml
ifdef FIX_LINT
    LINT_CMD += --fix
endif
DOCKERFILE=Dockerfile
IMG_SUFFIX=
BUILD_DIR=.

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

build-image/%:
	docker build -t $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG) $(BUILD_DIR) -f $(DOCKERFILE) \
	--build-arg SERVICE_NAME="$*" \
	--build-arg ODIGOS_VERSION=$(TAG) \
	--build-arg VERSION=$(TAG) \
	--build-arg RELEASE=$(TAG) \
	--build-arg SUMMARY="$(SUMMARY)" \
	--build-arg DESCRIPTION="$(DESCRIPTION)"

.PHONY: build-odiglet
build-odiglet:
	$(MAKE) build-image/odiglet DOCKERFILE=odiglet/$(DOCKERFILE) SUMMARY="Odiglet for Odigos" DESCRIPTION="Odiglet is the core component of Odigos managing auto-instrumentation. This container requires a root user to run and manage eBPF programs." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: build-autoscaler
build-autoscaler:
	$(MAKE) build-image/autoscaler SUMMARY="Autoscaler for Odigos" DESCRIPTION="Autoscaler manages the installation of Odigos components." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: build-instrumentor
build-instrumentor:
	$(MAKE) build-image/instrumentor SUMMARY="Instrumentor for Odigos" DESCRIPTION="Instrumentor manages auto-instrumentation for workloads with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: build-scheduler
build-scheduler:
	$(MAKE) build-image/scheduler SUMMARY="Scheduler for Odigos" DESCRIPTION="Scheduler manages the installation of OpenTelemetry Collectors with Odigos." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: build-collector
build-collector:
	$(MAKE) build-image/collector DOCKERFILE=collector/$(DOCKERFILE) BUILD_DIR=collector SUMMARY="Odigos Collector" DESCRIPTION="The Odigos build of the OpenTelemetry Collector." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: build-ui
build-ui:
	$(MAKE) build-image/ui DOCKERFILE=frontend/$(DOCKERFILE) SUMMARY="UI for Odigos" DESCRIPTION="UI provides the frontend webapp for managing an Odigos installation." TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX)

.PHONY: build-odiglet-with-agents
build-odiglet-with-agents:
	docker build -t $(ORG)/odigos-odiglet$(IMG_SUFFIX):$(TAG) . -f odiglet/$(DOCKERFILE) --build-arg ODIGOS_VERSION=$(TAG) --build-context nodejs-agent-src=../opentelemetry-node \
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

.PHONY: build-images
build-images:
	# prefer to build timeconsuimg images first to make better use of parallelism
	make -j 3 build-ui build-collector build-odiglet build-autoscaler build-scheduler build-instrumentor TAG=$(TAG) ORG=$(ORG) IMG_SUFFIX=$(IMG_SUFFIX) DOCKERFILE=$(DOCKERFILE)

.PHONY: build-images-rhel
build-images-rhel:
	$(MAKE) build-images IMG_SUFFIX=-ubi9 DOCKERFILE=Dockerfile.rhel TAG=$(TAG) ORG=$(ORG)

push-image/%:
	docker buildx build --platform linux/amd64,linux/arm64/v8 -t $(ORG)/odigos-$*$(IMG_SUFFIX):$(TAG) $(BUILD_DIR) -f $(DOCKERFILE) \
	--build-arg SERVICE_NAME="$*" \
	--build-arg VERSION=$(TAG) \
	--build-arg RELEASE=$(TAG) \
	--build-arg SUMMARY="$(SUMMARY)" \
	--build-arg DESCRIPTION="$(DESCRIPTION)"

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
	make -j 6 load-to-kind-instrumentor load-to-kind-autoscaler load-to-kind-scheduler load-to-kind-odiglet load-to-kind-collector load-to-kind-ui ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX) DOCKERFILE=$(DOCKERFILE)

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

deploy-%:
	make build-$* ORG=$(ORG) TAG=$(TAG) DOCKERFILE=$(DOCKERFILE) IMG_SUFFIX=$(IMG_SUFFIX) && make load-to-kind-$* ORG=$(ORG) TAG=$(TAG) IMG_SUFFIX=$(IMG_SUFFIX) && make restart-$*

.PHONY: deploy
deploy:
	make deploy-odiglet && make deploy-autoscaler && make deploy-collector && make deploy-instrumentor && make deploy-scheduler && make deploy-ui

# Use this target to deploy odiglet with local clones of the agents.
# To work, the agents must be cloned in the same directory as the odigos (e.g. in '../opentelemetry-node')
# There you can make code changes to the agents and deploy them with the odiglet.
.PHONY: deploy-odiglet-with-agents
deploy-odiglet-with-agents: verify-nodejs-agent build-odiglet-with-agents load-to-kind-odiglet restart-odiglet

.PHONY: debug-odiglet
debug-odiglet:
	docker build -t $(ORG)/odigos-odiglet:$(TAG) . -f odiglet/debug.Dockerfile
	kind load docker-image $(ORG)/odigos-odiglet:$(TAG)
	kubectl delete pod -n odigos-system -l app.kubernetes.io/name=odiglet
	kubectl wait --for=condition=ready pod -n odigos-system -l app.kubernetes.io/name=odiglet --timeout=180s
	kubectl port-forward -n odigos-system daemonset/odiglet 2345:2345

,PHONY: e2e-test
e2e-test:
	./e2e-test.sh

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

GO_AUTO_VERSION=v0.19.0-alpha
UNSTABLE_COLLECTOR_VERSION=v0.118.0
STABLE_COLLECTOR_VERSION=v1.24.0
STABLE_OTEL_GO_VERSION=v1.34.0
UNSTABLE_OTEL_GO_VERSION=v0.59.0

.PHONY: update-otel
update-otel:
	$(MAKE) update-dep MODULE=go.opentelemetry.io/auto VERSION=$(GO_AUTO_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/cmd/mdatagen VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/component VERSION=$(UNSTABLE_COLLECTOR_VERSION)
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
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/extension VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/extension/zpagesextension VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/otelcol VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/pdata VERSION=$(STABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor/batchprocessor VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor/memorylimiterprocessor VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/processor/processortest VERSION=$(UNSTABLE_COLLECTOR_VERSION)
	$(MAKE) update-dep MODULE=go.opentelemetry.io/collector/receiver VERSION=$(UNSTABLE_COLLECTOR_VERSION)
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
	cd ./cli ; go run -tags=embed_manifests . install --version $(ODIGOS_CLI_VERSION) --nowait $(FLAGS)

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
	@echo "Building the cli executable for tests"
	cd cli && go build -tags=embed_manifests -o odigos .

.PHONY: cli-diagnose
cli-diagnose:
	@echo "Diagnosing cluster data for debugging"
	cd ./cli ; go run -tags=embed_manifests . diagnose

.PHONY: helm-install
helm-install:
	@echo "Installing odigos using helm"
	helm upgrade --install odigos  ./helm/odigos --create-namespace --namespace odigos-system --set image.tag=$(ODIGOS_CLI_VERSION)
	kubectl label namespace odigos-system odigos.io/system-object="true"

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
dev-tests-setup: dev-tests-kind-cluster cli-build build-images load-to-kind

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

.PHONY: dev-add-backpressue-destination
dev-backpressue-destination:
	kubectl apply -f ./tests/backpressure-exporter.yaml
