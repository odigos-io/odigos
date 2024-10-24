FROM maven:3.8.5-openjdk-17 AS build
COPY src /home/app/src
COPY pom.xml /home/app
RUN mvn -f /home/app/pom.xml clean package

FROM azul/zulu-openjdk-alpine:17.0.12_7-jre
COPY --from=build /home/app/target/*.jar /app/java-azul.jar
USER 15000
CMD ["java", "-XX:+UnlockExperimentalVMOptions", "-XX:+UseZGC", "-jar", "/app/java-azul.jar"]
