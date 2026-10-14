# Odigos Google Cloud Pub/Sub exporter patch

This module reuses the upstream v0.159.0 implementation and tests under the original module path. It is selected by `collector/builder-config.yaml` through a local replacement.

The only behavior change preserves the original OTLP batch before ordering splits move resource groups. Without that copy, a failed ordered publish empties the original request, and exporterhelper retries successfully without republishing its telemetry. `ordering_retry_test.go` reproduces the failure for traces, metrics, and logs. Retries may republish already accepted keys, so downstream duplicate handling is still required.

Remove this replacement when the pinned upstream exporter preserves retry data for ordered batches.

Upstream source: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.159.0/exporter/googlecloudpubsubexporter
