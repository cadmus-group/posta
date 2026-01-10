export type EnqueueRequest = {
  type: string;
  queue?: string;
  payload?: Record<string, unknown>;
};

export type EnqueueResponse = {
  id: string;
  queue: string;
};

export type ErrorResponse = {
  code: string;
  message: string;
  requestId?: string;
};

type Options = {
  token?: string;
  edgeToken?: string;
  signal?: AbortSignal;
};

/**
 * Minimal fetch-based client for POST /v1/enqueue.
 */
export async function enqueue(
  baseUrl: string,
  req: EnqueueRequest,
  opts: Options = {},
): Promise<EnqueueResponse> {
  const url = `${baseUrl.replace(/\/$/, '')}/v1/enqueue`;
  const headers: Record<string, string> = {
    'content-type': 'application/json',
    accept: 'application/json',
  };

  if (opts.token) {
    headers.authorization = `Bearer ${opts.token}`;
  }
  if (opts.edgeToken) {
    headers['x-edge-token'] = opts.edgeToken;
  }

  const res = await fetch(url, {
    method: 'POST',
    headers,
    body: JSON.stringify(req),
    signal: opts.signal,
  });

  if (res.ok) {
    return res.json();
  }

  let data: ErrorResponse | undefined;
  try {
    data = (await res.json()) as ErrorResponse;
  } catch (_) {
    // ignore JSON parse errors
  }

  if (data?.message) {
    throw new Error(`enqueue failed (${res.status}): ${data.code ?? 'error'} ${data.message}`);
  }

  throw new Error(`enqueue failed (${res.status})`);
}

