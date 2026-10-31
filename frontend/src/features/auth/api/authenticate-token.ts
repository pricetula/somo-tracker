/**
 * Authenticate token API
 * POST /api/v1/auth/authenticate-token
 */

import { apiFetch } from '@/lib/api/client';
import type { AuthenticateTokenRequest, AuthenticateTokenResponse } from '../types';

export async function authenticateToken(
  token: string
): Promise<AuthenticateTokenResponse> {
  const body: AuthenticateTokenRequest = { token };
  return apiFetch<AuthenticateTokenResponse>('/auth/authenticate-token', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}