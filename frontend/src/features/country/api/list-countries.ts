import { apiFetch } from '@/lib/api/client';
import type { Country } from '../types';

export async function listCountries(): Promise<Country[]> {
  return apiFetch<Country[]>('/api/v1/reference/countries', { method: 'GET' });
}
