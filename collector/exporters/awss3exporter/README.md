# Odigos S3 exporter compatibility patch

Based on upstream awss3exporter v0.159.0 (1120cc0714bf4d14600105e5673c3b00fe87467a).
The upstream upload manager, marshalers, batching, retries, and multipart implementation are retained.

Odigos additions under `s3uploader`:
- `use_static_credentials`: require static keys and prevent fallback to ambient credentials.
- `access_key_id` and `secret_access_key`: optional paired static credentials, scoped to one exporter. Omit both to retain the AWS credential chain and role authentication.
- `tls`: optional collector TLS client configuration, including `ca_pem`.
- `request_checksum_calculation`: `when_supported` (AWS SDK default) or `when_required` for S3 gateways that reject optional checksums.

These additions should be contributed upstream and the local replacement removed once available in a compatible release.
