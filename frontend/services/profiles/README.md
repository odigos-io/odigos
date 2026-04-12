# Flame graph pipeline

### **OTLP profile chunks**:
They are decoded into stack **samples** (ordered frames + weight).
Samples are **merged** along matching frame paths into one **tree**, then **flattened** into flame graph fields (name table, level rows, tick totals).
**Timeline** uses the earliest profile time among chunks (e.g. **t_min** seconds), independent of tick sums.

### **Symbols**:
frames **F1 → F2 → F3** (caller first);
per-export weights **n1**, **n2**, …;
merged leaf **n1 + n2** when paths match;
time anchor **t_min** (seconds).

### **Weight**:
The numeric value on a sample: how much that **stack** contributes in that export
(e.g. sample count). It drives bar width after merge.

**n1** — the weight from the **first** chunk on the example path (same idea as **n2**, **n3**, … for later exports).

**Ticks** — the **sum of merged weights** in the whole flame graph after aggregation (e.g. **n1** alone, or **n1 + n2** after two matching exports).
This is what the UI uses as overall “size,” distinct from **t_min**.

---

## Stages

### **Chunk**:
One serialized profile export: lookup tables plus samples.
Each sample points at a stack and carries a **weight** (**n1** in the first diagram step).

### **Tree nodes**:
Each sample is walked frame-by-frame (root first).
Every frame becomes a **node** on a path; the **leaf** records **self** equal to that sample’s weight on the first pass (**n1**).

### **Merged tree**:
Further chunks with the **same** stack reuse the same path; weights **add** at the leaf (**n1 + n2**) and propagate as **totals** up the spine.

### **Flame graph**:
The merged tree is **flattened**: deduplicated frame strings, **levels** of horizontal bars, and a **tick** total equal to the sum of merged weights.

---

## Chunk → tree nodes → merged tree → flame graph

```
  ┌─────────────────────┐
  │       CHUNK         │     dictionaries + samples (bytes)
  │  weight n1 on stack │ ──► one export, one path F1→F2→F3
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │     TREE NODES      │     walk top-down: each frame is a node;
  │   F1                │     leaf gets self += n1
  │    └ F2             │
  │       └ F3 (self n1)│
  └──────────┬──────────┘
             │
             │    another chunk, same path, weight n2
             ▼
  ┌─────────────────────┐
  │    MERGED TREE      │     same nodes; leaf self += n2
  │   F1                │     → leaf holds n1 + n2
  │    └ F2             │     → ancestors’ totals include n1 + n2
  │       └ F3 (n1+n2)  │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │    FLAME GRAPH      │     names[] · levels[][] (bars per depth)
  │  ticks = Σ weights  │     tick sum matches merged tree
  │  timeline ≈ t_min   │     (wall clock, not tick math)
  └─────────────────────┘
```
