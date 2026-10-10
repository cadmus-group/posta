#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC="${ROOT_DIR}/api/openapi.yaml"
OUT_DIR="${ROOT_DIR}/pkg/queueclient/gen"

if ! command -v oapi-codegen >/dev/null 2>&1; then
	echo "oapi-codegen not found; installing v2.x..." >&2
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.2.0
fi

mkdir -p "${OUT_DIR}"
oapi-codegen -generate types -o "${OUT_DIR}/types.gen.go" -package queueclientgen "${SPEC}"

echo "generated ${OUT_DIR}/types.gen.go from ${SPEC}"

