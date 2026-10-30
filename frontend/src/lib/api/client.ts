import { ApiError, codeFromStatus, parseRetryAfter } from './errors';

function apiBase(isServer: boolean): string {
  if (isServer) {
    // Server-side calls must use absolute URL
    return process.env.API_URL ?? process.env.API_INTERNAL_URL ?? 'http://localhost:8080/api/v1';
  }
  return '/api/v1';
}

async function getServerForwardHeaders(): Promise<Record<string, string>> {
  if (typeof window !== 'undefined') return {};
  try {
    const mod = await import('@tanstack/react-start/server');
    const headers = mod.getRequestHeaders?.();
    if (!headers) return {};
    const out: Record<string, string> = {};
    const cookie = headers.get('cookie');
    const requestId = headers.get('x-request-id');
    if (cookie) out['Cookie'] = cookie;
    if (requestId) out['X-Request-ID'] = requestId;
    return out;
  } catch {
    return {};
  }
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const isServer = typeof window === 'undefined';
  const headers = new Headers(init.headers);
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (isServer) {
    const forwarded = await getServerForwardHeaders();
    for (const [k, v] of Object.entries(forwarded)) {
      headers.set(k, v);
    }
  }

  let res: Response;
  try {
    res = await fetch(`${apiBase(isServer)}${path}`, { ...init, headers, credentials: 'include' });
  } catch (cause) {
    if (init.signal?.aborted) throw cause;
    throw new ApiError({ status: 0, code: 'NETWORK_ERROR', message: 'Network error', cause });
  }

  if (!res.ok) {
    let body: Partial<{ code: string; message: string; errors: Record<string, { code: string; message: string }[]>; request_id: string }> = {};
    try {
      body = await res.json();
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError({
      status: res.status,
      code: body.code ?? codeFromStatus(res.status),
      message: body.message ?? res.statusText,
      fieldErrors: (body.errors ?? {}) as Record<string, { code: string; message: string }[]>,
      requestId: body.request_id ?? res.headers.get('X-Request-ID') ?? undefined,
      retryAfterSeconds: parseRetryAfter(res.headers.get('Retry-After')),
    });
  }

  if (res.status === 204) return undefined as unknown as T;
  return (await res.json()) as T;
}
