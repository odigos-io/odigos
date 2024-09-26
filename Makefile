TAG ?= $(shell odigos version --cluster)
ODIGOS_CLI_VERSION ?= $(shell odigos version --cli)
ORG := keyval

.PHONY: build-odiglet
build-odiglet:
	docker build -t $(ORG)/odigos-odiglet:$(TAG) . -f odiglet/Dockerfile --build-arg ODIGOS_VERSION=$(TAG)

.PHONY: verify-nodejs-agent
verify-nodejs-agent:
	@if [ ! -f ../opentelemetry-node/package.json ]; then \
		echo "Error: To build odiglet agents from source, first clone the agents code locally"; \
		exit 1; \
	fi

.PHONY: build-odiglet-with-agents
build-odiglet-with-agents:
	docker build -t $(ORG)/odigos-odiglet:$(TAG) . -f odiglet/Dockerfile --build-arg ODIGOS_VERSION=$(TAG) --build-context nodejs-agent-src=../opentelemetry-node

.PHONY: build-autoscaler
build-autoscaler:
	docker build -t $(ORG)/odigos-autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler

.PHONY: build-instrumentor
build-instrumentor:
	docker build -t $(ORG)/odigos-instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: build-scheduler
build-scheduler:
	docker build -t $(ORG)/odigos-scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler

.PHONY: build-collector
build-collector:
	docker build -t $(ORG)/odigos-collector:$(TAG) collector -f collector/Dockerfile

.PHONY: build-ui
build-ui:
	docker build -t $(ORG)/odigos-ui:$(TAG) . -f frontend/Dockerfile

.PHONY: build-images
build-images:
	# prefer to build timeconsuimg images first to make better use of parallelism
	make -j 6 build-ui build-collector build-odiglet build-autoscaler build-scheduler build-instrumentor TAG=$(TAG)

.PHONY: push-odiglet
push-odiglet:
	docker buildx build --platform linux/amd64,linux/arm64/v8 --push -t $(ORG)/odigos-odiglet:$(TAG) . -f odiglet/Dockerfile

.PHONY: push-autoscaler
push-autoscaler:
	docker buildx build --platform linux/amd64,linux/arm64/v8 --push -t $(ORG)/odigos-autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler

.PHONY: push-instrumentor
push-instrumentor:
	docker buildx build --platform linux/amd64,linux/arm64/v8 --push -t $(ORG)/odigos-instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-scheduler
push-scheduler:
	docker buildx build --platform linux/amd64,linux/arm64/v8 --push -t $(ORG)/odigos-scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler

.PHONY: push-collector
push-collector:
	docker buildx build --platform linux/amd64,linux/arm64/v8 --push -t $(ORG)/odigos-collector:$(TAG) collector -f collector/Dockerfile

.PHONY: push-images
push-images:
	make push-autoscaler TAG=$(TAG)
	make push-scheduler TAG=$(TAG)
	make push-odiglet TAG=$(TAG)
	make push-instrumentor TAG=$(TAG)
	make push-collector TAG=$(TAG)

.PHONY: load-to-kind-odiglet
load-to-kind-odiglet:
	kind load docker-image $(ORG)/odigos-odiglet:$(TAG)

.PHONY: load-to-kind-autoscaler
load-to-kind-autoscaler:
	kind load docker-image $(ORG)/odigos-autoscaler:$(TAG)

.PHONY: load-to-kind-collector
load-to-kind-collector:
	kind load docker-image $(ORG)/odigos-collector:$(TAG)

.PHONY: load-to-kind-instrumentor
load-to-kind-instrumentor:
	kind load docker-image $(ORG)/odigos-instrumentor:$(TAG)

.PHONY: load-to-kind-ui
load-to-kind-ui:
	kind load docker-image $(ORG)/odigos-ui:$(TAG)

.PHONY: load-to-kind-scheduler
load-to-kind-scheduler:
	kind load docker-image $(ORG)/odigos-scheduler:$(TAG)

.PHONY: load-to-kind
load-to-kind:
	make -j 6 load-to-kind-instrumentor load-to-kind-autoscaler load-to-kind-scheduler load-to-kind-odiglet load-to-kind-collector load-to-kind-ui TAG=$(TAG)


