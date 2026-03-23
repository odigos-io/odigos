# Profile data structure and symbol resolution in Odigos backend

## 1. How profiles are arranged in the collected sample (structured, not random)

### Incoming OTLP batch (gateway → UI backend)

- **One batch** = one `ProfilesData` message containing:
  - **One shared `ProfilesDictionary`** at the root (string_table, function_table, location_table, **stack_table**, mapping_table, attribute_table, link_table).
  - **Multiple `ResourceProfiles`** (e.g. 3 in gw-profiles-dump: odiglet, node-level, metrics-server).

So profiles are **structured by resource**: each ResourceProfile has its own resource attributes (k8s.namespace.name, k8s.deployment.name or k8s.daemonset.name, service.name, etc.) and its own ScopeProfiles → Profile → **Samples**. All samples in that batch refer to the **same** dictionary (same stack_table, location_table, string_table, etc.). Sample identity is (stack_index, attribute_indices, link_index); stack_index points into the shared dictionary.stack_table, which gives location_indices → locations → functions → string_table for symbol names.

### Per-resource grouping in the backend

- Backend derives a **source key** only when both exist:
  - `k8s.namespace.name`
  - one of: `k8s.deployment.name`, `k8s.daemonset.name`, `k8s.statefulset.name`, … or fallback `service.name`
- Key format: `"namespace/kind/name"` (e.g. `odigos-system/DaemonSet/odiglet`, `kube-system/Deployment/metrics-server`).
- **Node-level** resources (only `k8s.node.name`, no namespace/deployment/daemonset) do **not** get a source key and are **dropped**; they never get a slot or cache.

So in the cache, profiles are **always per service** (per source key). What gets stored is **one or more chunks per source key**, where each chunk is either:
- **Single-RP path**: one ResourceProfile for that key + a **copy of the batch’s dictionary**.
- **Merge path**: multiple ResourceProfiles for the same key merged into one Profiles + **same dictionary copy**; if `MergeTo` fails (e.g. attribute index errors), the merged payload can end up with **no usable dictionary**.

---

## 2. What is stored in the cache

- **Key**: source key string, e.g. `odigos-system/DaemonSet/odiglet`, `kube-system/Deployment/metrics-server`.
- **Value**: a **slot** (per source key) holding:
  - `LastRequestAt` (for TTL)
  - **BoundedBuffer** of **chunks**: each chunk is raw JSON bytes of `ProfilesData` (dictionary + one or more ResourceProfiles for that key).

Chunks are **append-only**; when total size exceeds `slotMaxBytes`, **oldest chunks are dropped**. So the “cache” is a rolling window of recent OTLP JSON chunks for that source.

Important: **symbols live inside each chunk’s dictionary**. If a chunk has no dictionary (e.g. merge failed) or an empty one, that chunk contributes samples but no symbol names for them.

---

## 3. How symbols are associated (and why they can be missing)

### Correct association (per chunk)

1. **Parse chunk** → root has `dictionary` (string_table, function_table, location_table, **stack_table**, mapping_table).
2. **stack_table** is parsed: `stack_index → location_indices` (root-first). (Implemented in `extractStackTable` + `getSampleLocIDs(..., stackTable)`.)
3. **Samples** use `stack_index` → resolve to **location_indices** from stack_table.
4. **Names** for those location indices come from the **same** chunk’s dictionary: location_table → lines → function_table → string_table (see `extractNamesFromDictionary`).
5. So for a given chunk, **symbol association is correct** when:
   - The chunk has a non-empty dictionary (including stack_table, location_table, function_table, string_table), and
   - The parser uses that dictionary for both stack resolution and name resolution.

### Why symbols are missing in the UI

1. **Merge path loses dictionary**  
   When multiple ResourceProfiles for the same key are merged with `single.MergeTo(merged)`, the collector’s merge can fail (e.g. “invalid attribute index 119”). On failure the code still stores the merged payload, which may have no or broken dictionary. Then that chunk has no usable names → frame_N or empty.

2. **refParsed fallback is unsafe across different batches**  
   The code picks the **first** chunk with a non-empty dictionary as `refParsed` and uses it to fill missing names for **all** chunks. But location indices are **per-chunk**: location index 10 in chunk A (batch 1) and location index 10 in chunk B (batch 2) can refer to different functions if the two batches have different dictionaries. So using refParsed from chunk A for chunk B can **wrongly** associate symbols. The fallback is only safe when all chunks share the **exact same** dictionary (e.g. same batch split by key).

3. **Node-level profiles never get a slot**  
   Node-level resources have no namespace/deployment/daemonset, so `SourceKeyFromResource` returns false and those profiles are dropped. They are never stored for “a specific service” and never appear in the per-service flame graph.

