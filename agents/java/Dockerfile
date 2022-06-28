FROM busybox
ARG OTEL_VERSIN=1.15.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$OTEL_VERSIN/opentelemetry-javaagent.jar /javaagent.jar
RUN chmod -R go+r /javaagent.jar