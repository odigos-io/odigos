.PHONY: build-images
build-images:
	docker build -t ghcr.io/keyval-dev/odigos/autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler
	docker build -t ghcr.io/keyval-dev/odigos/scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler
	docker build -t ghcr.io/keyval-dev/odigos/ui:$(TAG) ui/ -f ui/Dockerfile
	docker build -t ghcr.io/keyval-dev/odigos/odiglet:$(TAG) . -f odiglet/Dockerfile
	docker build -t ghcr.io/keyval-dev/odigos/instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-images
push-images:
	docker push ghcr.io/keyval-dev/odigos/autoscaler:$(TAG)
	docker push ghcr.io/keyval-dev/odigos/scheduler:$(TAG)
	docker push ghcr.io/keyval-dev/odigos/ui:$(TAG)
	docker push ghcr.io/keyval-dev/odigos/instrumentor:$(TAG)
	docker push ghcr.io/keyval-dev/odigos/odiglet:$(TAG)

.PHONY: load-to-kind
load-to-kind:
	kind load docker-image ghcr.io/keyval-dev/odigos/autoscaler:$(TAG)
	kind load docker-image ghcr.io/keyval-dev/odigos/scheduler:$(TAG)
	kind load docker-image ghcr.io/keyval-dev/odigos/ui:$(TAG)
	kind load docker-image ghcr.io/keyval-dev/odigos/odiglet:$(TAG)
	kind load docker-image ghcr.io/keyval-dev/odigos/instrumentor:$(TAG)