4. **First chunk(s) might have no dictionary**  
   If the first chunks in the buffer are merged chunks that lost the dictionary, `refParsed` stays nil and every chunk resolves names only from its own dictionary; any chunk with empty dictionary then gets only frame_N.

---

## 4. Recommendations: what to store and how to keep symbols correct

### A. Prefer single-RP path (avoid merge when possible)

- When there is **only one** ResourceProfile per key in a batch, the code uses `storeOne`: one chunk with **dictionary copied from the batch**. That chunk then has full dictionary and correct symbol association.
- When there are **multiple** RPs per key, the code merges and may lose the dictionary. So:
  - **Option 1**: Do **not** merge; store **one chunk per ResourceProfile** (each with a copy of the batch dictionary). That way every chunk carries its own dictionary and symbols stay correct. Downside: more chunks per key.
  - **Option 2**: Keep merge but **do not** store the result when `MergeTo` fails; instead store each RP as a separate chunk (same as Option 1 for that batch). That avoids storing chunks with no dictionary.

### B. Ensure at least one “reference” chunk per key has a full dictionary

- For a given source key, ensure the buffer **always** contains at least one chunk that has a non-empty dictionary (e.g. from the single-RP path or from a successful merge).
- When trimming the buffer (oldest chunks dropped), prefer **not** to drop the only chunk that has a dictionary; or periodically ensure a “fresh” chunk with dictionary is stored so refParsed can be chosen from a chunk that actually matches the current batches.

### C. Use refParsed only when dictionaries are known to align

- **Safe**: Use refParsed only for chunks that are known to share the same dictionary (e.g. same batch split by key). Then location indices align and refParsed names are correct.
- **Unsafe**: Using refParsed from an old batch for a new batch (different dictionary) can mis-associate names. So either:
  - Do not use refParsed across chunks from different batches; resolve names only from each chunk’s own dictionary, or
  - Store a “canonical” dictionary per source key (e.g. from the first successful chunk) and resolve names from it only when the chunk’s dictionary is empty **and** the chunk’s location/mapping tables match the canonical one (e.g. same binary). This is more involved and only needed if you must support chunks without their own dictionary.

### D. Showing profile data for a specific service

- **Already implemented**: Data is keyed by source key (`namespace/kind/name`). The UI requests profiling for a source (e.g. `default/Deployment/frontend`); the backend uses that as the key, creates/refreshes a slot, and stores incoming chunks for that key only. On “get profile”, it returns `GetProfileData(key)` → snapshot of chunks → `BuildPyroscopeProfileFromChunks` → one flame graph per service.
- **Gap**: Node-level profiles (no namespace/deployment) are dropped and cannot be shown as a “service”. If you need them, introduce a synthetic source key (e.g. `node/<node_name>`) and derive it when only `k8s.node.name` (and maybe `container.id`) are present.

### E. Summary checklist

| Goal | Action |
|------|--------|
| Symbols in flame graph | Ensure every stored chunk has a valid dictionary (avoid storing failed merges; prefer one chunk per RP when merge fails). |
| Correct symbol association | Resolve names from **the same chunk’s** dictionary for that chunk’s samples; use refParsed only when chunks share the same dictionary. |
| Per-service view | Keep keying by source key; only store and return chunks for that key. |
| Cache contents | Keep storing raw OTLP JSON chunks per key (dictionary + ResourceProfiles for that key); ensure at least one chunk per key has a full dictionary. |

---

## 5. Data flow (reference)

```
Gateway (OTLP) → Receiver (port 4318)
  → NewProfilesConsumer: for each batch (ProfilesData with 1 dictionary + N ResourceProfiles)
       → SourceKeyFromResource(attrs) → key (or drop if no key)
       → If key is active (slot exists): 
            - Single RP for key  → storeOne(pd, rps, i)  → newPd = dictionary + 1 RP → JSON → Buffer.Add
            - Multiple RPs for key → merge → merged.MergeTo() → if ok, JSON has dictionary; if fail, JSON may have no dict → Buffer.Add
  → Store: slots[key].Buffer = list of JSON chunks (each = dictionary + resource profiles for that key)

GET /api/.../profiling
  → store.GetProfileData(key) → chunks = snapshot of buffer
  → BuildPyroscopeProfileFromChunks(chunks)
       → First pass: refParsed = first chunk with ParsedChunkHasDictionary(parsed)
       → Second pass: for each chunk, parse → get samples (stack_index → stack_table → location_indices) → names from chunk dict (or refParsed) → tree.InsertStack(value, stack names)
  → TreeToFlamebearer → return Pyroscope-style JSON (names, levels, numTicks, …)
```

Symbols are correct when each chunk carries its own dictionary and the parser uses it for both stack resolution (stack_index → location_indices) and name resolution (location → function → string_table). Ensuring that every stored chunk has a valid dictionary (and avoiding cross-batch refParsed) keeps symbol association correct for that service’s flame graph.
