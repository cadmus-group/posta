#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILES=("-f" "$ROOT_DIR/deploy/docker-compose/dev.yml")

if [[ "${SMOKE_ENABLE_AUTH:-}" == "1" ]]; then
	COMPOSE_FILES+=("-f" "$ROOT_DIR/deploy/docker-compose/ci.yml")
	if [[ -z "${ENQUEUE_AUTH_TOKEN:-}" ]]; then
		export ENQUEUE_AUTH_TOKEN="smoke-auth-token"
	fi
fi

cleanup() {
	docker compose "${COMPOSE_FILES[@]}" down -v
}
trap cleanup EXIT

docker compose "${COMPOSE_FILES[@]}" up -d --build

BASE_URL="${SMOKE_BASE_URL:-http://localhost:8080}"

wait_for() {
	local url="$1"
	local name="$2"
	for i in {1..60}; do
		if curl -fsS "$url" >/dev/null; then
			echo "$name ok"
			return
		fi
		sleep 2
	done
	echo "$name did not become ready" >&2
	exit 1
}

wait_for "${BASE_URL}/health" "health"
wait_for "${BASE_URL}/ready" "ready"

AUTH_HEADERS=()
if [[ -n "${ENQUEUE_AUTH_TOKEN:-}" ]]; then
	AUTH_HEADERS+=("-H" "X-Edge-Token: ${ENQUEUE_AUTH_TOKEN}")
fi

enqueue_payload='{"type":"example:ping"}'
curl -fsS -X POST -H "Content-Type: application/json" "${AUTH_HEADERS[@]}" \
	-d "${enqueue_payload}" "${BASE_URL}/v1/enqueue" >/tmp/posta-enqueue.json

for i in {1..20}; do
	if docker compose "${COMPOSE_FILES[@]}" logs worker --since 30s | grep -q "handled ping"; then
		echo "worker processed ping"
		break
	fi
	if [[ $i -eq 20 ]]; then
		echo "worker did not process ping" >&2
		exit 1
	fi
	sleep 2
done

statuses="$(seq 1 80 | xargs -n1 -P16 -I{} curl -o /dev/null -w "%{http_code}\n" -s "${BASE_URL}/health")"
if ! echo "${statuses}" | grep -q "429"; then
	echo "rate limiting did not return 429 in burst test" >&2
	exit 1
fi

if [[ -n "${ENQUEUE_AUTH_TOKEN:-}" ]]; then
	status_without="$(curl -o /dev/null -w "%{http_code}" -s -X POST -H "Content-Type: application/json" \
		-d "${enqueue_payload}" "${BASE_URL}/v1/enqueue")"
	if [[ "${status_without}" -ne 401 ]]; then
		echo "expected 401 when token required and missing" >&2
		exit 1
	fi

	status_with="$(curl -o /dev/null -w "%{http_code}" -s -X POST -H "Content-Type: application/json" \
		"${AUTH_HEADERS[@]}" -d "${enqueue_payload}" "${BASE_URL}/v1/enqueue")"
	if [[ "${status_with}" -ne 200 ]]; then
		echo "expected 200 when token provided" >&2
		exit 1
	fi
fi

echo "smoke tests passed"

