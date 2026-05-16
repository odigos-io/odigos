# Frontend Config Schema

> 28 nodes · cohesion 0.08

## Key Concepts

- **ProfileStore** (16 connections) — `services/profiles/store.go`
- **BoundedBuffer** (6 connections) — `services/profiles/buffer.go`
- **.EnsureSlot()** (3 connections) — `services/profiles/store.go`
- **store.go** (3 connections) — `services/profiles/store.go`
- **.Add()** (2 connections) — `services/profiles/buffer.go`
- **.trimToMaxLocked()** (2 connections) — `services/profiles/buffer.go`
- **NewBoundedBuffer()** (2 connections) — `services/profiles/buffer.go`
- **.cleanupExpired()** (2 connections) — `services/profiles/store.go`
- **.evictOldestSlotLocked()** (2 connections) — `services/profiles/store.go`
- **.RunCleanup()** (2 connections) — `services/profiles/store.go`
- **NewProfileStore()** (2 connections) — `services/profiles/store.go`
- **profiling_store.go** (2 connections) — `services/common/profiling_store.go`
- **buffer.go** (2 connections) — `services/profiles/buffer.go`
- **ProfileMemoryStats** (1 connections) — `services/common/profiling_store.go`
- **.Clear()** (1 connections) — `services/profiles/buffer.go`
- **.Size()** (1 connections) — `services/profiles/buffer.go`
- **.Snapshot()** (1 connections) — `services/profiles/buffer.go`
- **.ActiveSlots()** (1 connections) — `services/profiles/store.go`
- **.AddProfileData()** (1 connections) — `services/profiles/store.go`
- **.ClearAllSlots()** (1 connections) — `services/profiles/store.go`
- **.ClearSlotBuffer()** (1 connections) — `services/profiles/store.go`
- **.GetProfileData()** (1 connections) — `services/profiles/store.go`
- **.IsActive()** (1 connections) — `services/profiles/store.go`
- **.MaxSlots()** (1 connections) — `services/profiles/store.go`
- **.MemoryStats()** (1 connections) — `services/profiles/store.go`
- *... and 3 more nodes in this community*

## Relationships

- [[CLI Endpoints Detection]] (60 shared connections)
- [[Frontend GraphQL Loaders]] (1 shared connections)

## Source Files

- `services/common/profiling_store.go`
- `services/profiles/buffer.go`
- `services/profiles/store.go`

## Audit Trail

- EXTRACTED: 58 (95%)
- INFERRED: 3 (5%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*