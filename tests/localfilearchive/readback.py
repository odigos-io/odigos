import argparse
import hashlib
import json
from collections import Counter
from pathlib import Path

SIGNALS = {
    "traces": ("resourceSpans", "scopeSpans", "spans"),
    "metrics": ("resourceMetrics", "scopeMetrics", "metrics"),
    "logs": ("resourceLogs", "scopeLogs", "logRecords"),
}


def read(directory, signal):
    resource_key, scope_key, item_key = SIGNALS[signal]
    records = []
    files = []
    batches = 0
    for path in sorted(directory.glob(f"{signal}*.json")):
        content = path.read_bytes()
        files.append({
            "name": path.name,
            "bytes": len(content),
            "sha256": hashlib.sha256(content).hexdigest(),
        })
        for line in content.splitlines():
            batch = json.loads(line)
            batches += 1
            for resource in batch.get(resource_key, []):
                for scope in resource.get(scope_key, []):
                    for item in scope.get(item_key, []):
                        records.append({
                            "resource": resource.get("resource", {}),
                            "resourceSchemaUrl": resource.get("schemaUrl", ""),
                            "scope": scope.get("scope", {}),
                            "scopeSchemaUrl": scope.get("schemaUrl", ""),
                            "record": item,
                        })
    return records, files, batches


def canonical(records):
    return Counter(json.dumps(record, sort_keys=True) for record in records)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Compare archived and replayed OTLP JSON records, resources, and scopes."
    )
    parser.add_argument("archive", type=Path)
    parser.add_argument("replayed", type=Path)
    args = parser.parse_args()
    results = {}
    for signal in SIGNALS:
        source, files, batches = read(args.archive, signal)
        replayed, _, _ = read(args.replayed, signal)
        results[signal] = {
            "records": len(source),
            "replayedRecords": len(replayed),
            "equal": bool(source) and canonical(source) == canonical(replayed),
            "files": files,
            "batches": batches,
            "resourceServices": sorted({
                attr["value"]["stringValue"]
                for record in source
                for attr in record["resource"].get("attributes", [])
                if attr["key"] == "service.name"
            }),
        }
    print(json.dumps(results, indent=2))
    raise SystemExit(0 if all(result["equal"] for result in results.values()) else 1)
