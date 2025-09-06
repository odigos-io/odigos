FROM maven:latest AS build
COPY src /home/app/src
COPY pom.xml /home/app
RUN mvn -f /home/app/pom.xml clean package

FROM eclipse-temurin:latest
COPY --from=build /home/app/target/*.jar /app/java-latest-version.jar
USER 15000
CMD ["java","-jar", "/app/java-latest-version.jar"]