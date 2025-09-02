FROM maven:3.8.5-openjdk-17 AS build
COPY src /home/app/src
COPY pom.xml /home/app
RUN mvn -f /home/app/pom.xml clean package

FROM eclipse-temurin:17.0.12_7-jre-jammy
COPY --from=build /home/app/target/*.jar /app/java-supported-version.jar
USER 15000
CMD ["java","-jar", "/app/java-supported-version.jar"]