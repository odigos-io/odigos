# Profiling backend – design and API for the UI

Use this in the **profiling UI** chat so the frontend can call the backend correctly. The backend is implemented in this repo; the UI is implemented in the other chat.

---

## Backend design (agreed)

- **No session IDs.** Up to **10 slots** keyed by **source** (namespace + kind + name). “Active” = slot exists and was requested in the last **10 minutes**.
- **Source key** = `namespace/kind/name` (e.g. `default/Deployment/frontend`). Same key is derived from profile resource attributes (k8s) and from the API path.
- **Storage:** Raw profile data only. Per-slot **20 MB** size cap; oldest chunks dropped when over. **Aggregation (flame graph) is done in the UI.**
- **OTLP profiles** received on a **separate gRPC port 4318** (feature-gated by `ENABLE_PROFILES_RECEIVER`, default true). Gateway must be configured to send profiles to the UI backend at `ui.<namespace>:4318`.
- **TTL:** Slots with no request for 10 minutes are removed by a background job. Opening “View profiling” or polling GET refreshes the slot.

---

## API contract

Base path for profiling: **`/api`** (routes are registered on `r.Group("/api")`).

All paths use **path parameters**: `namespace`, `kind`, `name`.  
`kind` must be the workload kind string used elsewhere in Odigos (e.g. **Deployment**, **StatefulSet**, **DaemonSet**, **Rollout**). Use the same `kind` as in the sources table / GraphQL (PascalCase).

---

### 1. Enable continuous profiling

**Request**

- **Method:** `PUT`
- **Path:** `/api/sources/:namespace/:kind/:name/profiling/enable`
- **Path params:** `namespace`, `kind`, `name` (e.g. `default`, `Deployment`, `frontend`)
- **Body:** none
- **Headers:** Same as rest of Odigos UI (e.g. CSRF if enabled).

**Response (success)**

- **Status:** `200 OK`
- **Body:** `{ "status": "ok", "sourceKey": "<namespace>/<kind>/<name>" }`

**Behavior**

- Ensures a slot exists for this source (creates or refreshes `lastRequestAt`). If there are already 10 slots, the slot with the oldest `lastRequestAt` is evicted.
- Call this when the user enables “continuous profiling” for a source, or when they open the “View profiling” screen (so the backend starts/keeps collecting for this source).

**Example**

```http
PUT /api/sources/default/Deployment/frontend/profiling/enable
```

---

### 2. Get profile data

**Request**

- **Method:** `GET`
- **Path:** `/api/sources/:namespace/:kind/:name/profiling`
- **Path params:** `namespace`, `kind`, `name`
- **Body:** none

**Response (success)**

- **Status:** `200 OK`
- **Body:** `{ "chunks": [ "<json string 1>", "<json string 2>", ... ] }`

Each element of `chunks` is a **JSON string**: one OTLP/JSON-serialized profile payload (one resource’s profiles). The UI should `JSON.parse` each string and then merge/aggregate the profile data to build the flame graph (e.g. stack samples → flame graph in the browser).

If no slot or no data: `{ "chunks": [] }`.

**Behavior**

- If the source has no slot yet, the backend creates/refreshes it (same as “enable”).
- Returns a **snapshot** of the current buffer for that source (up to 20 MB of recent data). No streaming in MVP; the UI can **poll** this endpoint every 5–10 seconds while the profiling view is open to refresh data and keep the slot alive.

**Example**

```http
GET /api/sources/default/Deployment/frontend/profiling
```

---

## UI flow (recommended)

1. **Sources page:** Add “Enable continuous profiling” and “View profiling” (or a single “View profiling” that implies enabling).
2. **On “View profiling” (or open profiling chart):**
   - Call **PUT** `/api/sources/{namespace}/{kind}/{name}/profiling/enable` once (so the backend has a slot).
   - Then call **GET** `/api/sources/{namespace}/{kind}/{name}/profiling` to fetch data.
3. **While the profiling view is open:**
   - Poll **GET** every 5–10 seconds to get new chunks and refresh the 10‑minute TTL.
   - Parse each `chunks[i]` as JSON (OTLP profile format), aggregate in the client, and render the flame graph.
4. **On close:** No need to call the backend; the slot will expire after 10 minutes without requests.

---

## Error responses

- **400 Bad Request:** Missing or invalid path params (e.g. empty `namespace`). Body: `{ "error": "missing namespace, kind, or name" }`.

---

## CORS / auth

- Same origin and auth as the rest of the Odigos UI (same backend). No extra CORS or auth for these endpoints.

---

## Summary for the other chat

- **Enable:** `PUT /api/sources/:namespace/:kind/:name/profiling/enable`
- **Get data:** `GET /api/sources/:namespace/:kind/:name/profiling` → `{ chunks: string[] }` (each string is OTLP/JSON; aggregate in UI for flame graph).
- **Source identity:** Use the source’s `namespace`, `kind`, and `name` from the sources table (same as GraphQL `sourceId` / workload id). `kind` is PascalCase (e.g. `Deployment`).
- **Polling:** Poll GET every 5–10 s while the profiling view is open to refresh data and TTL. SSE/streaming can be added later.
