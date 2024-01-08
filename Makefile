.PHONY: build-odiglet
build-odiglet:
	docker build -t keyval/odigos-odiglet:$(TAG) . -f odiglet/Dockerfile

.PHONY: build-images
build-images:
	docker build -t keyval/odigos-autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler
	docker build -t keyval/odigos-scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler
	make build-odiglet TAG=$(TAG)
	docker build -t keyval/odigos-instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-images
push-images:
	docker push keyval/odigos-autoscaler:$(TAG)
	docker push keyval/odigos-scheduler:$(TAG)
	docker push keyval/odigos-instrumentor:$(TAG)
	docker push keyval/odigos-odiglet:$(TAG)

.PHONY: load-to-kind-odiglet
load-to-kind-odiglet:
	kind load docker-image keyval/odigos-odiglet:$(TAG)

.PHONY: load-to-kind
load-to-kind:
	kind load docker-image keyval/odigos-autoscaler:$(TAG)
	kind load docker-image keyval/odigos-scheduler:$(TAG)
	make load-to-kind-odiglet TAG=$(TAG)
	kind load docker-image keyval/odigos-instrumentor:$(TAG)

.PHONY: restart-odiglet
restart-odiglet:
	kubectl rollout restart daemonset odiglet -n odigos-system

.PHONY: deploy-odiglet
deploy-odiglet:
	make build-odiglet TAG=$(TAG) && make load-to-kind-odiglet TAG=$(TAG) && make restart-odiglet

.PHONY: debug-odiglet
debug-odiglet:
	docker build -t keyval/odigos-odiglet:$(TAG) . -f odiglet/debug.Dockerfile
	kind load docker-image keyval/odigos-odiglet:$(TAG)
	kubectl delete pod -n odigos-system -l app=odiglet
	kubectl wait --for=condition=ready pod -n odigos-system -l app=odiglet
	kubectl port-forward -n odigos-system daemonset/odiglet 2345:2345