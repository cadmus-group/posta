#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC="${ROOT_DIR}/api/openapi.yaml"
OUT="${ROOT_DIR}/clients/typescript-fetch"
IMAGE="openapitools/openapi-generator-cli:v7.9.0"

mkdir -p "${OUT}"

echo "Generating TypeScript fetch client from ${SPEC}..."
docker run --rm -v "${ROOT_DIR}:/local" "${IMAGE}" generate \
	-i /local/api/openapi.yaml \
	-g typescript-fetch \
	-o /local/clients/typescript-fetch \
	--additional-properties npmName=posta-queue-client,npmVersion=0.1.0,supportsES6=true \
	--skip-validate-spec

echo "done; output written to ${OUT}"

