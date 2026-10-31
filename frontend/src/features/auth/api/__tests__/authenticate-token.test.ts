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
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => mockResponse,
    });

    const token = 'test-token-abc';
    const result = await authenticateToken(token);

    expect(global.fetch).toHaveBeenCalledTimes(1);
    const [url, init] = (global.fetch as any).mock.calls[0];
    expect(url).toBe('/api/v1/auth/authenticate-token');
    expect(init.method).toBe('POST');
    expect(init.credentials).toBe('include');
    expect(init.headers.get('Content-Type')).toBe('application/json');
    expect(JSON.parse(init.body)).toEqual({ token });

    expect(result).toEqual(mockResponse);
  });

  it('throws on non-2xx response', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({ code: 'INVALID_TOKEN', message: 'invalid' }),
    });

    await expect(authenticateToken('bad')).rejects.toThrow();
  });
});