.PHONY: restart-ui
restart-ui:
	kubectl rollout restart deployment odigos-ui -n odigos-system

.PHONY: restart-odiglet
restart-odiglet:
	kubectl rollout restart daemonset odiglet -n odigos-system

.PHONY: restart-autoscaler
restart-autoscaler:
	kubectl rollout restart deployment odigos-autoscaler -n odigos-system

.PHONY: restart-instrumentor
restart-instrumentor:
	kubectl rollout restart deployment odigos-instrumentor -n odigos-system

.PHONY: restart-scheduler
restart-scheduler:
	kubectl rollout restart deployment odigos-scheduler -n odigos-system


.PHONY: restart-collector
restart-collector:
	kubectl rollout restart deployment odigos-gateway -n odigos-system
	# DaemonSets don't directly support the rollout restart command in the same way Deployments do. However, you can achieve the same result by updating an environment variable or any other field in the DaemonSet's pod template, triggering a rolling update of the pods managed by the DaemonSet
	kubectl -n odigos-system patch daemonset odigos-data-collection -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"kubectl.kubernetes.io/restartedAt\":\"$(date +%Y-%m-%dT%H:%M:%S%z)\"}}}}}"

.PHONY: deploy-odiglet
deploy-odiglet:
	make build-odiglet TAG=$(TAG) && make load-to-kind-odiglet TAG=$(TAG) && make restart-odiglet

# Use this target to deploy odiglet with local clones of the agents.
# To work, the agents must be cloned in the same directory as the odigos (e.g. in '../opentelemetry-node')
# There you can make code changes to the agents and deploy them with the odiglet.
.PHONY: deploy-odiglet-with-agents
deploy-odiglet-with-agents: verify-nodejs-agent build-odiglet-with-agents load-to-kind-odiglet restart-odiglet

.PHONY: deploy-autoscaler
deploy-autoscaler:
	make build-autoscaler TAG=$(TAG) && make load-to-kind-autoscaler TAG=$(TAG) && make restart-autoscaler

.PHONY: deploy-collector
deploy-collector:
	make build-collector TAG=$(TAG) && make load-to-kind-collector TAG=$(TAG) && make restart-collector

.PHONY: deploy-instrumentor
deploy-instrumentor:
	make build-instrumentor TAG=$(TAG) && make load-to-kind-instrumentor TAG=$(TAG) && make restart-instrumentor

.PHONY: deploy-ui
deploy-ui:
	make build-ui TAG=$(TAG) && make load-to-kind-ui TAG=$(TAG) && make restart-ui

.PHONY: deploy-scheduler
deploy-scheduler:
	make build-scheduler TAG=$(TAG) && make load-to-kind-scheduler TAG=$(TAG) && make restart-scheduler


.PHONY: debug-odiglet
debug-odiglet:
	docker build -t $(ORG)/odigos-odiglet:$(TAG) . -f odiglet/debug.Dockerfile
	kind load docker-image $(ORG)/odigos-odiglet:$(TAG)
	kubectl delete pod -n odigos-system -l app.kubernetes.io/name=odiglet
	kubectl wait --for=condition=ready pod -n odigos-system -l app.kubernetes.io/name=odiglet --timeout=180s
	kubectl port-forward -n odigos-system daemonset/odiglet 2345:2345

.PHONY: deploy
deploy: deploy-odiglet deploy-autoscaler deploy-collector deploy-instrumentor deploy-scheduler

,PHONY: e2e-test
e2e-test:
	./e2e-test.sh

ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%)
go-mod-tidy/%: DIR=$*
go-mod-tidy/%:
	@cd $(DIR) && go mod tidy -compat=1.21

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
	cd ./cli ; go run -tags=embed_manifests . install --version $(ODIGOS_CLI_VERSION)

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
	kind create cluster

.PHONY: dev-tests-setup
dev-tests-setup: TAG := e2e-test
dev-tests-setup: dev-tests-kind-cluster cli-build build-images load-to-kind

# Use this target to avoid rebuilding the images if all that changed is the e2e test code
.PHONY: dev-tests-setup-no-build
dev-tests-setup-no-build: TAG := e2e-test
dev-tests-setup-no-build: dev-tests-kind-cluster load-to-kind
