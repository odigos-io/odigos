# SQL Query Processor

The `odigossqlquery` processor enhances database spans by parsing SQL from `db.query.text` or the legacy `db.statement` attribute. It can:

1. **Infer attributes** — extract `db.operation.name` and `db.collection.name` when missing, and align the span name.
2. **Redact literals** — replace literal values in the query with `?` placeholders.

Parsing uses [DataDog/go-sqllexer](https://github.com/DataDog/go-sqllexer), with dialect selection based on `db.system.name` / `db.system`.

## Configuration

```yaml
processors:
  odigossqlquery:
    infer_attributes: true
    redact_literals: true
```

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `infer_attributes` | bool | `false` | Infer operation/collection attributes from the query and update the span name when new attributes are added. |
| `redact_literals` | bool | `false` | Replace literals in `db.query.text` / `db.statement` with placeholders. |

When both are enabled, literal redaction and attribute inference run in a single pass (`ObfuscateAndNormalize`).

## Behavior

### Query source

The processor reads the query from, in order:

1. `db.query.text`
2. `db.statement`

When `redact_literals` is enabled, every present query attribute is redacted. Attribute inference
still uses the first one found in the order above.

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

When `infer_attributes` is enabled and attributes are missing:

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

**With `infer_attributes: true` and `redact_literals: true`**

```
span name: SELECT users
db.query.text: SELECT * FROM users WHERE id = ? AND name = ?
db.operation.name: SELECT
db.collection.name: users
db.system: postgresql
```
