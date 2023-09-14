.PHONY: build-images
build-images:
	docker build -t keyval/odigos-autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler --build-arg GITHUB_TOKEN=${GITHUB_TOKEN}
	docker build -t keyval/odigos-scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler --build-arg GITHUB_TOKEN=${GITHUB_TOKEN}
	docker build -t keyval/odigos-odiglet:$(TAG) . -f odiglet/Dockerfile --build-arg GITHUB_TOKEN=${GITHUB_TOKEN}
	docker build -t keyval/odigos-instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor --build-arg GITHUB_TOKEN=${GITHUB_TOKEN}

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