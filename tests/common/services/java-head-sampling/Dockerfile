FROM eclipse-temurin:17.0.15_6-jdk AS builder
WORKDIR /app
RUN apt-get update && apt-get install -y maven && rm -rf /var/lib/apt/lists/*
COPY pom.xml .
COPY src ./src
RUN mvn clean package -DskipTests -Dmaven.test.skip=true -Dmaven.javadoc.skip=true

FROM eclipse-temurin:17.0.15_6-jre
WORKDIR /app
COPY --from=builder /app/target/head-sampling-1.0.0.jar app.jar
EXPOSE 8080
CMD ["java", "-jar", "app.jar"]
