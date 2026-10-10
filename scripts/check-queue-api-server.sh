#!/usr/bin/env bash
# Diagnose the public gateway path without printing credentials.
# Usage: ./scripts/check-queue-api-server.sh
# Optional: GATEWAY_URL=http://localhost:8080 DIRECT_URL=http://localhost:8081
set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
echo "== Gateway queue-api liveness =="
curl -sS -i "${GATEWAY_URL%/}/queue-api/health" || true
echo
echo "== Gateway enqueue validation (no credentials) =="
echo "Expect 400 if auth is disabled, 401 if auth is configured."
curl -sS -i -X POST "${GATEWAY_URL%/}/queue-api/v1/enqueue" \
  -H "Content-Type: application/json" -d '{}' || true
echo

if [[ -n "${DIRECT_URL:-}" ]]; then
  echo "== Direct queue-api liveness (requires an explicitly reachable service port) =="
  curl -sS -i "${DIRECT_URL%/}/health" || true
fi
