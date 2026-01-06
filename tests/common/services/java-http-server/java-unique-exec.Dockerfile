# Stage 1: Build the Java application using Maven
FROM maven:3.9.9-eclipse-temurin-17 AS build
COPY src /home/app/src
COPY pom.xml /home/app
RUN mvn -f /home/app/pom.xml clean package

# Stage 2: Prepare the runtime environment
FROM eclipse-temurin:17.0.13_11-jre-jammy

# Copy the built application from the Maven build stage
COPY --from=build /home/app/target/*.jar /app/app.jar

# Move and rename Java binary
RUN mv /opt/java/openjdk/bin/java /usr/local/bin/app_runner

# Set user for security
USER 15000

# Run the application with the new binary name
ENTRYPOINT ["/usr/local/bin/app_runner", "-jar", "/app/app.jar"]
