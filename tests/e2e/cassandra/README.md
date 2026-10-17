# Apache Cassandra validation (CORE-1772)

The screenshots in `evidence/` show actual CQL query output rendered in a local report in Chrome. Apache Cassandra has no native APM/telemetry UI. They are not cloud dashboard screenshots. Raw query transcripts are retained beside them.

On 12 October 2026 the compiled Odigos collector v0.159.0 received SDK telemetry from checkout-service and inventory-service over its OTLP HTTP receiver, used the batch processor, and stored six spans and eight logs in Cassandra 5.0.9 over verified TLS with password authentication. Logs include correlated request records and uncorrelated structured background records. Zstd table compression was used. No Kubernetes auto-instrumentation deployment or managed Cassandra provider was tested.

## Reproduce the collector delivery path

Start a disposable Cassandra 5 server with native transport published on localhost:19042, password authentication enabled, and mandatory client TLS. Provision a server certificate with localhost/127.0.0.1 SANs and a matching CA. The fixture uses the default Cassandra development role; use a dedicated role with the documented permissions elsewhere.

From `common`, render the integration's actual destination helper:

```sh
GOWORK=off go run ../tests/e2e/cassandra/testdata/render-config.go > /tmp/cassandra-collector.json
```

From `collector/odigosotelcol`, build and start the collector. The renderer uses destination ID core1772, keyspace core1772_e2e and OTLP HTTP port 14318. Supply the fixture password and CA through these environment variables:

```sh
GOWORK=off go build -ldflags=-checklinkname=0 -o /tmp/cassandra-odigoscol .
export ODIGOS_DEST_core1772_CASSANDRA_PASSWORD='<fixture password>'
export ODIGOS_DEST_core1772_CASSANDRA_CA_PEM="$(cat /path/to/fixture-ca.pem)"
/tmp/cassandra-odigoscol --config /tmp/cassandra-collector.json
```

In another terminal, from `collector/odigosotelcol`:

```sh
GOWORK=off go run ../../tests/e2e/cassandra/testdata/workload.go
```

Use an authenticated, certificate-verifying CQL client to query:

```sql
SELECT traceid, spanname, duration, resourceattributes['service.name'] AS service
FROM core1772_e2e.otel_spans;
SELECT timestamp, severitytext, body, resourceattributes['service.name'] AS service
FROM core1772_e2e.otel_logs;
```

The fixture's full-table reads/counts are for a small validation dataset, not production queries. Each workload run creates fresh telemetry IDs/timestamps and therefore adds rows.

## Exporter regression tests

From `collector/exporters/cassandraexporter`:

```sh
GOWORK=off go test ./...
CASSANDRA_TEST_PORT=19042 \
CASSANDRA_TEST_USERNAME='<fixture username>' \
CASSANDRA_TEST_PASSWORD='<fixture password>' \
CASSANDRA_TEST_CA=/path/to/fixture-ca.pem \
GOWORK=off go test -v ./...
```

The real-server tests create/drop uniquely named keyspaces. They cover automatic schema creation, multi-resource ingestion, logs without span IDs, long-span duration, idempotent batch replay, rejected writes, recovery, exporter-helper retries, wrong-password rejection and untrusted-certificate rejection. Set CASSANDRA_TEST_COMPRESSION to test LZ4Compressor, SnappyCompressor, DeflateCompressor or ZstdCompressor. All four were exercised against Cassandra 5.0.9.

The named destination rejects metrics/profiles. The alpha storage mapping's omitted fields, schema migration requirements, permissions, production replication, queries and retention are documented in the generated destination page's custom section.
