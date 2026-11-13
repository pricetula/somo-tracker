/**
 * Server function to get current user profile
 * Runs on TanStack Start server, reads Cookie header and forwards to /api/v1/me
 * Returns MeResponse for root route loader (SSR, no flicker)
 */

import { createServerFn } from '@tanstack/react-start';
import { apiFetch } from '@/lib/api/client';
import type { MeResponse } from '../types';

export const getMe = createServerFn({ method: 'GET' }).handler(
  async (): Promise<MeResponse> => {
    const cookieHeader = (await import('@tanstack/react-start/server')).getRequestHeaders?.()
      ?.get('cookie');

    // Forward the cookie header so the backend can resolve the session
    const response = await apiFetch<MeResponse>('/me', {
      headers: cookieHeader ? { cookie: cookieHeader } : {},
    });

    return response;
  }
);
