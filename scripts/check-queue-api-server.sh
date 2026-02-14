#!/usr/bin/env bash
# Run this ON THE SERVER (e.g. in SSH) to see what queue-api and the gateway actually return.
# Usage: ./scripts/check-queue-api-server.sh

set -e
echo "=== 1. Direct queue-api (port 8081) ==="
echo "GET /health:"
curl -s -w "\nHTTP %{http_code}\n" http://localhost:8081/health || true
echo ""
echo "POST /v1/enqueue (no auth, expect 400 invalid json or 401 if auth enabled):"
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:8081/v1/enqueue -H "Content-Type: application/json" -d '{}' || true

echo ""
echo "=== 2. Via Traefik gateway (port 8090) ==="
echo "GET /queue-api/health:"
curl -s -w "\nHTTP %{http_code}\n" http://localhost:8090/queue-api/health || true
echo ""
echo "POST /queue-api/v1/enqueue (no auth):"
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:8090/queue-api/v1/enqueue -H "Content-Type: application/json" -d '{}' || true

echo ""
echo "=== 3. Queue-api container env (auth disabled if ENQUEUE_AUTH_TOKEN empty/unset) ==="
docker exec queue-api env 2>/dev/null | grep -E "ENQUEUE_AUTH|QUEUE_API" || echo "(container not found or no match)"
