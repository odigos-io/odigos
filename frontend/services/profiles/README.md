# Profiles: from cached OTLP chunks to a Pyroscope-style flame graph

This package implements the **service-layer** pipeline: **protobuf chunks in memory** → decode → Pyroscope-style merge → **`FlamebearerProfile`** (JSON-tagged struct the UI can render, same overall shape as Grafana Pyroscope).

---

## What is implemented here

| Stage | Role |
|--------|------|
| **Ingest** | OTLP profiles receiver → `OdigosProfilesConsumer` → `pprofile.ProtoMarshaler.MarshalProfiles` → `ProfileStore.AddProfileData` (one chunk per resource export, protobuf wire). |
| **Store** | `ProfileStore` + `BoundedBuffer`: TTL, per-slot byte cap, **shallow snapshot** on read (callers treat chunk bytes as read-only). |
| **Read helpers** | `ChunksForSourceKey(store, sourceKey)` in `read.go` — preferred entry for “get chunks for this workload key.” |
| **Decode** | `otlpchunk.UnmarshalExportProfilesRequest` — single place that unmarshals stored bytes to `ExportProfilesServiceRequest`. |
| **Transform** | `BuildPyroscopeProfileFromChunks` in `profile_builder.go`; `flamegraph/` (OTLP → samples → tree → flamebearer); `chunk_time.go` (earliest time for timeline). |

**Not in this package:** GraphQL schema, resolvers, or HTTP routes. The process wires `ProfileStore` into the GraphQL `Resolver` (`frontend/graph/resolver.go`), but **exposing a query (or similar) that calls `ChunksForSourceKey` + `BuildPyroscopeProfileFromChunks` and returns data to the browser** is still **application code in `graph/`** when you add the UI contract.

---

## Input

**`[][]byte` — a list of chunks**

Each element is **one OTLP Profiles payload** encoded as **protobuf** (same wire as pdata `ProtoMarshaler.MarshalProfiles` / `ExportProfilesServiceRequest`): a **dictionary** (string tables, attribute tables, etc.) plus **resource profiles**, each holding **profiles** with **samples** (stack indices, values, timestamps). These bytes are what the OTLP consumer wrote into the in-memory store for that key—typically several exports over a short window. The store returns a **shallow snapshot** (shared chunk backings; **do not mutate** those `[]byte`).

At the type level the input is: **many serialized profile messages**, in order, for the same logical source.

---

## Transformation (chunk bytes → flame graph)

```
   OTLP protobuf chunks                stack samples              merged tree           UI-ready JSON
  ┌─────────────────┐                ┌──────────────┐           ┌────────────┐        ┌─────────────────┐
  │  chunk[]byte    │   decode +     │ each sample: │  merge    │ one tree   │ encode│ FlamebearerPro- │
  │  chunk[]byte    │ ──convert──►  │ stack +      │ ────────► │ of frames  │ ────► │ file (names,    │
  │  …              │                │ weight       │           │ + weights  │       │ levels, meta,   │
  └─────────────────┘                └──────────────┘           └────────────┘        │ timeline, …)    │
        │                                    │                         │                  └─────────────────┘
        │                                    │                         │
        └── proto.Unmarshal to OTLP          └── Pyroscope’s OTLP→   └── InsertStack   └── TreeToFlamebearer
            types; each profile                 pprof step yields        aggregates          builds the nested
            becomes a list of                   root-first frame         duplicate stacks    “levels” rows and
            `{ stack frame names, value }`      into one series          name list used by   attaches metadata
                                                                              the graph      and a simple timeline

   (parallel) all chunks scanned for the earliest profile start time → fills timeline start in the output
```

1. **Per chunk**  
   Unmarshal as OTLP `ExportProfilesServiceRequest` protobuf. Each profile is converted (via Grafana Pyroscope’s OTLP ingester logic) into **samples**: each sample is a **stack** (function names, root → leaf) and a **numeric weight** (e.g. sample count).

2. **Merge**  
   Every sample from every chunk is **inserted into one in-memory tree**: same stack path adds to the same nodes; weights accumulate.

3. **Encode**  
   **Flamebearer** fields (`names`, `levels`, …), **metadata**, **timeline** (earliest profile time across chunks).

4. **Output**  
   **`FlamebearerProfile`** (version, flamebearer, metadata, optional timeline, …).

**Entry point:** `BuildPyroscopeProfileFromChunks(chunks [][]byte)`. Lower-level pieces: `flamegraph/`, `chunk_time.go`, `otlpchunk/`.

---

## Output

**One `FlamebearerProfile` value** (Go struct, JSON-tagged for the client):

| Part | Role |
|------|------|
| **Flamebearer** | `names`, `levels`, `numTicks`, `maxSelf` — flame graph geometry |
| **Metadata** | Format, units, profile name, sample-rate hint; optional symbolization hint |
| **Timeline** | Start time and minimal sample series when there is data |
| **Version** | Wire format version for the JSON object |

---

## Remaining work outside this package (checklist)

- **GraphQL (or HTTP):** Add operations that resolve a workload/source key, call `ChunksForSourceKey(r.ProfileStore, key)` then `BuildPyroscopeProfileFromChunks`, and return the result (or JSON) to the frontend.
- **Product flows:** “Start viewing” / slot lifecycle (`StartViewing`, TTL) must align with whatever triggers the collector to send profiles for that key.

Everything above the store boundary in this list is **not** defined inside `services/profiles/`.
