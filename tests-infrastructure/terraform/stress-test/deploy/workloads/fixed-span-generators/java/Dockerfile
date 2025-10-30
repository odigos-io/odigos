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
COPY --from=builder /app/target/java-fixed-span-gen-1.0.0.jar ./app.jar

# Set environment variables
ENV TOTAL_SPANS=10000
ENV SPAN_BYTES=2000

# Run the application
CMD ["java", "-jar", "app.jar"]
