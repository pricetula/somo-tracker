import { apiFetch } from '@/lib/api/client';
import type { School } from '../types';
import type { components } from '@/features/auth/api/generated/types';

type WireSchool = components['schemas']['School'];

function mapSchoolFromWire(school: WireSchool): School {
  return {
    id: school.id!,
    tenantId: school.tenant_id!,
    name: school.name!,
    code: school.code!,
    countryId: school.country_id!,
    educationSystemId: school.education_system_id!,
    createdAt: school.created_at!,
    updatedAt: school.updated_at!,
    deletedAt: school.deleted_at ?? null,
  };
}

export interface GetSchoolsParams {
  limit?: number;
  cursor?: string | null;
  sort?: string;
  filters?: Record<string, unknown>;
}

export interface GetSchoolsResponse {
  items: School[];
  nextCursor: string | null;
}

export async function getSchools(params: GetSchoolsParams = {}): Promise<GetSchoolsResponse> {
  const queryParams = new URLSearchParams();
  if (params.limit) queryParams.set('limit', String(params.limit));
  if (params.cursor) queryParams.set('cursor', params.cursor);
  if (params.sort) queryParams.set('sort', params.sort);
  if (params.filters && Object.keys(params.filters).length > 0) {
    queryParams.set(
      'filters',
      btoa(JSON.stringify(params.filters)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
    );
  }

  const response = await apiFetch<{ items: WireSchool[]; nextCursor: string | null }>(
    `/api/v1/schools?${queryParams.toString()}`,
    { method: 'GET' }
  );

  return {
    items: response.items.map(mapSchoolFromWire),
    nextCursor: response.nextCursor ?? null,
  };
}
