# Build stage
FROM maven:3.9.6-eclipse-temurin-17 AS builder

WORKDIR /app

# Copy pom.xml first for better layer caching
COPY pom.xml .

# Download dependencies
RUN mvn dependency:go-offline

# Copy source code
COPY src ./src

# Build the application
RUN mvn clean package -DskipTests

# Final stage - use eclipse-temurin:17-jre which supports both platforms
FROM eclipse-temurin:17-jre

WORKDIR /app

# Copy the built JAR from builder stage
COPY --from=builder /app/target/java-span-gen-1.0.0.jar ./app.jar

# Set environment variables
ENV OTEL_SERVICE_NAME=java-span-generator
ENV OTEL_RESOURCE_ATTRIBUTES=service.name=java-span-generator,service.version=v1.0.0
ENV SPANS_PER_SEC=1000
ENV SPAN_BYTES=1000

# Run the application
CMD ["java", "-jar", "app.jar"]
