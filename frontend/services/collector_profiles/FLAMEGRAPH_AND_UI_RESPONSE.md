# Flame graph building and Pyroscope-like UI response

## How Pyroscope solves it

1. **Ingest (OTLP)**  
   - Receives OTLP `ExportProfilesServiceRequest` (ProfilesData: dictionary + ResourceProfiles).  
   - **ConvertOtelToGoogle**: OTLP `Profile` + `ProfilesDictionary` → Google pprof `Profile` (samples with `LocationId[]`, `Location`, `Function`, `StringTable`). Resolves stack_index → stack_table → location_indices → locations → functions → string table so each sample gets a list of symbol names.

2. **Storage**  
   - Converts to internal format and writes to **phlaredb** (blocks, time-range indexed). Not raw chunks; full time-series store.

3. **Query (Render API)**  
   - `GET /render?query=...&from=...&until=...&format=json`  
   - Backend runs **SelectMergeStacktraces** over the time range → returns a **merged flame graph** (proto: `Names`, `Levels`, `Total`, `MaxSelf`).  
   - **ExportToFlamebearer** converts that to **FlamebearerProfile** JSON.

4. **Flamebearer format (backend → frontend)**  
   - **FlamebearerProfile**: `version`, `flamebearer`, `metadata`, `timeline`, `groups`, `heatmap`.  
   - **flamebearer**:  
     - `names`: `[]string` — symbol names (index 0 is usually `"total"`).  
     - `levels`: `[][]int` — each level is one row of the flame graph; each node is **4 ints**: `[xOffsetDelta, total, self, nameIndex]`.  
     - `numTicks`: total samples (root width).  
     - `maxSelf`: max self value (for scaling).  
   - **Tree** (in-memory): `InsertStack(value, stack...)` merges stacks; **NewFlameGraph(tree, maxNodes)** walks the tree and produces levels + names with **delta-encoded x offsets** (each node’s x = previous x + previous width). Small nodes are folded into `"other"`.

5. **Frontend**  
   - Decodes Flamebearer (delta decoding of x offsets), converts to a DataFrame / node tree, and renders with `@grafana/flamegraph`. No OTLP parsing or tree building on the client.

---

## Odigos backend: what we do

### Ingest (simplified)

- **One chunk per ResourceProfile**: for each batch, for each ResourceProfile that has a source key and is active, we append **one chunk** = batch dictionary + that single RP. No merge. Chunks are appended in time order; buffer is a rolling window (trimmed by `slotMaxBytes`).

### GET /api/.../profiling → flame graph

1. **Snapshot chunks** for the source key from the store (`GetProfileData(key)`).
2. **Build flame graph** (`BuildPyroscopeProfileFromChunks`):
   - For each chunk: parse OTLP JSON (ParseOTLPChunk or ChunksFromPyroscopeOTLP) → get **samples** (stack = list of symbol names, value = count).
   - Merge all samples into a **Tree**: for each sample, `tree.InsertStack(value, stack...)`.
   - **TreeToFlamebearer(tree, 1024)** → `Flamebearer` (names, levels with 4-tuple and delta-encoded x, numTicks, maxSelf).
3. **Response**: Pyroscope-compatible **FlamebearerProfile**:
   - `version`: 1  
   - `flamebearer`: `{ names, levels, numTicks, maxSelf }`  
   - `metadata`: `{ format: "single", spyName, sampleRate, units, name: "cpu" }`  
   - `timeline`: optional `{ startTime, samples, durationDelta }`  
   - `groups`: null  
   - `heatmap`: null  

So the **UI receives the same shape as Pyroscope**: one JSON object per request. The frontend can decode `flamebearer.levels` (delta decoding) and render with any Flamebearer-compatible component (e.g. Pyroscope’s decode + `@grafana/flamegraph`).

---

## Response shape (for UI)

```json
{
  "version": 1,
  "flamebearer": {
    "names": ["total", "main.foo", "runtime.main", ...],
    "levels": [
      [0, 1000, 0, 0],
      [0, 800, 100, 1, 200, 700, 50, 2],
      ...
    ],
    "numTicks": 1000,
    "maxSelf": 100
  },
  "metadata": {
    "format": "single",
    "spyName": "",
    "sampleRate": 1000000000,
    "units": "samples",
    "name": "cpu"
  },
  "timeline": {
    "startTime": 1710000000,
    "samples": [0, 1000],
    "durationDelta": 15
  },
  "groups": null,
  "heatmap": null
}
```

- **levels**: each row is a level (depth). Each node = 4 ints: `[xOffsetDelta, total, self, nameIndex]`. The frontend **delta-decodes** the first value (x = prevX + prevDelta; first node x = 0). So the UI gets a compact, Pyroscope-identical encoding.
