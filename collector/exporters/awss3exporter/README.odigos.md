Based on upstream awss3exporter v0.159.0, retaining its upload, retry, partition, compression and marshaling implementations.

Odigos additions under `s3uploader`:

- `access_key_id` and `secret_access_key`: opaque, exporter-scoped static credentials. Both are required together. Omitting both preserves the AWS default credential chain.
- `checksum_when_required`: forwards the AWS SDK checksum policy to both the S3 client and transfer manager. Defaults to false, preserving AWS behavior.

Spaces uses destination Secret placeholders, HTTPS regional origin endpoints and UUIDv7 object keys. No process-wide credential environment variables are changed. Remove the local replacement once upstream supports these settings.

Run `AWS_EC2_METADATA_DISABLED=true go test ./...` in this directory. Regenerate metadata through the module-scoped workflow in `collector/Makefile`, and rebuild the distribution using `make genodigoscol` from `collector`.
