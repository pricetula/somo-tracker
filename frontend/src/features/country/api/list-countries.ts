import { apiFetch } from '@/lib/api/client';
import type { Country } from '../types';

export async function listCountries(): Promise<Country[]> {
  return apiFetch<Country[]>('/reference/countries', { method: 'GET' });
}
