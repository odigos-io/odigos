# CORE-36 local GCS emulator validation

This evidence was captured from a local `fake-gcs-server` environment, not Google Cloud. The temporary cloud project was deleted after the user requested local validation.

The built Odigos collector at commit `96581d412bafe3da447da6604f2ee2f9648df655` received 12 synthetic spans and 12 log records over OTLP/HTTP. A configuration produced by `common/config.GoogleCloudStorage.ModifyConfig` exported both signals as OTLP JSON and protobuf with default exporter queue/retry settings. The batch processor used a maximum of five records per batch.

The emulator ran with filesystem persistence, exposed only on `127.0.0.1:15443`, using:

```sh
docker run -d --name core36-gcs-e2e \
  -p 127.0.0.1:15443:4443 \
  -v /private/tmp/core36-gcs-env/storage:/storage \
  fsouza/fake-gcs-server@sha256:797ce226d62f947c009dc40246b30cfb456b8473d8241407f9d6f2c04e4d69ef \
  -scheme http -backend filesystem \
  -external-url http://127.0.0.1:15443 -public-host 127.0.0.1:15443
```

The collector used `STORAGE_EMULATOR_HOST=http://127.0.0.1:15443` and the private local bucket `core36-gcs-e2e`. To replay the same test after starting the emulator and creating this bucket, run the built collector with `local-collector-config.json`, then run `python3 local-send-telemetry.py`:

```sh
curl -fsS -X POST -H 'Content-Type: application/json' \
  --data '{"name":"core36-gcs-e2e"}' \
  'http://127.0.0.1:15443/storage/v1/b?project=local-core36'
STORAGE_EMULATOR_HOST=http://127.0.0.1:15443 odigosotelcol --config=local-collector-config.json
# In a second terminal, from this evidence directory:
python3 local-send-telemetry.py
```

All 12 persisted objects were downloaded through the storage API and decoded with the Collector pdata JSON/protobuf unmarshallers. Each signal/format contained exactly 12 unique records in three objects with batch sizes 5, 5 and 2. Verification checked service.name, record contents, timestamps and uniqueness. `local-readback-summary.json` contains the complete object manifest and totals; `local-traces.json` and `local-logs.json` are representative decoded objects.

Exporter unit tests additionally exercise transient HTTP failures and retry, permission rejection, queueing, lifecycle and legacy object-key layouts. Run them with `go test -race ./...` from the exporter module.

The emulator does not validate cloud IAM or Workload Identity. The earlier authenticated cloud upload attempt returned 403 for missing `storage.objects.create`. Local results do not claim successful Google Cloud ingestion.
