.PHONY: build run clean docker-build ecr-build-push docker-run test

# Binary name
BINARY_NAME=java-span-gen

# Build the application (compile with Maven)
build:
	mvn clean compile

# Run the application
run: build
	mvn exec:java -Dexec.mainClass="span.gen.JavaSpanGenerator"

# Package the application
package:
	mvn clean package -DskipTests

# Clean build artifacts
clean:
	mvn clean

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME) .

# Build and push to ECR
ecr-build-push:
	./build-and-push.sh

# Run Docker container
docker-run: docker-build
	docker run --rm -e SPANS_PER_SEC=2000 -e SPAN_BYTES=2000 $(BINARY_NAME)

# Run tests
test:
	mvn test

# Install dependencies
deps:
	mvn dependency:resolve

# Format code
fmt:
	mvn spotless:apply

# Lint code
lint:
	mvn checkstyle:check

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Compile Java application with Maven"
	@echo "  run          - Compile and run the application"
	@echo "  package      - Build JAR file"
	@echo "  clean        - Clean Maven build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  ecr-build-push - Build and push Docker image to ECR"
	@echo "  docker-run   - Build and run Docker container"
	@echo "  test         - Run tests"
	@echo "  deps         - Resolve dependencies"
	@echo "  fmt          - Format code with spotless"
	@echo "  lint         - Lint code with checkstyle"
	@echo "  help         - Show this help message"
