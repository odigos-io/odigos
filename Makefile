.PHONY: build-odiglet
build-odiglet:
	docker build -t keyval/odigos-odiglet:$(TAG) . -f odiglet/Dockerfile

.PHONY: build-autoscaler
build-autoscaler:
	docker build -t keyval/odigos-autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler

.PHONY: build-collector
build-collector:
	docker build -t keyval/odigos-collector:$(TAG) collector -f collector/Dockerfile

.PHONY: build-images
build-images:
	make build-autoscaler TAG=$(TAG)
	docker build -t keyval/odigos-scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler
	make build-odiglet TAG=$(TAG)
	docker build -t keyval/odigos-instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor
	make build-collector TAG=$(TAG)

.PHONY: push-images
push-images:
	docker push keyval/odigos-autoscaler:$(TAG)
	docker push keyval/odigos-scheduler:$(TAG)
	docker push keyval/odigos-instrumentor:$(TAG)
	docker push keyval/odigos-odiglet:$(TAG)
	docker push keyval/odigos-collector:$(TAG)

.PHONY: load-to-kind-odiglet
load-to-kind-odiglet:
	kind load docker-image keyval/odigos-odiglet:$(TAG)

.PHONY: load-to-kind-autoscaler
load-to-kind-autoscaler:
	kind load docker-image keyval/odigos-autoscaler:$(TAG)

.PHONY: load-to-kind-collector
load-to-kind-collector:
	kind load docker-image keyval/odigos-collector:$(TAG)

.PHONY: load-to-kind
load-to-kind:
	make load-to-kind-autoscaler TAG=$(TAG)
	kind load docker-image keyval/odigos-scheduler:$(TAG)
	make load-to-kind-odiglet TAG=$(TAG)
	kind load docker-image keyval/odigos-instrumentor:$(TAG)
	make load-to-kind-collector TAG=$(TAG)

.PHONY: restart-odiglet
restart-odiglet:
	kubectl rollout restart daemonset odiglet -n odigos-system

.PHONY: restart-autoscaler
restart-autoscaler:
	kubectl rollout restart deployment odigos-autoscaler -n odigos-system

.PHONY: restart-collector
restart-collector:
	kubectl rollout restart deployment odigos-gateway -n odigos-system
	# DaemonSets don't directly support the rollout restart command in the same way Deployments do. However, you can achieve the same result by updating an environment variable or any other field in the DaemonSet's pod template, triggering a rolling update of the pods managed by the DaemonSet
	kubectl -n odigos-system patch daemonset odigos-data-collection -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"kubectl.kubernetes.io/restartedAt\":\"$(date +%Y-%m-%dT%H:%M:%S%z)\"}}}}}"

.PHONY: deploy-odiglet
deploy-odiglet:
	make build-odiglet TAG=$(TAG) && make load-to-kind-odiglet TAG=$(TAG) && make restart-odiglet

.PHONY: deploy-autoscaler
deploy-autoscaler:
	make build-autoscaler TAG=$(TAG) && make load-to-kind-autoscaler TAG=$(TAG) && make restart-autoscaler

.PHONY: deploy-collector
deploy-collector:
	make build-collector TAG=$(TAG) && make load-to-kind-collector TAG=$(TAG) && make restart-collector

.PHONY: debug-odiglet
debug-odiglet:
	docker build -t keyval/odigos-odiglet:$(TAG) . -f odiglet/debug.Dockerfile
	kind load docker-image keyval/odigos-odiglet:$(TAG)
	kubectl delete pod -n odigos-system -l app=odiglet
	kubectl wait --for=condition=ready pod -n odigos-system -l app=odiglet
	kubectl port-forward -n odigos-system daemonset/odiglet 2345:2345

,PHONY: e2e-test
e2e-test:
	./e2e-test.sh

ALL_GO_MOD_DIRS := $(shell go list -m -f '{{.Dir}}' | sort)

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%)
go-mod-tidy/%: DIR=$*
go-mod-tidy/%:
	@cd $(DIR) && go mod tidy -compat=1.20

.PHONY: check-clean-work-tree
check-clean-work-tree:
	if [ -n "$$(git status --porcelain)" ]; then \
		git status; \
		git --no-pager diff; \
		echo 'Working tree is not clean, did you forget to run "make go-mod-tidy"?'; \
		exit 1; \
	fi