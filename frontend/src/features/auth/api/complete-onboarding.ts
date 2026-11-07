/**
 * Complete onboarding API
 * POST /api/v1/onboarding/complete
 */

import { apiFetch } from '@/lib/api/client';
import type { CompleteOnboardingRequest, CompleteOnboardingResponse } from '../types';

export async function completeOnboarding(
  data: CompleteOnboardingRequest
): Promise<CompleteOnboardingResponse> {
  return apiFetch<CompleteOnboardingResponse>('/onboarding/complete', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}