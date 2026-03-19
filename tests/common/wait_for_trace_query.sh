#!/bin/bash

# Common script to poll trace DB until a query succeeds.
# Usage: wait_for_trace_query.sh <query_file> [extra_args...]
# Example: wait_for_trace_query.sh ../../common/queries/wait-for-trace.yaml
# Example: wait_for_trace_query.sh ../../common/queries/wait-for-trace-stream-2.yaml false traces-2 simple-trace-db-2

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
QUERY_FILE="$1"
shift

if [ -z "$QUERY_FILE" ]; then
    echo "Usage: wait_for_trace_query.sh <query_file> [extra_args...]"
    exit 1
fi

while true; do
    if "$SCRIPT_DIR/simple_trace_db_query_runner.sh" "$QUERY_FILE" "$@"; then
        break
    fi
    sleep 1
done
