/**
 * Frontend-only test for getMe server function path.
 * No backend involved — mocks global.fetch and verifies the API path.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch } from '@/lib/api/client';

describe('apiFetch path for /me', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    global.fetch = vi.fn();
  });

  it('apiFetch prepends /api/v1 to /me path on client', async () => {
    const mockedFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({}),
    } as Response);
    global.fetch = mockedFetch;

    // apiBase returns '/api/v1' on client (typeof window !== 'undefined')
    await apiFetch('/me', {});

    expect(global.fetch).toHaveBeenCalledTimes(1);
    const [url] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [
      string,
      RequestInit?,
    ];
    expect(url).toBe('/api/v1/me');
  });

  it('apiFetch includes credentials:include', async () => {
    const mockedFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({}),
    } as Response);
    global.fetch = mockedFetch;

    await apiFetch('/me', {});

    const [, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [
      string,
      RequestInit?,
    ];
    expect(init?.credentials).toBe('include');
  });
});
