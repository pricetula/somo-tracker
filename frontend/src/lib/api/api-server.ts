export async function getServerForwardHeaders(): Promise<Record<string, string>> {
  const mod = await import('@tanstack/react-start/server');
  const headers = mod.getRequestHeaders?.();
  if (!headers) return {};
  const out: Record<string, string> = {};
  const cookie = headers.get('cookie');
  const requestId = headers.get('x-request-id');
  if (cookie) out['Cookie'] = cookie;
  if (requestId) out['X-Request-ID'] = requestId;
  return out;
}
