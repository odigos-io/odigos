# Pin a recent Prometheus 3.x so we can use --web.enable-otlp-receiver
FROM prom/prometheus:v3.7.3 AS validator

WORKDIR /work
COPY prometheus.yml .

# Validate config (will fail the build on errors)
RUN promtool check config /work/prometheus.yml

# Final image
FROM prom/prometheus:v3.7.3

# Copy only validated config
COPY --from=validator /work/prometheus.yml /etc/prometheus/prometheus.yml

# Run as the non-root user shipped by the base image
USER 65534

# Notes:
# - --web.enable-otlp-receiver exposes HTTP OTLP ingest at /api/v1/otlp/v1/metrics on port 9090
# - --enable-feature=remote-write-receiver enables Prometheus Remote Write API at /api/v1/write on port 9090
# - --storage.tsdb.path points TSDB to an in-container path (no volume -> non-persistent)
# - --storage.tsdb.retention.time keeps data short-lived to minimize disk use
# - --storage.tsdb.retention.size limits storage size to 2 GB
CMD [ \
  "--config.file=/etc/prometheus/prometheus.yml", \
  "--web.enable-otlp-receiver", \
  "--enable-feature=remote-write-receiver", \
  "--storage.tsdb.path=/tmp/prom-tsdb", \
  "--storage.tsdb.retention.time=24h", \
  "--storage.tsdb.retention.size=2GB", \
  "--web.enable-lifecycle" \
]
