#!/bin/bash

# Common script to generate traffic and wait for job completion.
# Usage: generate_traffic_and_wait.sh [job_manifest]
# Default manifest: ../../common/apply/generate-traffic-job.yaml

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JOB_MANIFEST="${1:-$SCRIPT_DIR/apply/generate-traffic-job.yaml}"
TIMEOUT="${2:-60s}"

kubectl apply -f "$JOB_MANIFEST"

JOB_NAME=$(kubectl get -f "$JOB_MANIFEST" -o=jsonpath='{.metadata.name}')
kubectl wait --for=condition=complete "job/$JOB_NAME" --timeout="$TIMEOUT"

kubectl delete -f "$JOB_MANIFEST"
