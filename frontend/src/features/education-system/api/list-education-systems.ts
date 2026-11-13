import { apiFetch } from '@/lib/api/client';
import type { EducationSystem } from '../types';

export async function listEducationSystems(countryId: string): Promise<EducationSystem[]> {
  return apiFetch<EducationSystem[]>(`/reference/education-systems?country_id=${countryId}`, {
    method: 'GET',
  });
}
