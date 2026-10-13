# Google Cloud Storage Exporter

Archives traces and logs as one OTLP object per export batch in an existing Google Cloud Storage bucket. Metrics and profiles are not supported.

```yaml
exporters:
  googlecloudstorage:
    gcs:
      bucket: odigos-otlp
      prefix: odigos
      partition: minute
      file_prefix: ""
    marshaler_name: otlp_json
```

The exporter uses [Application Default Credentials](https://cloud.google.com/docs/authentication/provide-credentials-adc), including GKE Workload Identity Federation, `GOOGLE_APPLICATION_CREDENTIALS`, or local ADC. Grant the gateway identity `storage.objects.create` on the bucket. Reading objects for validation also requires `storage.objects.get` and `storage.objects.list`.

`gcs.bucket` is required. `partition` accepts `minute` (default) or `hour`. `prefix` and `file_prefix` default to empty. `marshaler_name` accepts `otlp_json` (default), `otlp_proto`, and the legacy protobuf alias `otlp`.

Object names retain the existing layout: `<prefix>/year=YYYY/month=MM/day=DD/hour=HH[/minute=MM]/<file_prefix><traces|logs>_<random>.<json|proto>`. Empty prefixes retain the leading slash for compatibility. JSON and protobuf are OTLP payloads, not newline-delimited JSON.

Collector `timeout`, `sending_queue`, and `retry_on_failure` settings use the standard exporter helper defaults. The bounded queue buffers export requests, and transient failures are retried. The storage client also retries resumable uploads. A successful export means the object writer closed successfully; clients close during exporter shutdown.

Run `go test ./...` for configuration, lifecycle, batched upload/readback, OTLP formats and transient failure tests against a local storage endpoint. Real GCS validation additionally requires an authenticated project and bucket. In Cloud Console, download traces and logs objects and inspect their decoded payloads.
