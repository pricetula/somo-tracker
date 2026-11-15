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

export async function getSchool(id: string): Promise<School> {
  const wireSchool = await apiFetch<WireSchool>(`/api/v1/schools/${id}`, {
    method: 'GET',
  });

  return mapSchoolFromWire(wireSchool);
}
