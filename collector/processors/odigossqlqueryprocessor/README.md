# SQL Query Processor

The `odigossqlquery` processor enhances database spans by parsing SQL from `db.query.text` or the legacy `db.statement` attribute. It can:

1. **Infer attributes** — extract `db.operation.name` and `db.collection.name` when missing, and align the span name.
2. **Redact literals** — replace literal values in the query with `?` placeholders.

Parsing uses [DataDog/go-sqllexer](https://github.com/DataDog/go-sqllexer), with dialect selection based on `db.system.name` / `db.system`.

## Configuration

The processor resolves per-source options from an `OdigosConfigExtension` (typically `odigosconfigk8s`).
On each resource, it calls `GetFromResource` and applies the workload's collector config:

| Field on `ContainerCollectorConfig` | Effect |
| --- | --- |
| `inferDbAttributes` (non-nil) | Infer `db.operation.name` / `db.collection.name` and update the span name when attributes are added. |
| `dbQueryTemplatization.templatizeLiterals: true` | Replace literals in `db.query.text` / `db.statement` with `?` placeholders. |

Sources without those fields set are skipped. When both are enabled for a source, infer and redact run in a single pass (`ObfuscateAndNormalize`).

```yaml
processors:
  odigossqlquery:
    odigos_config_extension: odigosconfigk8s
```

| Option | Type | Description |
| --- | --- | --- |
| `odigos_config_extension` | component ID | Extension implementing `OdigosConfigExtension`. Required in Odigos-managed configs. |

In Odigos, the shared Processor CR is created when an enabled `DbQueryTemplatization` or `InferDbAttributes` Action exists. Per-source options are written into InstrumentationConfig / the extension cache by the instrumentor; the processor does not take static per-feature flags in that mode.

## Behavior

### Query source

The processor reads the query from, in order:

1. `db.query.text`
2. `db.statement`

The same attribute that was read is updated when literal redaction is enabled for the source.

### Dialect selection

`db.system.name` is preferred over `db.system`. Known SQL systems are mapped to a sqllexer dialect (`WithDBMS`):

| `db.system` / `db.system.name` | Dialect |
| --- | --- |
| `mssql`, `mssqlcompact`, `microsoft.sql_server`, `sql-server`, `sqlserver` | SQL Server |
| `postgresql`, `postgres` | PostgreSQL |
| `mysql`, `mariadb` | MySQL |
| `oracle`, `oracle.db` | Oracle |
| `snowflake` | Snowflake |

Unmapped systems use the default sqllexer behavior (no `WithDBMS`).

### Non-SQL systems

Spans whose `db.system` / `db.system.name` identifies a known non-SQL database are skipped entirely (no literal redaction, no attribute inference). Examples include MongoDB, Redis, Cassandra, DynamoDB, Elasticsearch/OpenSearch, CouchDB/Couchbase, Cosmos DB, HBase, Memcached, Neo4j, Geode, and InfluxDB.

### Attribute inference

When `inferDbAttributes` is set for the source and attributes are missing:

- `db.operation.name` is set only when exactly one SQL operation is detected (JOIN is ignored as a clause).
- `db.collection.name` is set only when exactly one table is detected.

Existing attributes are never overwritten.

### Span name

The span name is updated only when new attributes were added by this processor:

- `{operation} {collection}` when both are available
- `{operation}` when only the operation is available

The name is left unchanged if it already contains the operation (and collection, when present).

## Examples

**Input**

```
span name: db
db.query.text: SELECT * FROM users WHERE id = 1 AND name = 'alice'
db.system: postgresql
```

**With `inferDbAttributes` and `dbQueryTemplatization.templatizeLiterals: true` for the source**

```
span name: SELECT users
db.query.text: SELECT * FROM users WHERE id = ? AND name = ?
db.operation.name: SELECT
db.collection.name: users
db.system: postgresql
```
