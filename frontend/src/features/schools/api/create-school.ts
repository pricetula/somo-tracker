import { apiFetch } from '@/lib/api/client';
import type { School, SchoolCreateRequest } from '../types';
import type { components } from '@/features/auth/api/generated/types';

type WireSchoolCreateRequest = components['schemas']['SchoolCreateRequest'];
type WireSchool = components['schemas']['School'];

function mapSchoolCreateRequestToWire(req: SchoolCreateRequest): WireSchoolCreateRequest {
  return {
    name: req.name,
    code: req.code,
    country_id: req.countryId,
    education_system_id: req.educationSystemId,
  };
}

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

export async function createSchool(req: SchoolCreateRequest): Promise<School> {
  const wireReq = mapSchoolCreateRequestToWire(req);
  const wireSchool = await apiFetch<WireSchool>('/api/v1/schools', {
    method: 'POST',
    body: JSON.stringify(wireReq),
  });

  return mapSchoolFromWire(wireSchool);
}
