/**
 * Education system combobox API
 * GET /api/v1/reference/education-systems?country_id=UUID
 */

import { apiFetch } from '@/lib/api/client';
import type { EducationSystem } from '../types';

export async function listEducationSystems(countryId: string): Promise<EducationSystem[]> {
  return apiFetch<EducationSystem[]>(`/reference/education-systems?country_id=${countryId}`, {
    method: 'GET',
  });
}
