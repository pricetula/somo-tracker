/**
 * Request magic link API
 * POST /api/v1/auth/request-magic-link
 */

import { apiFetch } from '@/lib/api/client';
import type { RequestMagicLinkRequest, RequestMagicLinkResponse } from '../types';

export async function requestMagicLink(
  email: string
): Promise<RequestMagicLinkResponse> {
  const body: RequestMagicLinkRequest = { email };
  return apiFetch<RequestMagicLinkResponse>('/auth/request-magic-link', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}