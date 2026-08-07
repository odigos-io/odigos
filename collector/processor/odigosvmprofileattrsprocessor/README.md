# Odigos VM Profile Attributes Processor

Enriches OpenTelemetry profile resources using a PID→attribute map streamed from the VM agent over `/var/exchange/exchange.sock` (`GET_PROF_ATTR`, `R<pid>:attrs` / `U<pid>`).

Only PIDs registered by the VM agent (enabled Odigos sources) are exported. Unregistered PIDs are removed from each batch. Empty batches are not forwarded to exporters (avoids Pyroscope `InvalidArgument: missing resource profiles`).

`service.name` is set on the resource and copied onto every profile sample (Grafana Pyroscope reads sample attributes for the `service_name` label). The OTLP profiles dictionary from the receiver is preserved on export.

```yaml
processors:
  odigosvmprofileattrsprocessor/profiles-vm: {}
```

Optional:

```yaml
processors:
  odigosvmprofileattrsprocessor/profiles-vm:
    socket_path: /var/exchange/exchange.sock
```
