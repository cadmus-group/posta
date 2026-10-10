# API clients

Posta's HTTP contract is maintained in [`api/openapi.yaml`](../api/openapi.yaml). This repository includes a small handwritten TypeScript client and scripts that can generate additional types/clients locally.

## TypeScript: minimal fetch client

The client in [`clients/ts/minimal.ts`](ts/minimal.ts) calls `POST /v1/enqueue` relative to the supplied base URL.

```ts
import { enqueue } from "./ts/minimal";

await enqueue("http://localhost:8080/queue-api", {
  type: "generic",
  queue: "default",
  payload: { message: "hello" },
});
```

Use the `generic` module **only** for local demonstrations; it logs incoming payloads. In protected deployments, supply `{ token: "..." }` or `{ edgeToken: "..." }` in the third argument. The supplied gateway route is `/queue-api/v1/enqueue`; direct calls on the private container network can use the base URL `http://queue-api:8081`.

## Generate a TypeScript fetch client

```bash
./scripts/gen-clients.sh
```

Generated output goes to `clients/typescript-fetch/` and is not committed by default. The script requires Docker and runs OpenAPI Generator.

## Generate Go API types

```bash
./scripts/gen-openapi-go.sh
```

This generates Go **types**, not a reusable transport client, at `pkg/queueclient/gen/types.gen.go`. The `pkg/` directory is currently ignored by Git. Do not import that generated path from another repository as a stable published Go package.

Update the OpenAPI contract whenever you change the HTTP API, then regenerate downstream clients as needed.

