#!/usr/bin/env bash
set -euo pipefail

TARGET_URL="${TARGET_URL:-http://127.0.0.1:8090}"
AUTH_TOKEN="${AUTH_TOKEN:-}"

echo "=========================================================="
echo " Running HydroGo k6 Performance Benchmark Suite"
echo " Target: ${TARGET_URL}"
echo "=========================================================="

if ! command -v k6 &> /dev/null; then
    echo "k6 command not found. Please install k6 (https://k6.io/docs/get-started/installation/)."
    exit 1
fi

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Running Scenario A: Cached Public Reads"
k6 run -e TARGET_URL="${TARGET_URL}" "${DIR}/cached_reads.js"

echo "==> Running Scenario B: Concurrent Authenticated Writes"
k6 run -e TARGET_URL="${TARGET_URL}" -e AUTH_TOKEN="${AUTH_TOKEN}" "${DIR}/concurrent_writes.js"

echo "==> Running Scenario C: SSE Connection Saturation"
k6 run -e TARGET_URL="${TARGET_URL}" "${DIR}/sse_saturation.js"

echo "All k6 benchmarks finished successfully!"
