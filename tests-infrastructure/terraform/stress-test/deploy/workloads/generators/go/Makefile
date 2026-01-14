.PHONY: build run clean docker-build docker-run test

# Binary name
BINARY_NAME=go-span-gen

# Build the application
build:
	go mod tidy
	go build -o $(BINARY_NAME) .

# Run the application
run: build
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME) .

# Build and push to ECR
ecr-build-push:
	./build-and-push.sh

# Run Docker container
docker-run: docker-build
	docker run --rm -e SPANS_PER_SEC=1000 -e SPAN_BYTES=5000 $(BINARY_NAME)

# Run tests
test:
	go test ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Build and run the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  ecr-build-push - Build and push Docker image to ECR"
	@echo "  docker-run   - Build and run Docker container"
	@echo "  test         - Run tests"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  help         - Show this help message"
