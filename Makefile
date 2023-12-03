.PHONY: build-images
build-images:
	docker build -t keyval/odigos-autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler
	docker build -t keyval/odigos-scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler
	docker build -t keyval/odigos-odiglet:$(TAG) . -f odiglet/Dockerfile
	docker build -t keyval/odigos-instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-images
push-images:
	docker push keyval/odigos-autoscaler:$(TAG)
	docker push keyval/odigos-scheduler:$(TAG)
	docker push keyval/odigos-instrumentor:$(TAG)
	docker push keyval/odigos-odiglet:$(TAG)

.PHONY: load-to-kind
load-to-kind:
	kind load docker-image keyval/odigos-autoscaler:$(TAG)
	kind load docker-image keyval/odigos-scheduler:$(TAG)
	kind load docker-image keyval/odigos-odiglet:$(TAG)
	kind load docker-image keyval/odigos-instrumentor:$(TAG)

.PHONY: debug-odiglet
debug-odiglet:
	docker build -t keyval/odigos-odiglet:$(TAG) . -f odiglet/debug.Dockerfile
	kind load docker-image keyval/odigos-odiglet:$(TAG)
	kubectl rollout restart daemonset odiglet -n odigos-system
	kubectl wait --for=condition=ready pod -l app=odiglet -n odigos-system
	$(eval POD_NAME := $(shell kubectl get pods -n odigos-system --field-selector=status.phase=Running -l app=odiglet -o jsonpath='{.items[0].metadata.name}'))
	kubectl port-forward -n odigos-system $(POD_NAME) 2345:2345