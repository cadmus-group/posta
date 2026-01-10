# Clients

Generated and handwritten clients derived from `api/openapi.yaml`.

## TypeScript (generated, typescript-fetch)

```bash
./scripts/gen-clients.sh
```

Output: `clients/typescript-fetch/` (not committed by default).

## TypeScript (minimal, handwritten)

```ts
import { enqueue } from "./ts/minimal";

await enqueue("https://queue-api.internal", {
  type: "example:ping",
  payload: { ok: true },
}, { token: process.env.QUEUE_API_TOKEN });
```

## Go

See `pkg/queueclient` for the reusable Go client and generated types under `pkg/queueclient/gen`.

