# otlpproxygrpc exporter

An OTLP/gRPC exporter identical to the upstream `otlp` exporter, plus a
`proxy_url` option. The upstream `configgrpc` client has no proxy field, so
gRPC exporters cannot otherwise be routed through an egress proxy via config
(only via `HTTPS_PROXY` env, which requires a process restart). This exporter
injects an HTTP CONNECT dialer (`grpc.WithContextDialer`) so the gRPC
connection tunnels through the proxy, configurable at runtime and applied on
collector config reload — no restart.

```yaml
exporters:
  otlpproxygrpc/mybackend:
    endpoint: my-backend:4317
    proxy_url: http://proxy.corp.local:8080   # http/https/socks5, optional user:pass@
    tls:
      ca_file: /etc/odigos/proxy-ca.pem        # for TLS-terminating corp proxies
```

When `proxy_url` is empty it behaves exactly like the stock `otlp` exporter.
TLS to the backend is preserved end-to-end (the proxy only tunnels); `tls.ca_file`
is for trusting a TLS-terminating proxy's certificate without disabling verification.
