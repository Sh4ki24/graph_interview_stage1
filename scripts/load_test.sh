#!/usr/bin/env bash
# Sends N requests to the API and captures pprof profiles from the
# metrics server. Usage: make load_test  or  make load_test N=1000
set -euo pipefail

N=${1:-100}
BASE_URL=${BASE_URL:-http://localhost:8080}
PPROF_URL=${PPROF_URL:-http://localhost:9090}
CONCURRENCY=${CONCURRENCY:-10}
DURATION=${DURATION:-30}

echo "==> health check"
curl -sf "$BASE_URL/healthz" >/dev/null || { echo "API not reachable at $BASE_URL"; exit 1; }

mkdir -p pprof

echo "==> starting CPU profile capture (${DURATION}s, background)"
curl -sf --max-time $((DURATION + 10)) "$PPROF_URL/debug/pprof/profile?seconds=$DURATION" -o pprof/cpu.prof &
CPU_PID=$!

echo "==> sending $N requests (concurrency $CONCURRENCY)"
START=$(date +%s)
seq "$N" | xargs -P "$CONCURRENCY" -I{} bash -c '
  code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "'"$BASE_URL"'/tasks" \
    -H "Content-Type: application/json" -H "X-Role: user" \
    -d "{\"title\":\"load test {}\"}")
  echo "$code"
' | sort | uniq -c
END=$(date +%s)

echo "==> elapsed: $((END - START))s"

wait "$CPU_PID"
curl -sf "$PPROF_URL/debug/pprof/heap" -o pprof/heap.prof
curl -sf "$PPROF_URL/debug/pprof/goroutine" -o pprof/goroutine.prof

echo "==> pprof profiles written to ./pprof"
echo "    go tool pprof pprof/cpu.prof"
echo "    go tool pprof pprof/heap.prof"
