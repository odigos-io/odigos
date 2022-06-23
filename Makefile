.PHONY: build-images
build-images:
	docker build -t keyval/odigos-autoscaler:$(TAG) . --build-arg SERVICE_NAME=autoscaler
	docker build -t keyval/odigos-scheduler:$(TAG) . --build-arg SERVICE_NAME=scheduler
	docker build -t keyval/odigos-lang-detector:$(TAG) . --build-arg SERVICE_NAME=langDetector
	docker build -t keyval/odigos-ui:$(TAG) ui/ -f ui/Dockerfile
	docker build -t keyval/odigos-instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-images
push-images:
	docker push keyval/odigos-autoscaler:$(TAG)
	docker push keyval/odigos-scheduler:$(TAG)
	docker push keyval/odigos-lang-detector:$(TAG)
	docker push keyval/odigos-ui:$(TAG)
	docker push keyval/odigos-instrumentor:$(TAG)