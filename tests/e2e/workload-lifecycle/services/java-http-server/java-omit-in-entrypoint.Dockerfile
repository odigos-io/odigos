# Stage 1: Build the Java application using Maven
FROM maven:latest AS build
COPY src /home/app/src
COPY pom.xml /home/app
RUN mvn -f /home/app/pom.xml clean package

# Stage 2: Prepare the runtime environment
FROM eclipse-temurin:21

# Copy the built application from the Maven build stage
COPY --from=build /home/app/target/*.jar /app/app.jar

# Rename Java binary to remove "java" from the execution command
RUN mv /opt/java/openjdk/bin/java /opt/java/openjdk/bin/app_runner

# Set user for security
USER 15000

# Run the application without "java" keyword
ENTRYPOINT ["/opt/java/openjdk/bin/app_runner", "-jar", "/app/app.jar"]
