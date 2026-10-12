# Odigos Cassandra exporter adaptation

Based on the Apache-2.0 upstream Cassandra exporter v0.159.0. Generated metadata is regenerated with the repository workflow; the collector consumes this module through the replacement in `collector/builder-config.yaml`.

Odigos changes initialize the keyspace before connecting to it, use verified TLS through `configtls`, return failed writes for bounded full-batch retries, widen span duration to bigint, store span timestamps as timestamp, key spans by trace/span ID and key logs by deterministic record hashes. Existing upstream tables require new names or an explicit operator-managed migration.

The upstream alpha mapping still omits span events/links and scope metadata. See the destination documentation for the exact storage, retention and query limits.

Run `GOWORK=off go test ./...`. To include real ingestion tests against a disposable Cassandra 5 server, set `CASSANDRA_TEST_PORT`; optionally set `CASSANDRA_TEST_USERNAME`, `CASSANDRA_TEST_PASSWORD` and `CASSANDRA_TEST_CA` (CA file path). The tests create and drop a uniquely named keyspace.
