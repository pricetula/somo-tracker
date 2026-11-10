/**
 * Update onboarding API
 * PUT /api/v1/onboarding/current
 */

import { apiFetch } from '@/lib/api/client';
import type { UpdateOnboardingRequest } from '../types';

export async function updateOnboarding(
  data: UpdateOnboardingRequest
): Promise<void> {
  return apiFetch<void>('/onboarding/current', {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}