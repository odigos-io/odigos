.PHONY: build run clean docker-build ecr-build-push docker-run test

# Binary name
BINARY_NAME=python-span-gen

# Build the application (install dependencies)
build:
	pip install -r requirements.txt

# Run the application
run: build
	python app.py

# Clean build artifacts
clean:
	rm -rf __pycache__ *.pyc .pytest_cache

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME) .

# Build and push to ECR
ecr-build-push:
	./build-and-push.sh

# Run Docker container
docker-run: docker-build
	docker run --rm -e SPANS_PER_SEC=1000 -e SPAN_BYTES=500 $(BINARY_NAME)

# Run tests
test:
	python -m pytest

# Install dependencies
deps:
	pip install -r requirements.txt

# Format code
fmt:
	black app.py

# Lint code
lint:
	flake8 app.py

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Install Python dependencies"
	@echo "  run          - Install dependencies and run the application"
	@echo "  clean        - Clean Python cache files"
	@echo "  docker-build - Build Docker image"
	@echo "  ecr-build-push - Build and push Docker image to ECR"
	@echo "  docker-run   - Build and run Docker container"
	@echo "  test         - Run tests"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code with black"
	@echo "  lint         - Lint code with flake8"
	@echo "  help         - Show this help message"
