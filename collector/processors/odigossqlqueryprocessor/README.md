# SQL Query Processor

The `odigossqlquery` processor enhances database spans by parsing SQL from `db.query.text` or the legacy `db.statement` attribute. It can:

1. **Enhance attributes** — extract `db.operation.name` and `db.collection.name` when missing, and align the span name.
2. **Obfuscate** — replace literal values in the query with `?` placeholders.

Parsing uses [DataDog/go-sqllexer](https://github.com/DataDog/go-sqllexer), with dialect selection based on `db.system.name` / `db.system`.

## Configuration

```yaml
processors:
  odigossqlquery:
    enhance_attributes: true
    obfuscate: true
```

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `enhance_attributes` | bool | `false` | Extract operation/collection attributes and update the span name when new attributes are added. |
| `obfuscate` | bool | `false` | Replace literals in `db.query.text` / `db.statement` with placeholders. |

When both are enabled, obfuscation and attribute extraction run in a single pass (`ObfuscateAndNormalize`).

## Behavior

### Query source

The processor reads the query from, in order:

1. `db.query.text`
2. `db.statement`

The same attribute that was read is updated when obfuscation is enabled.

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

Spans whose `db.system` / `db.system.name` identifies a known non-SQL database are skipped entirely (no obfuscation, no attribute enhancement). Examples include MongoDB, Redis, Cassandra, DynamoDB, Elasticsearch/OpenSearch, CouchDB/Couchbase, Cosmos DB, HBase, Memcached, Neo4j, Geode, and InfluxDB.

### Attribute enhancement

When `enhance_attributes` is enabled and attributes are missing:

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

**With `enhance_attributes: true` and `obfuscate: true`**

```
span name: SELECT users
db.query.text: SELECT * FROM users WHERE id = ? AND name = ?
db.operation.name: SELECT
db.collection.name: users
db.system: postgresql
```
