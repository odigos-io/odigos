FROM maven:3.8.5-openjdk-11 AS build
COPY src /home/app/src
COPY pom11.xml /home/app
RUN mvn -f /home/app/pom11.xml clean package

FROM eclipse-temurin:11.0.27_6-jre-jammy
COPY --from=build /home/app/target/*.jar /app/java-old-version.jar
USER 15000
CMD ["java","-jar", "/app/java-old-version.jar"]