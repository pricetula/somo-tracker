/**
 * Logout API
 * POST /api/v1/auth/logout
 */

import { apiFetch } from '@/lib/api/client';
import type { LogoutRequest, LogoutResponse } from '../types';

export async function logout(): Promise<LogoutResponse> {
  const body: LogoutRequest = {};
  return apiFetch<LogoutResponse>('/auth/logout', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}