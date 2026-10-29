/**
 * Server function to get auth state from cookie
 * Runs on TanStack Start server, reads Cookie header
 * Returns { hasSession: boolean } for router context
 */

import { createServerFn } from '@tanstack/react-start';
import type { AuthState } from '../types';

export const getAuthState = createServerFn({ method: 'GET' }).handler(
  async (): Promise<AuthState> => {
    // In TanStack Start, we can access request headers via the context
    // The cookie is HttpOnly so we can't read it in client JS,
    // but on the server we can check the Cookie header
    const cookieHeader = (await import('@tanstack/react-start/server')).getRequestHeaders?.()
      ?.get('cookie');

    // Check for session cookie presence (name per ADR-0002: __Secure-session_token with env suffix)
    const hasSession = cookieHeader?.includes('__Secure-session_token') ?? false;

    return { hasSession };
  }
);