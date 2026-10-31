import { describe, it, expect, vi, beforeEach } from 'vitest';
import { authenticateToken } from '../authenticate-token';

describe('authenticateToken', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    global.fetch = vi.fn();
  });

  it('POSTs token to /auth/authenticate-token with credentials include', async () => {
    const mockResponse = {
      status: 'authenticated',
      auth_state: 'authenticated',
    };
    const mockedFetch = vi.fn();
    mockedFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => mockResponse,
    } as Response);
    global.fetch = mockedFetch as typeof global.fetch;

    const token = 'test-token-abc';
    const result = await authenticateToken(token);

    expect(global.fetch).toHaveBeenCalledTimes(1);
    const mockCalls = (global.fetch as ReturnType<typeof vi.fn>).mock.calls;
    const [url, init] = mockCalls[0] as [string, RequestInit];
    expect(url).toBe('/api/v1/auth/authenticate-token');
    expect(init.method).toBe('POST');
    expect(init.credentials).toBe('include');
    const headers = init.headers as Headers;
    expect(headers.get('Content-Type')).toBe('application/json');
    expect(JSON.parse(init.body as string)).toEqual({ token });

    expect(result).toEqual(mockResponse);
  });

  it('throws on non-2xx response', async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({ code: 'INVALID_TOKEN', message: 'invalid' }),
    } as Response);

    await expect(authenticateToken('bad')).rejects.toThrow();
  });
});
