.PHONY: build run clean docker-build ecr-build-push docker-run test

# Binary name
BINARY_NAME=node-span-gen

# Build the application (install dependencies)
build:
	npm install

# Run the application
run: build
	node app.js

# Clean build artifacts
clean:
	rm -rf node_modules package-lock.json

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME) .

# Build and push to ECR
ecr-build-push:
	./build-and-push.sh

# Run Docker container
docker-run: docker-build
	docker run --rm -e SPANS_PER_SEC=3000 -e SPAN_BYTES=2000 $(BINARY_NAME)

# Run tests
test:
	npm test

# Install dependencies
deps:
	npm install

# Format code
fmt:
	npx prettier --write app.js

# Lint code
lint:
	npx eslint app.js

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Install Node.js dependencies"
	@echo "  run          - Install dependencies and run the application"
	@echo "  clean        - Clean node_modules and package-lock.json"
	@echo "  docker-build - Build Docker image"
	@echo "  ecr-build-push - Build and push Docker image to ECR"
	@echo "  docker-run   - Build and run Docker container"
	@echo "  test         - Run tests"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code with prettier"
	@echo "  lint         - Lint code with eslint"
	@echo "  help         - Show this help message"
