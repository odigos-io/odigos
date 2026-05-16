# Scheduler Controllers

> 36 nodes · cohesion 0.08

## Key Concepts

- **cache** (10 connections) — `extension/odigosconfigk8sextension/cache.go`
- **cache.go** (7 connections) — `extension/odigosconfigk8sextension/cache.go`
- **.logsReadLoop()** (7 connections) — `receivers/odigosebpfreceiver/logs.go`
- **.Delete()** (5 connections) — `extension/odigosconfigk8sextension/cache.go`
- **.Set()** (5 connections) — `extension/odigosconfigk8sextension/cache.go`
- **BufferReader** (5 connections) — `receivers/odigosebpfreceiver/buffer_reader.go`
- **.tracesReadLoop()** (5 connections) — `receivers/odigosebpfreceiver/traces.go`
- **logEventToPdata()** (5 connections) — `receivers/odigosebpfreceiver/logs.go`
- **logsAttrCache** (5 connections) — `receivers/odigosebpfreceiver/logs.go`
- **processorURLTemplateParsedRulesCache** (5 connections) — `processors/odigosurltemplateprocessor/cache.go`
- **.Get()** (4 connections) — `extension/odigosconfigk8sextension/cache.go`
- **logEvent** (4 connections) — `receivers/odigosebpfreceiver/logs.go`
- **keyPrefixFromKey()** (3 connections) — `extension/odigosconfigk8sextension/cache.go`
- **IsClosedError()** (3 connections) — `receivers/odigosebpfreceiver/buffer_reader.go`
- **NewBufferReader()** (3 connections) — `receivers/odigosebpfreceiver/buffer_reader.go`
- **.size()** (3 connections) — `receivers/odigosebpfreceiver/logs.go`
- **perfBufferReader** (3 connections) — `receivers/odigosebpfreceiver/buffer_reader.go`
- **ringBufferReader** (3 connections) — `receivers/odigosebpfreceiver/buffer_reader.go`
- **logs.go** (3 connections) — `receivers/odigosebpfreceiver/logs.go`
- **.clear()** (2 connections) — `extension/odigosconfigk8sextension/cache.go`
- **.getContainerKeysForWorkload()** (2 connections) — `extension/odigosconfigk8sextension/cache.go`
- **.commString()** (2 connections) — `receivers/odigosebpfreceiver/logs.go`
- **.logData()** (2 connections) — `receivers/odigosebpfreceiver/logs.go`
- **.streamString()** (2 connections) — `receivers/odigosebpfreceiver/logs.go`
- **.Close()** (2 connections) — `receivers/odigosebpfreceiver/buffer_reader.go`
- *... and 11 more nodes in this community*

## Relationships

- [[Collector Client gRPC Config]] (104 shared connections)
- [[Cypress E2E Tests]] (6 shared connections)
- [[Sampling Rule Types (GraphQL)]] (2 shared connections)
- [[CLI Uninstall & Logging]] (1 shared connections)
- [[Frontend Destination CRUD]] (1 shared connections)

## Source Files

- `extension/odigosconfigk8sextension/cache.go`
- `processors/odigosurltemplateprocessor/cache.go`
- `receivers/odigosebpfreceiver/buffer_reader.go`
- `receivers/odigosebpfreceiver/logs.go`
- `receivers/odigosebpfreceiver/traces.go`

## Audit Trail

- EXTRACTED: 99 (87%)
- INFERRED: 15 (13%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*