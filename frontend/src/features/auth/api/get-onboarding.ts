/**
 * Get onboarding current state API
 * GET /api/v1/onboarding/current
 */

import { apiFetch } from '@/lib/api/client';
import type { GetOnboardingResponse } from '../types';

export async function getOnboardingCurrent(): Promise<GetOnboardingResponse> {
  return apiFetch<GetOnboardingResponse>('/onboarding/current', {
    method: 'GET',
  });
}