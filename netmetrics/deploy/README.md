# netmetrics end-to-end deploy

Turnkey stack proving OBI network metrics enriched to **service names**:

```
OBI (netolly, node-wide)
  -> netmetrics-enricher (shared github.com/odigos-io/odigos/netmetrics)
  -> Prometheus
  -> Grafana  (dashboard "OBI Network Flows — Service Enriched (VM)")
```

## Run (docker compose)

```bash
docker compose up -d --build
# Grafana:    http://<host>:3000   (admin/admin) -> dashboard obi-vm-enriched
# Prometheus: http://<host>:9090
# Enriched:   http://<host>:9100/metrics
```

## Run without compose (docker run)

```bash
# 1) OBI
docker run -d --name obi-net --network=host --pid=host --privileged \
  -v "$PWD/obi-config.yaml":/config/obi-config.yaml:ro \
  -v /sys/kernel/debug:/sys/kernel/debug -v /sys/kernel/tracing:/sys/kernel/tracing \
  otel/ebpf-instrument:v0.9.0 --config=/config/obi-config.yaml

# 2) enricher (build the shared-module image first)
docker build -t odigos/netmetrics-enricher:dev ..
docker run -d --name netmetrics-enricher --network=host --pid=host --privileged \
  -v "$PWD/config.json":/config.json:ro \
  odigos/netmetrics-enricher:dev -obi http://localhost:8999/metrics -listen :9100 -config /config.json

# 3) Prometheus + Grafana: see docker-compose.yml
```

The enricher needs `--pid=host --privileged` to read `/proc/<pid>/fd` for the socket->PID
join, and host network to scrape OBI and serve enriched metrics.

## config.json

- `services`: how a local process maps to a `service.name` (by listen `port`, `comm`, or
  `cmdline` substring). In production this is replaced by the VM agent's PID->Source table
  / odiglet's k8s informer — injected into the shared resolver, same code.
- `peers`: CIDR -> service for the remote endpoint (stand-in for a discovery feed).

## Health
`/healthz` (always ok once serving), `/readyz` (ok after first /proc scan),
`netmetrics_resolved_endpoints` gauge in `/metrics`.
