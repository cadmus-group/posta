# Contributing to Posta

Thank you for helping improve Posta. Contributions that make the existing HTTP enqueue API, task registration, OpenAPI aggregation, examples, documentation, or deployment safer and easier to use are welcome.

## Before opening a change

- Check existing issues and pull requests. Use the bug report or feature request template for new discussions.
- For non-trivial changes, describe the observed problem and proposed scope in an issue first.
- Never include secrets, live credentials, private infrastructure addresses, customer data, or sensitive job payloads in issues, logs, tests, or pull requests. Follow [SECURITY.md](SECURITY.md) for vulnerabilities.

## Local development

You will need Go 1.25 and Docker Compose v2 for integration tests.

```bash
git clone https://github.com/cadmus-group/posta.git
cd posta
cp .env.example .env
go test ./...
go vet ./...
docker network inspect posta-network >/dev/null 2>&1 || docker network create posta-network
./run.sh -c "up -d --build"
```

The default demonstration module is `generic` and logs payloads. Do not use it for real customer data. The development configuration intentionally uses non-production settings. Run the existing integration smoke test from the repository root with `./scripts/smoke.sh` after the network and local environment have been prepared. The script manages and tears down its Compose services/volumes: do not run it against a development stack containing data you need to keep.

To stop local services: `./run.sh -c "down"`.

## Repository orientation

| Location | Purpose |
| --- | --- |
| `cmd/` | Binary entry points |
| `internal/httpapi/` | Queue API HTTP transport |
| `internal/queue/` | Asynq client and task registry |
| `internal/modules/` | Registered task modules |
| `internal/openapiagg/` | OpenAPI source loading and merging |
| `gateway/traefik/` | Reverse proxy and middleware |
| `deploy/docker-compose/` | Development, CI, and server examples |
| `api/openapi.yaml` | Published API contract |
| `clients/`, `scripts/` | Client examples and generation |

The internal Go packages are implementation details, not a promised public Go library API. Changing the HTTP contract should include corresponding OpenAPI documentation.

## Pull requests

1. Fork or branch from `main`, keep each change focused, and explain its user-visible effect.
2. Run `go test ./...` and `go vet ./...`. For changes affecting deployment or the HTTP path, run the Compose smoke test and explain the result.
3. Add or update tests and documentation when changing behavior; provide reproducible examples for bug fixes.
4. Keep backward compatibility in mind, especially route paths, environment variables, image tags, task delivery semantics, and Compose assumptions.
5. Open a PR with a clear summary, verification steps, and any compatibility or security implications.

Do not change the MIT license or introduce additional dependencies, services, or features without a clearly justified issue. CI and release workflows are part of the compatibility surface: keep them operational.

## Coding conventions

Follow standard Go conventions (`gofmt`, useful package boundaries, context propagation, and explicit error handling). Keep handlers small, validate input before enqueueing, and make task execution safe to retry. Prefer neutral examples over organization-specific configuration.

By submitting a contribution, you agree that it can be distributed under this repository's MIT license. Please follow the [Code of Conduct](CODE_OF_CONDUCT.md).

