FROM alpine:3
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v0.0.1/otel-dotnet-autoinstrumentation-0.0.1-musl.tar.gz /tmp/
ADD init.sh /tmp
RUN chmod +x /tmp/init.sh
ENTRYPOINT [ "/tmp/init.sh"]